// Регресс-тест фикса времени (v9.13, этап 15): cons.Replay strict —
// живой журнал с паттерном «сборка в периоде N, публикация в N+1»
// (находка прогона 12-testnet, FINDING-replay-live-divergence) даёт
// state_root живых узлов: live и replay применяют каждую tx от
// txNow = timestamp_claim одним механизмом, отдельного режима часов нет.
// Strict — без послаблений: повреждённый журнал (tx, отклоняемая
// ядром) даёт ошибку с диагностикой — какая tx, заявленный период,
// период валидации (пинирован на claim), вердикт.
package cons

import (
	"crypto/ecdsa"
	"strings"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// replayEnv — короткие периоды (120 с, genesis-поле 61) и genesis в
// прошлом: граница периода достижима за минуты, как в прогоне 12-testnet.
type replayEnv struct {
	t       *testing.T
	gp      *wire.GenesisPackage
	keys    map[string]*ecdsa.PrivateKey
	addrs   map[string][]byte
	genesis uint64
	ps      uint64
}

func newReplayEnv(t *testing.T) *replayEnv {
	now := uint64(1767000000)
	genesis := now - 3600 // час назад: текущий период 30
	gp := params.TestGenesis()
	gp.Genesis.GenesisTime = genesis
	gp.Manifest.Profile.PeriodSeconds = 120 // поле 61
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.MintChallengeWindowHours = 0 // окно выключено (не объект теста)
	gp.Manifest.Profile.DemurrageRateMicros = 0      // демередж не объект теста
	// quota_reset (профиль 8, как сеть A прогона 12-testnet): квота
	// заявляется ровно одним периодом (diff = 1, M5)
	gp.Manifest.Profile.IssuanceMode = wire.IssuanceQuotaReset
	gp.Manifest.Profile.AccrualWindowPeriods = 0
	e := &replayEnv{t: t, gp: gp, keys: map[string]*ecdsa.PrivateKey{},
		addrs: map[string][]byte{}, genesis: genesis, ps: 120}
	var circle []wire.SeedMember
	for _, id := range []string{"S1", "S2", "S3"} {
		priv := crypto.DeterministicKey("replay:" + id)
		e.keys[id] = priv
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		e.addrs[id] = crypto.AddressOfKey(pub, gp.Genesis.NetworkID)
		circle = append(circle, wire.SeedMember{PublicKey: pub})
	}
	gp.Manifest.SeedCircle = circle
	wire.SetManifestHash(gp)
	return e
}

func (e *replayEnv) newLedger() *ledger.Ledger {
	l, err := ledger.New(e.gp, ledger.Options{SkipMintChallengeWindow: true})
	if err != nil {
		e.t.Fatalf("genesis: %v", err)
	}
	return l
}

// mintTx — выпуск завершённого периода посевным кошельком: claim в
// периоде p+1 (паттерн «сборка до границы, публикация после» — заявка
// периода p валидна от claim: p < p+1, M4). seq/prev — по желанию
// (для повреждённой ветки журнала).
func (e *replayEnv) mintTx(id string, p, claim uint64, seq uint64, prev []byte) *wire.Transaction {
	l := e.newLedger()
	w := l.State.Wallet(e.addrs[id])
	if seq == 0 {
		seq = w.Seq + 1
	}
	if prev == nil {
		prev = append([]byte(nil), w.TipTxID...)
	}
	tx := &wire.Transaction{
		NetworkID:       e.gp.Genesis.NetworkID,
		Wallet:          e.addrs[id],
		Seq:             seq,
		Prev:            prev,
		Type:            wire.Mint,
		Payload:         &wire.MintPayload{Period: p, Amount: 1000},
		ProtocolVersion: e.gp.Genesis.ProtocolVersion,
		TimestampClaim:  claim,
	}
	e.solve(tx, id, true)
	return tx
}

func (e *replayEnv) solve(tx *wire.Transaction, id string, hard bool) {
	net := wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	if hard {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
		ap := crypto.Argon2Params{MemoryKiB: net.Argon2MemoryKiB,
			Iterations: net.Argon2Iterations, Parallelism: net.Argon2Parallelism}
		for n := uint64(0); ; n++ {
			nb := make([]byte, 8)
			nb[0] = byte(n >> 56)
			nb[1] = byte(n >> 48)
			nb[2] = byte(n >> 40)
			nb[3] = byte(n >> 32)
			nb[4] = byte(n >> 24)
			nb[5] = byte(n >> 16)
			nb[6] = byte(n >> 8)
			nb[7] = byte(n)
			tx.Pow.Nonce = nb
			if crypto.VerifyHard(tx.SigningPreimage(), nb, net.TargetHard, ap) {
				break
			}
		}
	} else {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
		for n := uint64(0); ; n++ {
			nb := make([]byte, 8)
			nb[7] = byte(n)
			nb[6] = byte(n >> 8)
			tx.Pow.Nonce = nb
			if crypto.VerifyLight(tx.SigningPreimage(), nb, net.TargetLight) {
				break
			}
		}
	}
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(e.keys[id], sh[:])
	if err != nil {
		e.t.Fatalf("sign: %v", err)
	}
	tx.Signature = sig
}

// TestReplayStrictReproducesLiveRoot — закрытие находки 12-testnet:
// mint, собранный в периоде N (claim в N) и опубликованный в N+1,
// валидируется от claim одинаково live и в replay. Живой узел применяет
// журнал; свежий узел пересчитывает его cons.Replay-ем (strict) —
// state_root совпадает, расхождение находки закрыто по построению.
func TestReplayStrictReproducesLiveRoot(t *testing.T) {
	e := newReplayEnv(t)
	// журнал: квота периода 1 каждого посевного, claim — в периоде 2
	// (сборка в периоде 1, публикация после границы — паттерн прогона;
	// claim в прошлом относительно часов узла — вердикт от claim)
	claimIn := func(p uint64) uint64 { return e.genesis + p*e.ps + e.ps/2 }
	live := e.newLedger()
	var journal []*wire.Transaction
	for _, id := range []string{"S1", "S2", "S3"} {
		tx := e.mintTx(id, 1, claimIn(2), 0, nil)
		if res := live.Apply(tx); res.Outcome != ledger.Accepted {
			t.Fatalf("live mint %s периода 1: %v/%s", id, res.Outcome, res.Code)
		}
		journal = append(journal, tx)
	}
	liveRoot := StateRoot(live)

	// свежий узел: пересчёт из genesis + журнала (strict, без режима часов)
	fresh := e.newLedger()
	if err := Replay(fresh, journal, NewFinalityView(nil)); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := StateRoot(fresh); string(got) != string(liveRoot) {
		t.Fatalf("state_root свежего узла %x != живому %x — расхождение находки не закрыто",
			got, liveRoot)
	}
}

// TestReplayStrictRejectsCorruptedJournal — strict без послаблений:
// tx, отклоняемая ядром (двойной выпуск одного периода), — ошибка
// пересчёта с диагностикой: tx, заявленный период, период валидации
// (пинирован на claim), вердикт.
func TestReplayStrictRejectsCorruptedJournal(t *testing.T) {
	e := newReplayEnv(t)
	claimIn := func(p uint64) uint64 { return e.genesis + p*e.ps + e.ps/2 }
	good := e.mintTx("S1", 1, claimIn(2), 0, nil)
	// повреждение: повторный выпуск периода 1 тем же кошельком
	// (seq/prev корректны — отказ именно от M3, E_MINT_DOUBLE)
	goodID := good.TxID()
	bad := e.mintTx("S1", 1, claimIn(2), 2, goodID[:])

	l := e.newLedger()
	err := Replay(l, []*wire.Transaction{good, bad}, NewFinalityView(nil))
	if err == nil {
		t.Fatal("strict replay принял повреждённый журнал")
	}
	// диагностика: tx, вердикт, периоды
	for _, want := range []string{"cons: replay:", "E_MINT_DOUBLE", "заявленный период mint 1"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("диагностика: нет %q в %q", want, err.Error())
		}
	}
}
