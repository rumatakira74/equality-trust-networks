// export — экспорт реестра песочницы для анализа (6.5, этап 10):
// оборот, предложение, неравенство считаются из публичного реестра любым
// клиентом. Форматы: json (полный снимок + метрики) и csv (кошельки —
// для таблиц/скриптов). Реестр читается с узла-драйвера (как state).
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func exportCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	out := fs.String("out", "", "файл вывода (по умолчанию — stdout)")
	format := fs.String("format", "json", "json | csv")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()

	w := os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	if *format == "csv" {
		return exportCSV(w, env)
	}
	return exportJSON(w, env)
}

// walletRow — строка реестра кошелька.
type walletRow struct {
	Address     string `json:"address"`
	Status      string `json:"status"`
	Synthetic   bool   `json:"synthetic"`
	Seed        bool   `json:"seed"`
	Flagged     bool   `json:"flagged"`
	Balance     uint64 `json:"balance"`
	Liquid      uint64 `json:"liquid"`
	Cohorts     uint64 `json:"cohorts"` // незрелые
	Registered  uint64 `json:"registered_at"`
	Activated   uint64 `json:"activated_at"`
	LastPeriod  uint64 `json:"last_claimed_period"`
	Seq         uint64 `json:"seq"`
}

func exportJSON(w *os.File, env *txEnv) error {
	st := env.node.Ledger.State
	wallets := make([]walletRow, 0, len(st.Wallets))
	var balances []float64
	for _, wl := range st.Wallets {
		wallets = append(wallets, walletRow{
			Address:    fmt.Sprintf("%x", wl.Address),
			Status:     wl.Status.String(),
			Synthetic:  wl.Synthetic,
			Seed:       wl.Seed,
			Flagged:    wl.Flagged,
			Balance:    wl.TotalBalance(),
			Liquid:     wl.Liquid,
			Cohorts:    wl.TotalBalance() - wl.Liquid,
			Registered: wl.RegisteredAt,
			Activated:  wl.ActivatedAt,
			LastPeriod: wl.LastClaimedPeriod,
			Seq:        wl.Seq,
		})
		balances = append(balances, float64(wl.TotalBalance()))
	}
	sort.Slice(wallets, func(i, j int) bool { return wallets[i].Address < wallets[j].Address })

	// оборот по типам из публичного журнала (6.5: любой клиент)
	turnover := map[string]uint64{}
	txCount := map[string]uint64{}
	for _, tx := range env.node.Ledger.AcceptedTxs() {
		name := tx.Type.String()
		txCount[name]++
		switch tx.Type {
		case wire.Transfer:
			if tp, ok := tx.Payload.(*wire.TransferPayload); ok && tp.Send != nil {
				turnover["transfer"] += tp.Send.Amount
			}
		case wire.Redeem:
			if rp, ok := tx.Payload.(*wire.RedeemPayload); ok {
				turnover["redeem"] += rp.Amount
			}
		case wire.PoolDonate:
			if pp, ok := tx.Payload.(*wire.PoolDonatePayload); ok {
				turnover["pool_donate"] += pp.Amount
			}
		}
	}

	// пулы (6.3): прогресс публичен
	type poolRow struct {
		ID       string `json:"id"`
		Creator  string `json:"creator"`
		Goal     uint64 `json:"goal"`
		Balance  uint64 `json:"balance"`
		Released bool   `json:"released"`
		Closed   bool   `json:"closed"`
		Mode     string `json:"release_mode"`
	}
	pools := make([]poolRow, 0)
	for _, p := range st.Pools {
		pools = append(pools, poolRow{
			ID: fmt.Sprintf("%x", p.ID), Creator: fmt.Sprintf("%x", p.Creator),
			Goal: p.Goal, Balance: p.Balance, Released: p.Released, Closed: p.Closed,
			Mode: releaseModeName(p.ReleaseMode),
		})
	}
	sort.Slice(pools, func(i, j int) bool { return pools[i].ID < pools[j].ID })

	active, synthetic := 0, 0
	for _, wl := range st.Wallets {
		if wl.Status == wire.StatusActive {
			active++
		}
		if wl.Synthetic {
			synthetic++
		}
	}

	out := map[string]any{
		"epoch":   st.Epoch,
		"period":  st.CurrentPeriod,
		"wallets": wallets,
		"pools":   pools,
		"metrics": map[string]any{
			"wallets_total":     len(st.Wallets),
			"wallets_active":    active,
			"wallets_synthetic": synthetic,
			"supply_minted":     st.TotalMinted,
			"supply_burned":     st.TotalBurned,
			"supply_net":        st.TotalMinted - st.TotalBurned,
			"common_fund":       st.CommonFund,
			"turnover":          turnover,
			"tx_count":          txCount,
			"gini_balance":      gini(balances),
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func exportCSV(w *os.File, env *txEnv) error {
	st := env.node.Ledger.State
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"address", "status", "synthetic", "seed", "flagged",
		"balance", "liquid", "cohorts", "registered_at", "activated_at",
		"last_claimed_period", "seq",
	}); err != nil {
		return err
	}
	rows := make([][]string, 0, len(st.Wallets))
	for _, wl := range st.Wallets {
		rows = append(rows, []string{
			fmt.Sprintf("%x", wl.Address), wl.Status.String(),
			btoa(wl.Synthetic), btoa(wl.Seed), btoa(wl.Flagged),
			strconv.FormatUint(wl.TotalBalance(), 10),
			strconv.FormatUint(wl.Liquid, 10),
			strconv.FormatUint(wl.TotalBalance()-wl.Liquid, 10),
			strconv.FormatUint(wl.RegisteredAt, 10),
			strconv.FormatUint(wl.ActivatedAt, 10),
			strconv.FormatUint(wl.LastClaimedPeriod, 10),
			strconv.FormatUint(wl.Seq, 10),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })
	cw.WriteAll(rows)
	cw.Flush()
	return cw.Error()
}

// gini — коэффициент Джини по балансам кошельков (6.5: неравенство
// считается из публичного реестра). 0 — полное равенство.
func gini(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	var sum, cum float64
	for _, v := range sorted {
		sum += v
	}
	if sum == 0 {
		return 0
	}
	for i, v := range sorted {
		cum += float64(i+1)*v - float64(i)*v
	}
	return (2*cum - sorted[0]*float64(n) - sum) / (float64(n) * sum)
}

func releaseModeName(m wire.PoolReleaseMode) string {
	switch m {
	case wire.PoolReleaseOnDeadline:
		return "release_on_deadline"
	case wire.PoolReleaseOnGoal:
		return "release_on_goal"
	case wire.PoolReleaseContinuous:
		return "release_continuous"
	default:
		return "unspecified"
	}
}
