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
	"fmt"
	"time"

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
// подписи; спам: лёгкий PoW + комиссия уже в ядре, 12.5). Peer score (A3,
// тема net-dos-a3): счёт пира в pubsub — время в mesh, первые доставки,
// невалидные сообщения (ValidationReject валидатора) + счёт приложения
// (dosTracker) — низкий счёт вырезает пира из mesh (пониженный
// приоритет/бандвит) и грейзлистит по порогу.
func (n *Node) newPubSub(ctx context.Context, h host.Host) (*pubsub.PubSub, error) {
	scoreParams, thresholds := n.peerScoreParams()
	return pubsub.NewGossipSub(ctx, h,
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithMessageIdFn(txMessageID),
		pubsub.WithMaxMessageSize(n.Cfg.MaxTxSize+4096),
		pubsub.WithPeerScore(scoreParams, thresholds),
	)
}

// startGossip — тема транзакций + заготовка темы чекпоинтов.
func (n *Node) startGossip(ctx context.Context) error {
	ps, err := n.newPubSub(ctx, n.Host)
	if err != nil {
		return err
	}
	n.PS = ps

	// Валидатор темы: полный путь приёма до ретрансляции (A3):
	// дешёвые структурные проверки → подпись → бюджет дорогих
	// операций → Argon2id/PoW.
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
			if err := n.relayAdmit(tx, from); err != nil {
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
	//
	// Валидатор темы (C1 аудита Sonnet-2, шаг 4): дешёвые структурные
	// проверки сообщения до ретрансляции — сеть, длины корней/адресов,
	// собственный порог чекпоинта, подписант из состава. Семантика
	// (состав по режиму манифеста, цепочка, подписи) — в onCpMessage и
	// collectCpPending: валидатор реестр не читает.
	if err := ps.RegisterTopicValidator(n.cpTopic(),
		func(ctx context.Context, from peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
			m, err := n.cpAdmit(msg.Data)
			if err != nil {
				n.countSpam()
				n.logf("чекпоинт от %s: %v", from.ShortString(), err)
				return pubsub.ValidationReject
			}
			msg.ValidatorData = m
			return pubsub.ValidationAccept
		}); err != nil {
		return err
	}
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

// cpMsg — разобранное сообщение темы чекпоинтов (валидатор → цикл приёма).
type cpMsg struct {
	cp    *wire.Checkpoint
	share wire.SignatureShare
}

// cpAdmit — допуск сообщения темы чекпоинтов (C1, шаг 4): структура
// кадра, своя сеть, длины полей, собственный порог чекпоинта и
// принадлежность подписанта составу. Порог 0 и порог больше числа
// участников отвергаются здесь же — такое предложение не может стать
// финализированным ни в одном режиме (cons.Verify, modes.go), и
// ретранслировать его незачем.
func (n *Node) cpAdmit(raw []byte) (*cpMsg, error) {
	if len(raw) > 1<<20 {
		return nil, fmt.Errorf("сообщение %d байт > предела", len(raw))
	}
	msgType, d, err := readFrame(raw)
	if err != nil {
		return nil, err
	}
	if msgType != msgCpShare {
		return nil, fmt.Errorf("тип сообщения %d, ожидался %d", msgType, msgCpShare)
	}
	rawCp, err := d.blob()
	if err != nil {
		return nil, err
	}
	cp, err := decodeCheckpoint(rawCp)
	if err != nil {
		return nil, err
	}
	signer, err := d.blob()
	if err != nil {
		return nil, err
	}
	sig, err := d.blob()
	if err != nil {
		return nil, err
	}
	if len(cp.NetworkID) == 0 || string(cp.NetworkID) != string(n.networkID) {
		return nil, errForeignNet
	}
	if cp.Epoch == 0 {
		return nil, fmt.Errorf("эпоха 0")
	}
	if len(cp.StateRoot) != 32 || len(cp.PrevCheckpointHash) != 32 {
		return nil, fmt.Errorf("корень/prev_hash не 32 байта")
	}
	// состав ограничен размером сообщения (1 МиБ ≈ 30 тыс. адресов);
	// верхняя граница — защита от разбора «на всякий случай»
	if len(cp.Participants) == 0 || len(cp.Participants) > maxCheckpointParticipants {
		return nil, fmt.Errorf("состав подписантов: %d", len(cp.Participants))
	}
	for _, p := range cp.Participants {
		if len(p) != 32 {
			return nil, fmt.Errorf("участник не 32 байта")
		}
	}
	if cp.Threshold == 0 {
		return nil, fmt.Errorf("порог 0")
	}
	if cp.Threshold > uint64(len(cp.Participants)) {
		return nil, fmt.Errorf("порог %d > участников %d", cp.Threshold, len(cp.Participants))
	}
	for _, id := range cp.FinalizedTxIDs {
		if len(id) != 32 {
			return nil, fmt.Errorf("finalized_txid не 32 байта")
		}
	}
	// доля подписи: либо пустая (предложение/перепубликация), либо
	// подписант из состава с непустой подписью
	if len(signer) > 0 {
		if len(signer) != 32 {
			return nil, fmt.Errorf("подписант не 32 байта")
		}
		if len(sig) == 0 || len(sig) > 128 {
			return nil, fmt.Errorf("подпись %d байт", len(sig))
		}
		inSet := false
		for _, p := range cp.Participants {
			if string(p) == string(signer) {
				inSet = true
				break
			}
		}
		if !inSet {
			return nil, fmt.Errorf("подписант вне состава")
		}
	}
	return &cpMsg{cp: cp, share: wire.SignatureShare{Signer: signer, Signature: sig}}, nil
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
			// сообщение прошло до регистрации валидатора — полный
			// путь приёма выполняется здесь (relayAdmit в
			// handleTxFromPeer)
			tx, err = wire.DecodeTransaction(msg.Data)
			if err != nil {
				continue
			}
			n.handleTxFromPeer(tx, msg.ReceivedFrom, false)
			continue
		}
		n.handleTxFromPeer(tx, msg.ReceivedFrom, true)
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
		m, ok := msg.ValidatorData.(*cpMsg)
		if !ok || m == nil || m.cp == nil {
			// сообщение прошло до регистрации валидатора — полный
			// путь допуска выполняется здесь
			m, err = n.cpAdmit(msg.Data)
			if err != nil {
				continue
			}
		}
		n.onCpMessage(m.cp, m.share, false)
		n.collectCpPending(ctx)
		if n.Checkpoints != nil {
			n.Checkpoints.HandleCheckpoint(m.cp)
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
			// счётный предел до накопления: заявленный счётчик
			// участников не задаёт объём памяти (FINDING-1 D10)
			if len(cp.Participants) >= maxCheckpointParticipants {
				return nil, fmt.Errorf("net: слишком много участников чекпоинта (лимит %d)",
					maxCheckpointParticipants)
			}
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
			// счётный предел до накопления (FINDING-1 D10): до него
			// finalized_txid_list ограничивал только кадр
			if len(cp.FinalizedTxIDs) >= maxCheckpointFinalizedTxIDs {
				return nil, fmt.Errorf("net: слишком много finalized_txid (лимит %d)",
					maxCheckpointFinalizedTxIDs)
			}
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

// c9Admit — допуск C9 (9.2, v9.13 — слой допуска сети): tx с claim вне
// ±timestamp_tolerance_seconds часов узла не принимается и не
// распространяется. Ядро и replay C9 не применяют — вердикт полностью
// детерминирован от txNow = claim; допуск отсекает только приём/
// ретрансляцию (понятная диагностика). Sandbox/тестовые сети — допуск
// управляем: проверка не применяется (короткие period_seconds и
// предсборка tx прогонов не ломаются).
func (n *Node) c9Admit(tx *wire.Transaction) error {
	prof := n.Cfg.Genesis.Manifest.Profile
	if prof.Sandbox || n.Clock == nil {
		return nil
	}
	tol := prof.TimestampToleranceSeconds
	now := n.Clock.NetTime()
	if tx.TimestampClaim+tol < now || now+tol < tx.TimestampClaim {
		return fmt.Errorf("timestamp_claim %d вне допуска C9 (±%d с от часов узла %d)",
			tx.TimestampClaim, tol, now)
	}
	return nil
}

// Ошибки пути приёма (A3): классы для штрафа счёта пира; тексты —
// прежние (существующие коды/сообщения, новых кодов нет).
var (
	errForeignNet      = errors.New("чужая сеть")
	errForeignVersion  = errors.New("чужая версия протокола")
	errNoPow           = errors.New("нет PoW")
	errWrongPowAlgo    = errors.New("ожидался усиленный PoW")
	errWrongPowAlgoL   = errors.New("ожидался лёгкий PoW")
	errBadSignature    = errors.New("подпись не сходится")
	errBadRegisterKey  = errors.New("некорректный ключ")
	errRegisterPayload = errors.New("register без payload")
	errPrevMismatch    = errors.New("prev не сходится с tip")
	errUnknownWallet   = errors.New("кошелёк не известен узлу")
	errPowHard         = errors.New("усиленный PoW не сходится")
	errPowLight        = errors.New("лёгкий PoW не сходится")
	errDosBudget       = errors.New("бюджет дорогих операций на пира исчерпан")
)

// cheapCheck — дешёвая проверка до ретрансляции (4.3; A3 — порядок
// «дешёвое → подпись», дорогой Argon2id/PoW — только после подписи,
// в relayAdmit/verifyPoW): сеть, версия протокола, наличие/тип PoW
// (структурно), допуск C9, seq/prev в связке, подпись ECDSA P-256
// (каноничный DER, low-s — этап 9). Полный apply — в фоне.
//
// Полный узел отклоняет tx незарегистрированного кошелька дёшево
// (errUnknownWallet): ядро в тот же момент дало бы E_UNKNOWN_WALLET
// (C5) — множество принимаемых tx не меняется, а Argon2id для
// неизвестного отправителя не выполняется вовсе (A3: ключа нет —
// подпись проверить нельзя, значит и дорогая работа не запускается;
// догоняющая доставка такой tx — сверкой, журнал упорядочен).
// Лёгкий узел (без реестра) неизвестность проверить не может — для
// него путь прежний: подпись пропускается, PoW — под бюджетом пира
// (relayAdmit).
//
// Чтение полей кошелька (PublicKey/Seq/TipTxID) — под applyMu.RLock:
// валидатор gossip и handleTxFromPeer идут без applyMu, а фоновый
// applyLive пишет реестр параллельно (race-аудит этапа 9, -race).
func (n *Node) cheapCheck(tx *wire.Transaction) error {
	if len(tx.NetworkID) == 0 || string(tx.NetworkID) != string(n.networkID) {
		return errForeignNet
	}
	if tx.ProtocolVersion != n.Cfg.Genesis.Genesis.ProtocolVersion {
		return errForeignVersion
	}
	if tx.Pow == nil {
		return errNoPow
	}
	// C9 — допуск сети (v9.13): claim вне допуска часов узла — не
	// принимается и не ретранслируется
	if err := n.c9Admit(tx); err != nil {
		return err
	}
	// C10/C11 (структурно, дёшево): алгоритм соответствует типу.
	hard := tx.Type == wire.Register || tx.Type == wire.Mint
	if hard && tx.Pow.Algorithm != wire.PowArgon2id {
		return errWrongPowAlgo
	}
	if !hard && tx.Pow.Algorithm != wire.PowLight {
		return errWrongPowAlgoL
	}
	// Подпись до дорогой работы (A3): для register — ключ из payload;
	// для остальных — ключ кошелька из локального состояния. Полный
	// узел без кошелька отклоняет дёшево (см. комментарий выше);
	// лёгкий узел — пропускает (нет состояния), дорогой PoW для него
	// ограничен бюджетом пира.
	n.applyMu.RLock()
	if n.Ledger != nil && tx.Type != wire.Register && n.wallet(tx.Wallet) == nil {
		n.applyMu.RUnlock()
		return errUnknownWallet
	}
	if tx.Type == wire.Register {
		reg, ok := tx.Payload.(*wire.RegisterPayload)
		if !ok {
			n.applyMu.RUnlock()
			return errRegisterPayload
		}
		pk, err := crypto.ParsePubKey(reg.AttestedPublicKey)
		if err != nil {
			n.applyMu.RUnlock()
			return errBadRegisterKey
		}
		sh := tx.SigningHash()
		if !crypto.VerifyDER(pk, sh[:], tx.Signature) {
			n.applyMu.RUnlock()
			return errBadSignature
		}
	} else if w := n.wallet(tx.Wallet); w != nil && w.PublicKey != nil {
		sh := tx.SigningHash()
		if !crypto.VerifyDER(w.PublicKey, sh[:], tx.Signature) {
			n.applyMu.RUnlock()
			return errBadSignature
		}
	}
	// seq/prev в связке: при seq = tip+1 prev обязан равняться tip;
	// конфликтная ветка (seq = tip) доставляется (эквивокация — этап 5).
	if w := n.wallet(tx.Wallet); w != nil && tx.Seq == w.Seq+1 {
		if string(tx.Prev) != string(w.TipTxID) {
			n.applyMu.RUnlock()
			return errPrevMismatch
		}
	}
	n.applyMu.RUnlock()
	return nil
}

// verifyPoW — дорогая часть пути приёма (A3): выполняется только
// после дешёвых проверок и подписи (relayAdmit). Реестр не читает —
// вне applyMu.
func (n *Node) verifyPoW(tx *wire.Transaction) error {
	pre := tx.SigningPreimage()
	hard := tx.Type == wire.Register || tx.Type == wire.Mint
	if hard {
		n.expensiveChecks.Add(1)
		if !crypto.VerifyHard(pre, tx.Pow.Nonce, n.pow.TargetHard, crypto.Argon2Params{
			MemoryKiB: n.pow.Argon2MemoryKiB, Iterations: n.pow.Argon2Iterations,
			Parallelism: n.pow.Argon2Parallelism,
		}) {
			return errPowHard
		}
	} else {
		if !crypto.VerifyLight(pre, tx.Pow.Nonce, n.pow.TargetLight) {
			return errPowLight
		}
	}
	return nil
}

// relayAdmit — единый путь приёма tx от пира (A3): дешёвые структурные
// проверки → подпись → бюджет дорогих операций пира → Argon2id/PoW.
// Ошибки — существующие классы (дешёвый отказ), новых кодов нет;
// события — в счёт пира (штраф/награда), исчерпание бюджета — счёт
// падает, по порогу dosDisconnectScore соединение разрывается
// (крайняя мера). Вызывается валидатором темы и handleTxFromPeer
// (сообщения до регистрации валидатора).
func (n *Node) relayAdmit(tx *wire.Transaction, from peer.ID) error {
	now := time.Now()
	if err := n.cheapCheck(tx); err != nil {
		switch {
		case errors.Is(err, errBadSignature), errors.Is(err, errBadRegisterKey):
			n.dos.penalty(from, dosPenaltyInvalidSig, now)
		case errors.Is(err, errUnknownWallet):
			n.dos.penalty(from, dosPenaltyUnknownWallet, now)
		default:
			n.dos.penalty(from, dosPenaltyMalformed, now)
		}
		n.maybeDisconnect(from)
		return err
	}
	hard := tx.Type == wire.Register || tx.Type == wire.Mint
	if hard {
		// бюджет дорогих операций на пира (A3): окно исчерпано —
		// дорогая проверка не выполняется, сообщение дёшево
		// отклоняется (догоняющая доставка — сверкой)
		if !n.dos.chargeExpensive(from, now) {
			n.dos.penalty(from, dosPenaltyBudget, now)
			n.budgetRejected.Add(1)
			n.maybeDisconnect(from)
			return errDosBudget
		}
		// параллельные Argon2id — лимит на узел (память, A3)
		select {
		case n.hardSem <- struct{}{}:
			defer func() { <-n.hardSem }()
		default:
			// лимит занят: не копим очередь дорогих проверок —
			// дёшево отклоняем (бюджет уже списан у пира)
			n.budgetRejected.Add(1)
			return errDosBudget
		}
	}
	if err := n.verifyPoW(tx); err != nil {
		n.dos.penalty(from, dosPenaltyInvalidPow, now)
		n.maybeDisconnect(from)
		return err
	}
	n.dos.reward(from, dosRewardValid, now)
	return nil
}

// maybeDisconnect — крайняя мера по порогу счёта (A3): разрыв
// соединения с пиром, чей счёт упал ниже dosDisconnectScore.
func (n *Node) maybeDisconnect(from peer.ID) {
	if n.dos.score(from) < dosDisconnectScore && n.Host != nil {
		n.logf("dos: разрыв соединения с %s (счёт %.1f)", from.ShortString(), n.dos.score(from))
		_ = n.Host.Network().ClosePeer(from)
	}
}
