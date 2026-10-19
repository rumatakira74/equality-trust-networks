// tnp-sandbox-peer — host-узел песочницы для эмулятора (этап 7).
// Назначение: полный путь участника в эмуляторе (критерий этапа 7)
// требует K поручительств от active-кошельков — в песочнице это
// посевной круг V1..V3 (детерминированные ключи, та же деривация,
// что в cmd/tnp-node). Узел:
//
//   - использует ТОТ ЖЕ genesis, что приложение (tnp.SandboxGenesis
//     с фиксированным genesis-временем — идентичный manifest_hash);
//   - слушает на фиксированном порту (эмулятор подключается к хосту
//     через 10.0.2.2);
//   - выполняет выпуск периодов 1–2 для V1..V3 (условия поручителя:
//     возраст ≥ 90 дн., ≥ 2 выполненных периодов, 7.4);
//   - следит за реестром и поручается (vouch) за каждого нового
//     зарегистрированного кандидата от V1, V2, V3 — активация при
//     K = 3.
//
// Только песочница/эмулятор; реальные сети — этап 8.
package main

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	tnp "github.com/rumatakira74/equality-trust-networks/mobile"
)

// Те же параметры, что FlutterBridge.ensureGenesis: идентичный genesis.
const (
	sandboxNetworkID  = "tnp-sandbox-0001"
	sandboxGenesisT   = int64(1_760_000_000) // фикс: одинаковый у приложения и узла
	sandboxEpochSec   = 3600
	sandboxPowLight   = 0x0010_0000_0000_0000
	sandboxPowHard    = 0x0010_0000_0000_0000
	sandboxArgon2MiB  = 8
	sandboxArgon2Iter = 1
	sandboxArgon2Par  = 1
)

// loggerFunc — адаптер функции журнала к tnp.Logger.
type loggerFunc func(string)

func (f loggerFunc) Log(line string) { f(line) }

func main() {
	port := flag.Int("port", 4001, "UDP-порт QUIC-транспорта")
	db := flag.String("db", "", "путь к SQLite (пусто — без персистентности)")
	mint := flag.Bool("mint", true, "выполнить периоды 1–2 для посевного круга")
	flag.Parse()

	log := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "sandbox-peer: "+f+"\n", a...) }

	gen := tnp.SandboxGenesis(sandboxNetworkID, sandboxGenesisT, sandboxEpochSec,
		sandboxPowLight, sandboxPowHard, sandboxArgon2MiB, sandboxArgon2Iter, sandboxArgon2Par)
	listen := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1,/ip4/0.0.0.0/tcp/%d", *port, *port)
	// детерминированный ключ узла: стабильный peer ID — приложение
	// подключается по полному multiaddr с /p2p/<id> (без ID адрес
	// молча отбрасывается AddrInfoFromP2pAddr)
	node, err := tnp.StartNodeAddrsKey(gen, "full", *db, nil, "",
		listen, "tnp-sandbox-peer-node-key", loggerFunc(func(l string) { log("узел: %s", l) }))
	if err != nil {
		log("запуск: %v", err)
		os.Exit(1)
	}
	defer node.Close()
	log("слушает udp/tcp %d; genesis-время %d", *port, sandboxGenesisT)
	log("peer: %s", node.StatsText())

	// ключи посевного круга (деривация как в cmd/tnp-node)
	seeds := map[string][]byte{}
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := crypto.DeterministicKey("tnp-node-seed:" + id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		seeds[id] = crypto.DeriveAddress(fp, []byte(sandboxNetworkID))
	}

	if *mint {
		// выпуск периодов 1 и 2 для V1..V3 (V3: ≥ 2 периодов, 7.4)
		for _, id := range []string{"V1", "V2", "V3"} {
			if err := mintFor(node, gen, id, seeds[id], 1); err != nil {
				log("mint %s период 1: %v", id, err)
			}
			if err := mintFor(node, gen, id, seeds[id], 2); err != nil {
				log("mint %s период 2: %v", id, err)
			}
		}
	}

	// наблюдение за реестром: поручиться за каждого нового кандидата
	// от V1, V2, V3 (активация при K = 3)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	vouched := map[string]bool{}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	log("слежу за реестром (vouch за новичков от V1..V3)…")
	for {
		select {
		case <-stop:
			log("остановка")
			return
		case <-ticker.C:
			// диагностика этапа 7: видно, подключается ли узел приложения
			log("статус: %s", node.StatsText())
			for _, cand := range registeredWallets(node) {
				if vouched[cand] {
					continue
				}
				vouched[cand] = true
				log("новый кандидат %s — поручительства от V1..V3", cand[:12])
				for _, id := range []string{"V1", "V2", "V3"} {
					if err := vouchFor(node, gen, id, seeds[id], mustHex(cand)); err != nil {
						log("vouch %s: %v", id, err)
					}
				}
			}
		}
	}
}

// registeredWallets — адреса кошельков в статусе registered (кандидаты).
func registeredWallets(node *tnp.Node) []string {
	var rows []struct {
		Address string `json:"address"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal([]byte(node.RegistryJSON()), &rows); err != nil {
		return nil
	}
	var out []string
	for _, r := range rows {
		if r.Status == "REGISTERED" {
			out = append(out, r.Address)
		}
	}
	return out
}

// mintFor — выпуск завершённого периода для посевного кошелька.
// amount = эмиссия × diff (M6); для первого выпуска base =
// activated_period − 1 = −1 (uint64) — как в net-тестах, период 1
// даёт diff 2 (2000 единиц).
func mintFor(node *tnp.Node, gen *tnp.Genesis, id string, addr []byte, period uint64) error {
	seq, prev, err := tipOf(node, addr)
	if err != nil {
		return err
	}
	amount := uint64(1000)
	if seq == 0 && period == 1 {
		amount = 2000 // diff = period − (activated_period − 1) = 2
	}
	tx := tnp.NewMint(gen.NetworkID(), addr, int64(seq+1), prev,
		int64(period), int64(amount), gen.ProtocolVersion(), time.Now().Unix())
	tx.SolveHard(gen.PowTargetHard(), gen.Argon2MemKiB(), gen.Argon2Iterations(), gen.Argon2Par())
	if err := sign(tx, id); err != nil {
		return err
	}
	res, err := node.Submit(tx)
	if err != nil {
		return err
	}
	if res.Outcome() != "accept" {
		return fmt.Errorf("отклонён: %s/%s", res.Outcome(), res.Code())
	}
	return nil
}

// vouchFor — поручительство посевного кошелька за кандидата (7.4).
func vouchFor(node *tnp.Node, gen *tnp.Genesis, id string, addr, candidate []byte) error {
	seq, prev, err := tipOf(node, addr)
	if err != nil {
		return err
	}
	var clock struct {
		Epoch uint64 `json:"epoch"`
	}
	if err := json.Unmarshal([]byte(node.ClockJSON()), &clock); err != nil {
		return err
	}
	tx := tnp.NewVouch(gen.NetworkID(), addr, int64(seq+1), prev,
		candidate, int64(clock.Epoch), gen.ProtocolVersion(), time.Now().Unix())
	tx.SolveLight(gen.PowTargetLight())
	if err := sign(tx, id); err != nil {
		return err
	}
	res, err := node.Submit(tx)
	if err != nil {
		return err
	}
	if res.Outcome() != "accept" {
		return fmt.Errorf("отклонён: %s/%s", res.Outcome(), res.Code())
	}
	return nil
}

// tipOf — seq/prev цепочки кошелька по снапшоту моста.
func tipOf(node *tnp.Node, addr []byte) (uint64, []byte, error) {
	var info struct {
		Seq int64  `json:"seq"`
		Tip string `json:"tip"`
	}
	if err := json.Unmarshal([]byte(node.WalletInfoJSON(addr)), &info); err != nil {
		return 0, nil, err
	}
	if info.Tip == "" {
		return uint64(info.Seq), []byte{}, nil
	}
	prev, err := hex.DecodeString(info.Tip)
	if err != nil {
		return 0, nil, err
	}
	return uint64(info.Seq), prev, nil
}

// sign — подпись детерминированным ключом посевного круга.
func sign(tx *tnp.Tx, id string) error {
	priv := crypto.DeterministicKey("tnp-node-seed:" + id)
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}
	sig, err := tnp.SignWithKey(der, tx.SigningHash())
	if err != nil {
		return err
	}
	tx.SetSignature(sig)
	return nil
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
