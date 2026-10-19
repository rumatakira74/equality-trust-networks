// Сценарии симуляции (9.1: «проверить до реализации»; промпт этапа 5, п. 4):
// конфликт веток; эквивокация; двойной выпуск; потеря связности и
// расщепление сети с последующим слиянием; злонамеренное меньшинство в
// голосовании (доля варьируется); холодный старт ~10 узлов;
// members_quorum при офлайне части участников; переход seed_sunset.
// Каждый сценарий детерминирован по seed; метрики — в Network.Metrics.
package sim

import (
	"encoding/binary"
	"fmt"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// buildTx — сборка валидной транзакции от узла from (seq/prev по состоянию
// узла-строителя; лёгкий PoW + подпись; малые таргеты sim-genesis).
func (net *Network) buildTx(from int, typ wire.TxType, payload wire.Payload) *wire.Transaction {
	n := net.Nodes[from]
	w := n.Ledger.State.Wallet(n.Wallet)
	tx := &wire.Transaction{
		NetworkID:       net.GP.Genesis.NetworkID,
		Wallet:          n.Wallet,
		Seq:             w.Seq + 1,
		Prev:            append([]byte(nil), w.TipTxID...),
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: net.GP.Genesis.ProtocolVersion,
		TimestampClaim:  net.Time,
	}
	solvePoW(tx, net.GP.Manifest.Profile.Pow, typ == wire.Register || typ == wire.Mint)
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(n.Key, sh[:])
	if err != nil {
		panic(err)
	}
	tx.Signature = sig
	return tx
}

// solvePoW — подбор nonce (как net.SolveTxPoW; локальная копия, чтобы sim
// не тянул libp2p-зависимости).
func solvePoW(tx *wire.Transaction, pow *wire.PowParams, hard bool) {
	if hard {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
		ap := crypto.Argon2Params{MemoryKiB: pow.Argon2MemoryKiB,
			Iterations: pow.Argon2Iterations, Parallelism: pow.Argon2Parallelism}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyHard(tx.SigningPreimage(), nb[:], pow.TargetHard, ap) {
				return
			}
		}
	}
	tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		tx.Pow.Nonce = nb[:]
		if crypto.VerifyLight(tx.SigningPreimage(), nb[:], pow.TargetLight) {
			return
		}
	}
}

// transferTx — перевод с кошелька узла from на кошелёк узла to.
func (net *Network) transferTx(from, to int, amount uint64) *wire.Transaction {
	return net.buildTx(from, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: net.Nodes[to].Wallet, Amount: amount},
	})
}

// ScenarioResult — итог сценария для отчёта и проверок.
type ScenarioResult struct {
	Name        string
	Metrics     Metrics
	Winner      string // "A" | "B" | "" (не финализировано)
	Converged   bool   // все узлы сошились к одному состоянию
	Checkpoints int
	Flagged     bool // эквивокация помечена
	Details     string
}

// injectConflictPair — конфликт двух веток одного seq: сначала firstForA
// узлы видят ветку A, остальные — B (70/30 — асимметрия большинства).
func (net *Network) injectConflictPair(a, b *wire.Transaction, fracForA float64) []int {
	var firstForA []int
	for i, n := range net.Nodes {
		if float64(i) < fracForA*float64(len(net.Nodes)) {
			firstForA = append(firstForA, i)
		}
		_ = n
	}
	net.InjectConflict(a, b, firstForA)
	return firstForA
}

// winnerOf — победившая ветка по финализированному выбору сети:
// приоритет — чекпоинт (finalized_txid_list; 9.1 «побеждает ветка,
// выбранная финализацией»), затем согласованное локальное голосование.
// При расщеплении группы голосуют независимо (Snowball финализируется
// в каждой) — общий выбор даёт только чекпоинт.
func (net *Network) winnerOf(key string, branches ...[32]byte) []byte {
	// чекпоинт: первая цепочка, где ветка зафиксирована
	for _, n := range net.Nodes {
		view := cons.NewFinalityView(n.Chain)
		for _, br := range branches {
			if view.IsFinalized(br[:]) {
				return br[:]
			}
		}
	}
	// локальное голосование (сошлись все узлы)
	var winner []byte
	for _, n := range net.Nodes {
		st := n.Snowball.State(key)
		if st == nil || !st.Final {
			return nil
		}
		if winner == nil {
			winner = st.Pref
		} else if string(winner) != string(st.Pref) {
			return nil
		}
	}
	return winner
}

// converged — все онлайн-узлы в одном состоянии (корни совпадают).
func (net *Network) converged() bool {
	var root []byte
	for _, n := range net.Nodes {
		if n.Offline {
			continue
		}
		r := cons.StateRoot(n.Ledger)
		if root == nil {
			root = r
		} else if string(root) != string(r) {
			return false
		}
	}
	return true
}

// RunConflict — сценарий 1: конфликт веток (70/30), финальность голосованием.
func RunConflict(seed int64, n int) ScenarioResult {
	net := New(seed, n)
	// первый перевод (seq 1) — общий префикс
	first := net.transferTx(5, 6, 100)
	net.InjectTx(5, first)
	net.Run(2)
	// конфликт на seq 2: A — большинству первым, B — меньшинству
	a := net.transferTx(5, 6, 50)
	b := net.buildTx(5, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: net.Nodes[7].Wallet, Amount: 50},
	})
	net.injectConflictPair(a, b, 0.7)
	net.Run(30)
	key := cons.ConflictKey(a)
	idA, idB := a.TxID(), b.TxID()
	winner := net.winnerOf(key, idA, idB)
	res := ScenarioResult{Name: fmt.Sprintf("конфликт веток (n=%d)", n),
		Metrics: net.Metrics, Converged: net.converged(),
		Checkpoints: net.Metrics.Checkpoints}
	if winner != nil {
		idA := a.TxID()
		if string(winner) == string(idA[:]) {
			res.Winner = "A"
		} else {
			res.Winner = "B"
		}
	}
	return res
}

// RunEquivocation — сценарий 2: эквивокация + evidence (7.5): ветка
// выбирается голосованием, кошелёк помечается flagged (доказательство
// подаёт любой узел, вердиктов нет).
func RunEquivocation(seed int64, n int) ScenarioResult {
	net := New(seed, n)
	first := net.transferTx(5, 6, 100)
	net.InjectTx(5, first)
	net.Run(2)
	a := net.transferTx(5, 6, 50)
	b := net.buildTx(5, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: net.Nodes[7].Wallet, Amount: 50},
	})
	net.injectConflictPair(a, b, 0.7)
	net.Run(10)
	// evidence подаёт узел 9 (любой узел; 7.5)
	ev := net.buildTx(9, wire.Evidence, &wire.EvidencePayload{
		TxA: a, TxB: b, Kind: wire.EvidenceEquivocation,
	})
	net.InjectTx(9, ev)
	net.Run(25)
	w := net.Nodes[5].Ledger.State.Wallet(net.Nodes[5].Wallet)
	res := ScenarioResult{Name: fmt.Sprintf("эквивокация + evidence (n=%d)", n),
		Metrics: net.Metrics, Converged: net.converged(), Flagged: w.Flagged,
		Checkpoints: net.Metrics.Checkpoints}
	idA, idB := a.TxID(), b.TxID()
	if winner := net.winnerOf(cons.ConflictKey(a), idA, idB); winner != nil {
		if string(winner) == string(idA[:]) {
			res.Winner = "A"
		} else {
			res.Winner = "B"
		}
	}
	return res
}

// RunDoubleMint — сценарий 3: двойной выпуск за один период (частный
// случай эквивокации, 7.5): два mint одного period; периоды меняются на
// границах эпох (9.2), поэтому сценарий прогоняет ~2 календарных месяца.
func RunDoubleMint(seed int64, n int) ScenarioResult {
	net := New(seed, n)
	// ждём period >= 2 (две границы «1-го числа» после genesis):
	// genesis 2024-01-15 -> Feb 1 (period 1) -> Mar 1 (period 2)
	net.Run(200)
	// mint периода 1 двумя ветками (один seq)
	a := net.buildTx(5, wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000})
	b := net.buildTx(5, wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000})
	// b отличается от a только nonce PoW — сделаем отличимым получателем
	// невозможно (mint без получателя): отличаем nonce/подписью
	net.InjectConflict(a, b, nil)
	net.Run(30)
	res := ScenarioResult{Name: fmt.Sprintf("двойной выпуск (n=%d)", n),
		Metrics: net.Metrics, Converged: net.converged(),
		Checkpoints: net.Metrics.Checkpoints}
	idA, idB := a.TxID(), b.TxID()
	if winner := net.winnerOf(cons.ConflictKey(a), idA, idB); winner != nil {
		if string(winner) == string(idA[:]) {
			res.Winner = "A"
		} else {
			res.Winner = "B"
		}
	}
	return res
}

// RunPartitionHeal — сценарий 4: расщепление сети на две группы с разными
// ветками, затем слияние. members_quorum: во время расщепления ни одна
// группа не достигает 2/3 — чекпоинтов нет; после слияния голосование
// выбирает ветку, чекпоинт фиксирует.
func RunPartitionHeal(seed int64, n int) ScenarioResult {
	// members_quorum сразу (seed_sunset достигнут по числу кошельков)
	net := New(seed, n, WithMode(wire.CheckpointModeMembersQuorum), WithSeedSunset(1, 1))
	// общий префикс (seq 1) доставляется всем ДО расщепления
	first := net.transferTx(5, 6, 100)
	net.InjectTx(5, first)
	net.Run(2)
	// асимметричное расщепление 70/30 (симметричное 50/50 не даёт кворума
	// 0,7 ни одной стороне — конфликт висит до восстановления связности)
	groups := make(map[int]int)
	for i := 0; i < n; i++ {
		if i < n*7/10 {
			groups[i] = 1
		} else {
			groups[i] = 2
		}
	}
	net.SetPartition(groups)
	a := net.transferTx(5, 6, 50)
	b := net.buildTx(5, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: net.Nodes[7].Wallet, Amount: 50},
	})
	// группа 1 (70%) видит A первой, группа 2 (30%) — B
	var firstForA []int
	for i := 0; i < n*7/10; i++ {
		firstForA = append(firstForA, i)
	}
	net.InjectConflict(a, b, firstForA)
	net.Run(12)
	during := net.Metrics.Checkpoints
	// слияние
	net.Heal()
	net.Run(30)
	res := ScenarioResult{Name: fmt.Sprintf("расщепление + слияние (n=%d)", n),
		Metrics: net.Metrics, Converged: net.converged(),
		Checkpoints: net.Metrics.Checkpoints,
		Details:     fmt.Sprintf("чекпоинтов во время расщепления: %d", during)}
	idA, idB := a.TxID(), b.TxID()
	if winner := net.winnerOf(cons.ConflictKey(a), idA, idB); winner != nil {
		if string(winner) == string(idA[:]) {
			res.Winner = "A"
		} else {
			res.Winner = "B"
		}
	}
	return res
}

// RunMaliciousMinority — сценарий 5: доля злонамеренных f (голосуют за
// ветку атакующего B, игнорируя своё состояние). Возвращает победителя.
func RunMaliciousMinority(seed int64, n int, frac float64) ScenarioResult {
	net := New(seed, n)
	mal := int(float64(n) * frac)
	for i := 0; i < mal; i++ {
		net.Nodes[n-1-i].Malicious = true
	}
	first := net.transferTx(5, 6, 100)
	net.InjectTx(5, first)
	net.Run(2)
	// честные видят A первой; злонамеренные — B (и голосуют B)
	a := net.transferTx(5, 6, 50)
	b := net.buildTx(5, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: net.Nodes[7].Wallet, Amount: 50},
	})
	var firstForA []int
	for i := 0; i < n-mal; i++ {
		firstForA = append(firstForA, i)
	}
	net.InjectConflict(a, b, firstForA)
	net.Run(30)
	res := ScenarioResult{Name: fmt.Sprintf("злонамеренное меньшинство f=%.0f%% (n=%d)", frac*100, n),
		Metrics: net.Metrics, Converged: net.converged(),
		Checkpoints: net.Metrics.Checkpoints}
	idA, idB := a.TxID(), b.TxID()
	if winner := net.winnerOf(cons.ConflictKey(a), idA, idB); winner != nil {
		if string(winner) == string(idA[:]) {
			res.Winner = "A"
		} else {
			res.Winner = "B"
		}
	}
	return res
}

// RunColdStart — сценарий 6: холодный старт ~10 узлов: переходный режим
// (посевной круг N-из-M) подписывает первые чекпоинты; голосование
// сходится и в малой сети.
func RunColdStart(seed int64) ScenarioResult {
	net := New(seed, 10) // SeedSunset 100/24 не достигнут — переходный режим
	first := net.transferTx(5, 6, 100)
	net.InjectTx(5, first)
	net.Run(2)
	a := net.transferTx(5, 6, 50)
	b := net.buildTx(5, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: net.Nodes[7].Wallet, Amount: 50},
	})
	net.injectConflictPair(a, b, 0.7)
	net.Run(30)
	res := ScenarioResult{Name: "холодный старт (10 узлов, посевной круг)",
		Metrics: net.Metrics, Converged: net.converged(),
		Checkpoints: net.Metrics.Checkpoints}
	if len(net.Nodes[0].Chain) > 0 {
		res.Details = fmt.Sprintf("подписанты первого чекпоинта: %d (посевной круг)",
			len(net.Nodes[0].Chain[0].Participants))
	}
	idA, idB := a.TxID(), b.TxID()
	if winner := net.winnerOf(cons.ConflictKey(a), idA, idB); winner != nil {
		if string(winner) == string(idA[:]) {
			res.Winner = "A"
		} else {
			res.Winner = "B"
		}
	}
	return res
}

// RunMembersQuorumOffline — сценарий 7: members_quorum при офлайне части
// участников: чекпоинт финализируется, пока онлайн ≥ 2/3; граница — 1/3
// офлайна. Возвращает долю офлайна и число чекпоинтов.
func RunMembersQuorumOffline(seed int64, n int, offlineFrac float64) ScenarioResult {
	// members_quorum сразу (без переходного режима посевного круга)
	net := New(seed, n, WithMode(wire.CheckpointModeMembersQuorum), WithSeedSunset(1, 1))
	off := int(float64(n) * offlineFrac)
	for i := 0; i < off; i++ {
		net.Nodes[n-1-i].Offline = true
	}
	net.Run(24)
	return ScenarioResult{
		Name:    fmt.Sprintf("members_quorum, офлайн %.0f%% (n=%d)", offlineFrac*100, n),
		Metrics: net.Metrics, Converged: true,
		Checkpoints: net.Metrics.Checkpoints,
	}
}

// RunSeedSunset — сценарий 8: переходный режим -> режим манифеста по
// seed_sunset (активные >= порога или эпоха >= порога).
func RunSeedSunset(seed int64) ScenarioResult {
	net := New(seed, 20, WithSeedSunset(100, 3)) // sunset по эпохе 3
	net.Run(28)                                  // 7 эпох (4 раунда на эпоху)
	res := ScenarioResult{Name: "переход seed_sunset (эпоха 3)",
		Metrics: net.Metrics, Converged: true,
		Checkpoints: net.Metrics.Checkpoints}
	chain := net.Nodes[0].Chain
	if len(chain) >= 4 {
		transitional := len(chain[0].Participants)
		after := len(chain[len(chain)-1].Participants)
		res.Details = fmt.Sprintf("подписанты: переходный=%d, после sunset=%d", transitional, after)
		res.Flagged = transitional != after
	}
	return res
}

// RunSmallNetworkBoundary — граница малых сетей для members_quorum
// (раздел 15): размер чекпоинта и финальность при росте сети
// (20% участников офлайн — типичный мобильный сценарий).
func RunSmallNetworkBoundary(seed int64) []ScenarioResult {
	var out []ScenarioResult
	for _, n := range []int{10, 30, 100, 300} {
		out = append(out, RunMembersQuorumOffline(seed, n, 0.2))
	}
	return out
}
