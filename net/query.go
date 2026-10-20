// net/query.go — read-only запросы состояния реестра для UI приложения
// (этап 7). Исключение из правила 3 промпта этапа 7 (авторизовано
// 19.10.2026): ядро core/ledger работает без мьютексов, Apply сериализуется
// applyMu, а фоновый applyLive идёт параллельно с чтением состояния из UI.
// Без синхронизации это data race, поэтому applyMu переведён в sync.RWMutex
// (Lock — везде Apply, RLock — геттеры этого файла). Поведение протокола не
// меняется: новых записей и новых сетевых сообщений нет.
//
// Правила файла:
//   - только чтение; никаких мутаций состояния;
//   - снапшоты — копии: возвращённые значения безопасны после снятия RLock;
//   - точный момент mint-слота НЕ дублируется (приватная формула
//     mintChallengeSlot в core/ledger): UI v1 достаточно
//     QueryMintWindowOpen (окно открыто/закрыто) и accrual_window (M5);
//     точный момент слота — PENDING-вопрос отчёта этапа 7.
package net

import (
	"bytes"
	"sort"

	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// QueryClock — сетевое время, эпоха и период (9.2). Источник правды —
// эпохи чекпоинтов; время устройства не используется.
func (n *Node) QueryClock() (epoch, netTime, period uint64) {
	if n.Ledger == nil {
		return 0, 0, 0
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	s := n.Ledger.State
	return s.Epoch, s.NetTime, s.CurrentPeriod
}

// QueryWallet — снапшот состояния кошелька (копия; nil — кошелька нет).
// Поля слайсов копируются, поэтому снапшот консистентен на момент вызова.
func (n *Node) QueryWallet(addr []byte) *ledger.Wallet {
	if n.Ledger == nil {
		return nil
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	w := n.Ledger.State.Wallet(addr)
	if w == nil {
		return nil
	}
	snap := *w
	snap.Address = append([]byte(nil), w.Address...)
	snap.Fingerprint = append([]byte(nil), w.Fingerprint...)
	snap.TipTxID = append([]byte(nil), w.TipTxID...)
	snap.Cohorts = append([]ledger.Cohort(nil), w.Cohorts...)
	snap.OccupiedSlots = make([]ledger.Slot, len(w.OccupiedSlots))
	for i, s := range w.OccupiedSlots {
		s.Candidate = append([]byte(nil), s.Candidate...)
		snap.OccupiedSlots[i] = s
	}
	snap.VouchesReceived = make([]ledger.VouchReceipt, len(w.VouchesReceived))
	for i, r := range w.VouchesReceived {
		r.Voucher = append([]byte(nil), r.Voucher...)
		snap.VouchesReceived[i] = r
	}
	snap.Activators = make([][]byte, len(w.Activators))
	for i, a := range w.Activators {
		snap.Activators[i] = append([]byte(nil), a...)
	}
	snap.PersonalGuardians = make([][]byte, len(w.PersonalGuardians))
	for i, g := range w.PersonalGuardians {
		snap.PersonalGuardians[i] = append([]byte(nil), g...)
	}
	return &snap
}

// QueryVouchProgress — сколько vouch набрано кандидатом до K (7.4).
// Возвращает (получено, K); K=0 — профиля нет.
func (n *Node) QueryVouchProgress(addr []byte) (received, k uint32) {
	if n.Ledger == nil {
		return 0, 0
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	w := n.Ledger.State.Wallet(addr)
	if w == nil {
		return 0, n.Ledger.State.Net.Profile.VouchK
	}
	return uint32(len(w.VouchesReceived)), n.Ledger.State.Net.Profile.VouchK
}

// QuerySlotCapacity — размер пула слотов поручителя по возрасту (7.4:
// 3 при 3–12 мес., 5 старше 12; посевному кругу 10 в первый год).
func (n *Node) QuerySlotCapacity(addr []byte) uint32 {
	if n.Ledger == nil {
		return 0
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	s := n.Ledger.State
	w := s.Wallet(addr)
	if w == nil {
		return 0
	}
	age := w.AgeDays(s.NetTime)
	return s.Net.SlotCapacity(age, w.Seed, age)
}

// QueryHistory — публичная история цепочки кошелька (9.1). Полный узел —
// из индекса цепочки; лёгкий — своя цепочка и свои входящие (12.5).
// Порядок по seq; входящие лёгкого узла — в конце, по времени.
func (n *Node) QueryHistory(addr []byte) []*wire.Transaction {
	if n.Cfg != nil && n.Cfg.Level == LevelLight {
		n.mu.RLock()
		defer n.mu.RUnlock()
		if n.Light == nil {
			return nil
		}
		var out []*wire.Transaction
		out = append(out, n.Light.OwnChain...)
		out = append(out, n.Light.Incoming...)
		return out
	}
	n.mu.RLock()
	chain := n.chainIndex[string(addr)]
	var out []*wire.Transaction
	for _, tx := range chain {
		out = append(out, tx)
	}
	n.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

// QueryPendingTransfers — отложенные переводы кошелька (8.3): снапшоты,
// отсчёт владельцу показывает Deadline − NetTime.
func (n *Node) QueryPendingTransfers(addr []byte) []ledger.PendingTransfer {
	if n.Ledger == nil {
		return nil
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	var out []ledger.PendingTransfer
	for k, p := range n.Ledger.State.PendingTransfers {
		if !bytes.HasPrefix([]byte(k), append(append([]byte(nil), addr...), '|')) {
			continue
		}
		snap := *p
		snap.Recipient = append([]byte(nil), p.Recipient...)
		out = append(out, snap)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}

// QueryPendingMigrations — незавершённые миграции, где addr — старый или
// новый адрес (11). Отсчёт: ExecuteNotBefore − NetTime.
func (n *Node) QueryPendingMigrations(addr []byte) []ledger.PendingMigration {
	if n.Ledger == nil {
		return nil
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	var out []ledger.PendingMigration
	for _, m := range n.Ledger.State.PendingMigrations {
		if !bytes.Equal(m.OldAddress, addr) && !bytes.Equal(m.NewAddress, addr) {
			continue
		}
		snap := *m
		snap.NewAddress = append([]byte(nil), m.NewAddress...)
		snap.OldAddress = append([]byte(nil), m.OldAddress...)
		snap.Approvals = make([]wire.GuardianApproval, len(m.Approvals))
		for i, a := range m.Approvals {
			a.Guardian = append([]byte(nil), a.Guardian...)
			a.StatementSig = append([]byte(nil), a.StatementSig...)
			snap.Approvals[i] = a
		}
		out = append(out, snap)
	}
	return out
}

// QueryMintWindowOpen — открыто ли окно челленджа выпуска для периода
// (7.6: слот внутри месяца, окно ±48 ч). Точный момент слота не
// раскрывается (см. шапку файла).
func (n *Node) QueryMintWindowOpen(addr []byte, period, now uint64) bool {
	if n.Ledger == nil {
		return false
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	return n.Ledger.CheckMintChallengeWindow(addr, period, now)
}

// QueryFee — комиссия на перевод суммы (10.2): fee = max(fee_min,
// rate × amount); платит отправитель сверх суммы.
func (n *Node) QueryFee(amount uint64) uint64 {
	if n.Ledger == nil {
		return 0
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	return n.Ledger.State.Net.Fee(amount, n.Ledger.State.FeeRate)
}

// QueryFeeRate — текущая ставка комиссии в micros (адаптивный пересчёт
// из чекпоинтов, 10.2; для UI и диагностики).
func (n *Node) QueryFeeRate() uint64 {
	if n.Ledger == nil {
		return 0
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	return n.Ledger.FeeRateNow()
}

// QueryVouchDepth — расстояние по графу поручительств от посевного круга
// до addr (7.4: анализ графа — вспомогательный слой репутации, без
// автоматических вердиктов). 0 — сам посевной; -1 — недостижим.
// Рёбра: vouch-транзакции (поручитель → кандидат), BFS от всех seed.
func (n *Node) QueryVouchDepth(addr []byte) int {
	if n.Ledger == nil {
		return -1
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	s := n.Ledger.State
	// рёбра графа из состояния: Activators кандидата = его поручители;
	// для BFS от посевного круга нужен обратный порядок (кто за кого).
	// Строим из vouch-транзакций журнала: voucher -> candidate.
	children := make(map[string][][]byte)
	for _, tx := range n.Ledger.AcceptedTxs() {
		if tx.Type != wire.Vouch {
			continue
		}
		vp, ok := tx.Payload.(*wire.VouchPayload)
		if !ok {
			continue
		}
		children[string(tx.Wallet)] = append(children[string(tx.Wallet)], vp.Candidate)
	}
	// BFS от всех посевных кошельков.
	type qi struct {
		addr  []byte
		depth int
	}
	var queue []qi
	seen := make(map[string]bool)
	for _, w := range s.Wallets {
		if w.Seed {
			queue = append(queue, qi{w.Address, 0})
			seen[string(w.Address)] = true
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if bytes.Equal(cur.addr, addr) {
			return cur.depth
		}
		for _, next := range children[string(cur.addr)] {
			if !seen[string(next)] {
				seen[string(next)] = true
				queue = append(queue, qi{next, cur.depth + 1})
			}
		}
	}
	return -1
}

// QueryPublicGuardians — адреса кошельков, объявивших себя публичными
// гардианами (11, guardian_offer). Реестр публичен (9.1).
func (n *Node) QueryPublicGuardians() [][]byte {
	if n.Ledger == nil {
		return nil
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	var out [][]byte
	for _, w := range n.Ledger.State.Wallets {
		if w.PublicGuardian && !w.Terminal {
			out = append(out, append([]byte(nil), w.Address...))
		}
	}
	return out
}

// QueryProfile — параметры профиля сети (genesis; неизменяемы после
// старта, содержимое безопасно). nil — реестра нет (лёгкий узел).
func (n *Node) QueryProfile() *wire.ProfileParams {
	if n.Ledger == nil {
		return nil
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	if n.Ledger == nil {
		return nil
	}
	return &n.Ledger.State.Net.Profile
}

// QueryAwaiting — офлайн-транзакции, ожидающие публикации (12.4:
// «получено по Bluetooth» ≠ оплата). Снапшоты для UI получателя.
func (n *Node) QueryAwaiting() []*wire.Transaction {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]*wire.Transaction, 0, len(n.awaiting))
	for _, tx := range n.awaiting {
		out = append(out, tx)
	}
	return out
}

// QueryRegistry — адреса всех кошельков реестра (9.1: реестр публичен
// внутри сети; экран участника, этап 7). Полный узел; лёгкий — только
// свои данные, возвращает nil.
func (n *Node) QueryRegistry() [][]byte {
	if n.Ledger == nil {
		return nil
	}
	n.applyMu.RLock()
	defer n.applyMu.RUnlock()
	var out [][]byte
	for _, w := range n.Ledger.State.Wallets {
		out = append(out, append([]byte(nil), w.Address...))
	}
	return out
}

// CheckpointChain — финализированная цепочка чекпоинтов узла (копия):
// источник для лёгкого клиента-синхронизатора проверяющей библиотеки
// (6.1: синхронизация с любым узлом, серверы проекта не нужны).
func (n *Node) CheckpointChain() []*wire.Checkpoint {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]*wire.Checkpoint, len(n.Chain))
	copy(out, n.Chain)
	return out
}
