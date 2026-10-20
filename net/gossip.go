// GossipSub (4.3): тема tx/<network_id> — распространение транзакций;
// заготовка темы checkpoint/<network_id> для этапа 5 (интерфейс подписи —
// core/threshold; сбор подписей не реализуется).
//
// Политика ретрансляции: дешёвая проверка до gossip (подпись, лёгкий PoW,
// размер, seq/prev в связке) — невалидное не пересылается; полный apply —
// в фоне (12.5). Дедупликация по txid (message ID = txid). Конфликты
// (эквивокация) сетью не отсекаются: доставляются обе ветки, выбор за
// финализацией (этап 5).
package net

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// txMessageID — message ID pubsub = txid (дедупликация по txid, spec/01 1.2).
func txMessageID(pmsg *pb.Message) string {
	h := sha256.Sum256(pmsg.Data)
	return string(h[:])
}

// CheckpointHandler — приём чекпоинтов (этап 5: проверка пороговой подписи
// через core/threshold; здесь — заготовка темы и интерфейс).
type CheckpointHandler interface {
	// HandleCheckpoint вызывается для каждого сообщения темы чекпоинтов.
	// Проверка подписи и применение — этап 5.
	HandleCheckpoint(cp *wire.Checkpoint)
}

// newPubSub — GossipSub без подписей pubsub (транзакции несут собственные
// подписи; спам: лёгкий PoW + комиссия уже в ядре, 12.5).
func (n *Node) newPubSub(ctx context.Context, h host.Host) (*pubsub.PubSub, error) {
	return pubsub.NewGossipSub(ctx, h,
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithMessageIdFn(txMessageID),
		pubsub.WithMaxMessageSize(n.Cfg.MaxTxSize+4096),
	)
}

// startGossip — тема транзакций + заготовка темы чекпоинтов.
func (n *Node) startGossip(ctx context.Context) error {
	ps, err := n.newPubSub(ctx, n.Host)
	if err != nil {
		return err
	}
	n.PS = ps

	// Валидатор темы: дешёвая проверка ДО ретрансляции.
	if err := ps.RegisterTopicValidator(n.txTopic(),
		func(ctx context.Context, from peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
			if len(msg.Data) > n.Cfg.MaxTxSize {
				n.countSpam()
				return pubsub.ValidationReject
			}
			tx, err := wire.DecodeTransaction(msg.Data)
			if err != nil {
				n.countSpam()
				return pubsub.ValidationReject
			}
			if err := n.cheapCheck(tx); err != nil {
				n.countSpam()
				n.logf("спам от %s: %v", from.ShortString(), err)
				return pubsub.ValidationReject
			}
			msg.ValidatorData = tx
			return pubsub.ValidationAccept
		}); err != nil {
		return err
	}

	topic, err := ps.Join(n.txTopic())
	if err != nil {
		return err
	}
	n.TxTopic = topic
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}
	n.txSub = sub
	go n.gossipLoop(ctx)

	// Заготовка темы чекпоинтов (этап 5): приём без проверки подписи.
	cpTopic, err := ps.Join(n.cpTopic())
	if err != nil {
		return err
	}
	n.CpTopic = cpTopic
	cpSub, err := cpTopic.Subscribe()
	if err != nil {
		return err
	}
	go n.checkpointLoop(ctx, cpSub)
	return nil
}

// gossipLoop — приём транзакций из темы.
func (n *Node) gossipLoop(ctx context.Context) {
	for {
		msg, err := n.txSub.Next(ctx)
		if err != nil {
			n.logf("gossipLoop: выход: %v", err)
			return
		}
		tx, ok := msg.ValidatorData.(*wire.Transaction)
		if !ok {
			// сообщение прошло валидатор до его регистрации — декодируем
			tx, err = wire.DecodeTransaction(msg.Data)
			if err != nil || n.cheapCheck(tx) != nil {
				continue
			}
		}
		n.handleTxFromPeer(tx, msg.ReceivedFrom)
	}
}

// checkpointLoop — приём сообщений темы чекпоинтов (этап 5): предложение
// чекпоинта + доли подписей; сбор порога N-из-M, финализация.
func (n *Node) checkpointLoop(ctx context.Context, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			return
		}
		if len(msg.Data) > 1<<20 {
			continue
		}
		msgType, d, err := readFrame(msg.Data)
		if err != nil || msgType != msgCpShare {
			continue
		}
		raw, err := d.blob()
		if err != nil {
			continue
		}
		cp, err := decodeCheckpoint(raw)
		if err != nil {
			continue
		}
		signer, err := d.blob()
		if err != nil {
			continue
		}
		sig, err := d.blob()
		if err != nil {
			continue
		}
		share := wire.SignatureShare{Signer: signer, Signature: sig}
		n.onCpMessage(cp, share, false)
		n.collectCpPending(ctx)
		if n.Checkpoints != nil {
			n.Checkpoints.HandleCheckpoint(cp)
		}
	}
}

// decodeCheckpoint — разбор чекпоинта (wire.Checkpoint, spec/07 7.1;
// кодирование — wire, разбор здесь: core/wire не редактируется).
func decodeCheckpoint(b []byte) (*wire.Checkpoint, error) {
	d := &dec{b: b}
	cp := &wire.Checkpoint{}
	for d.pos < len(d.b) {
		f, wt, err := d.tag()
		if err != nil {
			return nil, err
		}
		switch {
		case f == 1 && wt == 2:
			if cp.NetworkID, err = d.blob(); err != nil {
				return nil, err
			}
		case f == 2 && wt == 0:
			if cp.Epoch, err = d.u64(); err != nil {
				return nil, err
			}
		case f == 3 && wt == 2:
			if cp.StateRoot, err = d.blob(); err != nil {
				return nil, err
			}
		case f == 4 && wt == 2:
			if cp.PrevCheckpointHash, err = d.blob(); err != nil {
				return nil, err
			}
		case f == 5 && wt == 0:
			if cp.NetworkTime, err = d.u64(); err != nil {
				return nil, err
			}
		case f == 6 && wt == 0:
			v, err := d.u64()
			if err != nil {
				return nil, err
			}
			cp.ProtocolVersion = uint32(v)
		case f == 7 && wt == 0:
			if cp.ActiveWallets, err = d.u64(); err != nil {
				return nil, err
			}
		case f == 8 && wt == 0:
			if cp.TotalSupply, err = d.u64(); err != nil {
				return nil, err
			}
		case f == 9 && wt == 2:
			p, err := d.blob()
			if err != nil {
				return nil, err
			}
			cp.Participants = append(cp.Participants, p)
		case f == 10 && wt == 0:
			if cp.Threshold, err = d.u64(); err != nil {
				return nil, err
			}
		case f == 11 && wt == 2:
			raw, err := d.blob()
			if err != nil {
				return nil, err
			}
			sig, err := decodeThresholdSig(raw)
			if err != nil {
				return nil, err
			}
			cp.ThresholdSig = sig
		case f == 12 && wt == 2:
			id, err := d.blob()
			if err != nil {
				return nil, err
			}
			cp.FinalizedTxIDs = append(cp.FinalizedTxIDs, id)
		default:
			return nil, errFrame
		}
	}
	return cp, nil
}

// PublishTx — публикация транзакции в тему.
func (n *Node) PublishTx(ctx context.Context, tx *wire.Transaction) error {
	if n.TxTopic == nil {
		return errors.New("net: gossip не запущен")
	}
	return n.TxTopic.Publish(ctx, tx.Encode())
}

// hex12 — первые 12 hex-символов txid (для журнала).
func hex12(tx *wire.Transaction) string {
	id := tx.TxID()
	return hex.EncodeToString(id[:])[:12]
}

// cheapCheck — дешёвая проверка до ретрансляции (4.3): подпись, лёгкий PoW,
// размер (размер — в валидаторе), seq/prev в связке. Полный apply — в фоне.
func (n *Node) cheapCheck(tx *wire.Transaction) error {
	if len(tx.NetworkID) == 0 || string(tx.NetworkID) != string(n.networkID) {
		return errors.New("чужая сеть")
	}
	if tx.ProtocolVersion != n.Cfg.Genesis.Genesis.ProtocolVersion {
		return errors.New("чужая версия протокола")
	}
	if tx.Pow == nil {
		return errors.New("нет PoW")
	}
	// C10/C11 (дешёвый вариант): алгоритм соответствует типу.
	hard := tx.Type == wire.Register || tx.Type == wire.Mint
	if hard && tx.Pow.Algorithm != wire.PowArgon2id {
		return errors.New("ожидался усиленный PoW")
	}
	if !hard && tx.Pow.Algorithm != wire.PowLight {
		return errors.New("ожидался лёгкий PoW")
	}
	pre := tx.SigningPreimage()
	if hard {
		if !crypto.VerifyHard(pre, tx.Pow.Nonce, n.pow.TargetHard, crypto.Argon2Params{
			MemoryKiB: n.pow.Argon2MemoryKiB, Iterations: n.pow.Argon2Iterations,
			Parallelism: n.pow.Argon2Parallelism,
		}) {
			return errors.New("усиленный PoW не сходится")
		}
	} else {
		if !crypto.VerifyLight(pre, tx.Pow.Nonce, n.pow.TargetLight) {
			return errors.New("лёгкий PoW не сходится")
		}
	}
	// Подпись: для register — ключ из payload; для остальных — ключ
	// кошелька из локального состояния (нет кошелька — пропускаем,
	// полный apply в фоне отклонит при невозможности).
	//
	// Чтение полей кошелька (PublicKey/Seq/TipTxID) — под applyMu.RLock:
	// валидатор gossip и handleTxFromPeer идут без applyMu, а фоновый
	// applyLive пишет реестр параллельно (race-аудит этапа 9, -race).
	// PoW (Argon2id) — до локом: проверка не читает реестр.
	n.applyMu.RLock()
	if tx.Type == wire.Register {
		reg, ok := tx.Payload.(*wire.RegisterPayload)
		if !ok {
			n.applyMu.RUnlock()
			return errors.New("register без payload")
		}
		pk, err := crypto.ParsePubKey(reg.AttestedPublicKey)
		if err != nil {
			n.applyMu.RUnlock()
			return errors.New("некорректный ключ")
		}
		sh := tx.SigningHash()
		if !crypto.VerifyDER(pk, sh[:], tx.Signature) {
			n.applyMu.RUnlock()
			return errors.New("подпись не сходится")
		}
	} else if w := n.wallet(tx.Wallet); w != nil && w.PublicKey != nil {
		sh := tx.SigningHash()
		if !crypto.VerifyDER(w.PublicKey, sh[:], tx.Signature) {
			n.applyMu.RUnlock()
			return errors.New("подпись не сходится")
		}
	}
	// seq/prev в связке: при seq = tip+1 prev обязан равняться tip;
	// конфликтная ветка (seq = tip) доставляется (эквивокация — этап 5).
	if w := n.wallet(tx.Wallet); w != nil && tx.Seq == w.Seq+1 {
		if string(tx.Prev) != string(w.TipTxID) {
			n.applyMu.RUnlock()
			return errors.New("prev не сходится с tip")
		}
	}
	n.applyMu.RUnlock()
	return nil
}
