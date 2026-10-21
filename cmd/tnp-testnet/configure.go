// configure — конфигуратор сети (задача 7, этап 10): организатор без
// программирования создаёт сеть-инстанс по шаблону профиля (раздел 5):
// шаблон → параметры → genesis-пакет + артефакты приложения, уровень
// гарантий вычисляется из манифеста (3.3).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/profiles"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func configureCmd(args []string) error {
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	list := fs.Bool("list", false, "показать шаблоны профилей и выйти")
	template := fs.String("template", "", "шаблон: 1–7 или имя (laboratory, humanness, quota, voucher, community, contribution, mutual-aid)")
	name := fs.String("network-id", "", "имя сети (обязательно; network_id = SHA-256 префикса+имени)")
	out := fs.String("out", "", "каталог артефактов (по умолчанию network-<шаблон>)")
	genesisTime := fs.Int64("genesis-time", 0, "genesis-время, unix (0 = сейчас)")
	seeds := fs.String("seeds", "V1,V2,V3", "метки посевного круга, через запятую")
	inviteHex := fs.String("invite-key-hex", "", "ключ приглашения (hex; по умолчанию детерминирован от имени)")
	issuance := fs.Uint64("issuance-amount", 0, "issuance_amount (0 — из шаблона)")
	vouchK := fs.Uint64("vouch-k", 0, "vouch_k (0 — из шаблона)")
	epochSeconds := fs.Uint64("epoch-seconds", 0, "epoch_seconds (0 — из шаблона)")
	migrateDelayDays := fs.Uint64("migrate-delay-days", 0, "migrate_delay_days — задержка исполнения миграции, genesis-поле 49 (0 — из шаблона, 14 сут)")
	resourceID := fs.String("resource-id", "", "resource_id / program_id (для профилей 3, 4)")
	displayName := fs.String("display-name", "", "отображаемое имя сети")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *list {
		return listTemplates()
	}
	tpl, err := pickTemplate(*template)
	if err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("нужно -network-id (имя сети)")
	}

	// параметры: шаблон + оверрайды организатора
	prof := tpl.Profile
	if *issuance > 0 {
		prof.IssuanceAmount = *issuance
	}
	if *vouchK > 0 {
		prof.VouchK = uint32(*vouchK)
	}
	if *epochSeconds > 0 {
		prof.EpochSeconds = *epochSeconds
	}
	// migrate_delay_days (поле 49): окно отмены живым старым ключом до
	// исполнения миграции (11); оверрайд шаблона (находка №6 — в прогоне
	// stage11-prep поле выставлялось пост-обработкой хелпером)
	if *migrateDelayDays > 0 {
		prof.MigrateDelayDays = *migrateDelayDays
	}
	if *resourceID != "" {
		prof.ResourceID = *resourceID
	}
	// валидация параметров (раздел 4 + защиты)
	if bad := profiles.Validate(&prof); len(bad) > 0 {
		return fmt.Errorf("профиль не проходит валидацию:\n  - %s", strings.Join(bad, "\n  - "))
	}

	networkID := networkIDOf(*name)
	gp := &wire.GenesisPackage{
		Manifest: &wire.Manifest{
			NetworkID: networkID,
			Profile:   &prof,
			SeedSunset: &wire.SeedSunset{UseActiveWallets: true, ActiveWallets: 100,
				DeadlineEpoch: 24}, // №15
		},
		Genesis: &wire.GenesisRecord{
			NetworkID:       networkID,
			GenesisTime:     uint64(*genesisTime),
			EpochZero:       0,
			ProtocolVersion: 1,
			SeedSunset:      nil,
		},
	}
	gp.Genesis.SeedSunset = gp.Manifest.SeedSunset
	dn := *displayName
	if dn == "" {
		dn = tpl.DisplayName + " " + *name
	}
	gp.Manifest.DisplayName = dn
	if *genesisTime == 0 {
		*genesisTime = time.Now().Truncate(time.Hour).Unix()
	}
	gp.Genesis.GenesisTime = uint64(*genesisTime)

	// посевной круг (детерминированные метки — тестовая сеть; живая сеть
	// генерирует ключи на устройствах)
	var circle []wire.SeedMember
	seedLabels := strings.Split(*seeds, ",")
	for _, label := range seedLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		key := seedKey(label)
		circle = append(circle, wire.SeedMember{
			Address:   addrOf(key, networkID),
			PublicKey: crypto.MarshalPubKey(&key.PublicKey),
		})
	}
	gp.Manifest.SeedCircle = circle
	inviteKey := []byte(*inviteHex)
	if *inviteHex == "" {
		h := sha256.Sum256([]byte("tnp-testnet-invite:" + *name))
		inviteKey = h[:]
	} else {
		b, err := hex.DecodeString(*inviteHex)
		if err != nil {
			return fmt.Errorf("-invite-key-hex: %w", err)
		}
		inviteKey = b
	}
	if prof.EntryMode == wire.EntryInviteOnlyVouch {
		gp.Genesis.InviteKey = inviteKey
	}
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]

	outDir := *out
	if outDir == "" {
		outDir = "network-" + tpl.Name
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/genesis.bin", gp.EncodeGenesisPackage(), 0o644); err != nil {
		return err
	}
	mj, err := manifestJSON(gp, *name, seedLabels, inviteKey)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/manifest.json", mj, 0o644); err != nil {
		return err
	}
	// уровень гарантий вычисляется из манифеста (3.3) — фиксируется в
	// артефакте для организатора и партнёров
	level, basis := profiles.GuaranteeLevel(&prof)
	guar := map[string]any{
		"level":         level,
		"basis":         basis,
		"template":      tpl.Name,
		"template_id":   tpl.ID,
		"network_id":    hex.EncodeToString(networkID),
		"manifest_hash": hex.EncodeToString(mh[:]),
	}
	gb, err := json.MarshalIndent(guar, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/guarantees.json", gb, 0o644); err != nil {
		return err
	}
	// приглашение (приложение подключается по genesis-пакету + ключу)
	var inv strings.Builder
	fmt.Fprintf(&inv, "Сеть %s (%s)\n", *name, tpl.DisplayName)
	fmt.Fprintf(&inv, "genesis-пакет: genesis.bin (передаётся вместе с приглашением)\n")
	fmt.Fprintf(&inv, "manifest_hash: %s\n", hex.EncodeToString(mh[:]))
	if gp.Genesis.InviteKey != nil {
		fmt.Fprintf(&inv, "invite_key: %s\n", hex.EncodeToString(inviteKey))
	} else {
		fmt.Fprintf(&inv, "вход: открытые поручительства (open_vouch), ключ не нужен\n")
	}
	if err := os.WriteFile(outDir+"/invite.txt", []byte(inv.String()), 0o644); err != nil {
		return err
	}

	fmt.Printf("сеть %s по шаблону %d (%s)\n", *name, tpl.ID, tpl.DisplayName)
	fmt.Printf(" network_id: %s\n manifest_hash: %s\n genesis_time: %d (%s)\n",
		hex.EncodeToString(networkID), hex.EncodeToString(mh[:]), *genesisTime,
		time.Unix(*genesisTime, 0).UTC().Format(time.RFC3339))
	fmt.Printf(" уровень гарантий: %s (%s)\n", level, basis)
	fmt.Printf(" единица: %s, выпуск: %d/%d мес., передача: %s, вход: %s, K=%d\n",
		unitName(prof.Unit), prof.IssuanceAmount, prof.IssuancePeriodMonths,
		transferModeName(prof.TransferMode), entryName(prof.EntryMode), prof.VouchK)
	fmt.Printf(" модули: %v\n посевной круг: %d\n артефакты: %s\n",
		moduleNames(prof.Modules), len(circle), outDir)
	return nil
}

func listTemplates() error {
	for _, tpl := range profiles.All() {
		level, _ := profiles.GuaranteeLevel(&tpl.Profile)
		fmt.Printf("%d  %-13s %s\n    уровень гарантий: %s\n    %s\n",
			tpl.ID, tpl.Name, tpl.DisplayName, level, tpl.Description)
	}
	return nil
}

func pickTemplate(s string) (*profiles.Template, error) {
	if s == "" {
		return nil, fmt.Errorf("нужно -template (1–7 или имя; -list — список)")
	}
	var tpl *profiles.Template
	if n := len(s); n == 1 && s[0] >= '1' && s[0] <= '7' {
		tpl = profiles.ByID(int(s[0] - '0'))
	} else {
		tpl = profiles.ByName(s)
	}
	if tpl == nil {
		return nil, fmt.Errorf("шаблон %q не найден (-list — список)", s)
	}
	return tpl, nil
}

func moduleNames(mods []uint32) []string {
	var out []string
	for _, m := range mods {
		switch m {
		case wire.ModuleAttest:
			out = append(out, "attest")
		case wire.ModuleRedeem:
			out = append(out, "redeem")
		case wire.ModulePool:
			out = append(out, "pool")
		case wire.ModuleContribution:
			out = append(out, "contribution")
		case wire.ModuleSandbox:
			out = append(out, "sandbox")
		}
	}
	return out
}

func entryName(v wire.EntryMode) string {
	switch v {
	case wire.EntryOpenVouch:
		return "open_vouch"
	case wire.EntryFoundersSeed:
		return "founders_seed"
	case wire.EntryInviteOnlyVouch:
		return "invite_only_vouch"
	default:
		return "unspecified"
	}
}

func unitName(v wire.UnitType) string {
	switch v {
	case wire.UnitCurrency:
		return "currency"
	case wire.UnitHours:
		return "hours"
	case wire.UnitQuota:
		return "quota"
	case wire.UnitPoints:
		return "points"
	case wire.UnitVoucher:
		return "voucher"
	case wire.UnitHeartbeat:
		return "heartbeat"
	default:
		return "unspecified"
	}
}

func transferModeName(v wire.TransferMode) string {
	switch v {
	case wire.TransferModeFree:
		return "free"
	case wire.TransferModeRestricted:
		return "restricted"
	case wire.TransferModeSoulbound:
		return "soulbound"
	default:
		return "unspecified"
	}
}
