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
	template := fs.String("template", "", "шаблон: 1–8 или имя (laboratory, humanness, quota, voucher, community, contribution, mutual-aid, subscriber-resource)")
	name := fs.String("network-id", "", "метка сети для детерминизма ключа приглашения (по умолчанию — slug; network_id деривируется = H(manifest ‖ genesis))")
	slug := fs.String("slug", "", "человекочитаемое имя сети (обязательно для не-глобальной; метка, не идентичность)")
	global := fs.Bool("global", false, "глобальная сеть (network_id = \"global\", зарезервированная константа)")
	recurring := fs.String("recurring", "auto", "recurring-единицы: auto (из шаблона) | on | off")
	displayName := fs.String("display-name", "", "отображаемое имя сети (по умолчанию slug, до ~24 символов; только внутри приложения, всегда в паре с Eq-хешем)")
	logo := fs.String("logo", "", "логотип организатора для foreground иконки (PNG, необязательно; без него — идентикон из network_id)")
	appVersion := fs.String("app-version", "1.0.0", "версия приложения для имени APK")
	signingDir := fs.String("signing-dir", "", "каталог боевых ключей подписи (по умолчанию $TNP_SIGNING_DIR или ~/.tnp-signing; вне git, тема A8)")
	out := fs.String("out", "", "каталог артефактов (по умолчанию network-<шаблон>)")
	genesisTime := fs.Int64("genesis-time", 0, "genesis-время, unix (0 = сейчас)")
	seeds := fs.String("seeds", "V1,V2,V3", "метки посевного круга, через запятую")
	inviteHex := fs.String("invite-key-hex", "", "ключ приглашения (hex; по умолчанию детерминирован от имени)")
	issuance := fs.Uint64("issuance-amount", 0, "issuance_amount (0 — из шаблона)")
	vouchK := fs.Uint64("vouch-k", 0, "vouch_k (0 — из шаблона)")
	epochSeconds := fs.Uint64("epoch-seconds", 0, "epoch_seconds (0 — из шаблона)")
	migrateDelayDays := fs.Uint64("migrate-delay-days", 0, "migrate_delay_days — задержка исполнения миграции, genesis-поле 49 (0 — из шаблона, 14 сут)")
	periodSeconds := fs.Uint64("period-seconds", 0, "period_seconds — длина периода выпуска, genesis-поле 61 (0 — конвенция 30 сут; тестовые сети живого прогона: границы периода на реальном времени)")
	moratoriumDays := fs.Int64("transfer-moratorium-days", -1, "transfer_moratorium_days — мораторий переводов от register, сут (-1 — из шаблона, 14; 0 — нет моратория)")
	windowHours := fs.Uint64("mint-challenge-window-hours", 0, "mint_challenge_window_hours — окно слота челленджа ±N часов (0 — из шаблона, 48; тестовые сети с period_seconds: окно, накрывающее слоты первых периодов при genesis в прошлом)")
	resourceID := fs.String("resource-id", "", "resource_id / program_id (для профилей 3, 4, 8)")
	providers := fs.String("providers", "", "fingerprint'ы поставщиков (hex 32 байта, через запятую; 6.2 — для restricted/peer-redeem и redeem; v9.11 — fingerprint, адрес выводится из network_id)")
	providerSeed := fs.String("provider-seed", "", "поставщик — кошелёк посевного круга по метке (например V1; адрес из манифеста)")
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
	// идентификация сети (v9.10): slug — человекочитаемая метка (обязательна
	// для не-глобальной сети), глобальная сеть — константа network_id =
	// "global"; network_id не выбирается — деривируется H(manifest ‖ genesis)
	slugName := *slug
	if *global {
		if slugName == "" {
			slugName = "global"
		}
	} else {
		if slugName == "" {
			return fmt.Errorf("нужен -slug (человекочитаемое имя сети; -global — глобальная сеть)")
		}
	}
	if err := validateSlug(slugName); err != nil {
		return err
	}
	if *name == "" {
		*name = slugName
	}

	// параметры: шаблон + оверрайды организатора
	prof := tpl.Profile
	switch *recurring {
	case "auto": // из шаблона (профиль 8 — on, остальные — off)
	case "on":
		prof.Recurring = true
	case "off":
		prof.Recurring = false
	default:
		return fmt.Errorf("-recurring: auto | on | off (получено %q)", *recurring)
	}
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
	// period_seconds (поле 61): длина периода выпуска тестовой сети
	// (живой прогон — границы периода на реальном времени за минуты)
	if *periodSeconds > 0 {
		prof.PeriodSeconds = *periodSeconds
	}
	// transfer_moratorium_days: тестовой сети живого прогона мораторий
	// 14 сут не даёт свежему кошельку приложения дойти до E_BURNED
	// (перевод сгоревшей единицы из UI — пункт чек-листа приложения)
	if *moratoriumDays >= 0 {
		prof.TransferMoratoriumDays = uint64(*moratoriumDays)
	}
	// mint_challenge_window_hours: при period_seconds слоты первых
	// заявляемых периодов (квоты посевного круга с периода 1, quota_reset)
	// лежат у genesis — окно из шаблона (±48 ч) их не накрывает,
	// выпуск посевного круга и vouch×3 недостижимы (расхождение №10
	// этапа 12 в живом прогоне); тестовая сеть задаёт окно явно
	if *windowHours > 0 {
		prof.MintChallengeWindowHours = *windowHours
	}
	if *resourceID != "" {
		prof.ResourceID = *resourceID
	}
	// поставщики (6.2): обычные кошельки; в параметрах сети —
	// fingerprint'ы (v9.11), адрес выводится из финального network_id;
	// нужны для restricted/peer-redeem и погашения через redeem
	if *providers != "" {
		prof.Providers = nil
		for _, h := range strings.Split(*providers, ",") {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			a, err := hex.DecodeString(h)
			if err != nil || len(a) != 32 {
				return fmt.Errorf("-providers: %q не 32-байтный hex-fingerprint", h)
			}
			prof.Providers = append(prof.Providers, a)
		}
	}
	// валидация параметров (раздел 4 + защиты)
	if bad := profiles.Validate(&prof); len(bad) > 0 {
		return fmt.Errorf("профиль не проходит валидацию:\n  - %s", strings.Join(bad, "\n  - "))
	}

	gp := &wire.GenesisPackage{
		Manifest: &wire.Manifest{
			Slug:    slugName,
			Profile: &prof,
			SeedSunset: &wire.SeedSunset{UseActiveWallets: true, ActiveWallets: 100,
				DeadlineEpoch: 24}, // №15
		},
		Genesis: &wire.GenesisRecord{
			GenesisTime:     uint64(*genesisTime),
			EpochZero:       0,
			ProtocolVersion: 3, // mint-fix: якорь слота M7 — текущий период (v9.13: 2 — txNow-семантика)
			SeedSunset:      nil,
		},
	}
	gp.Genesis.SeedSunset = gp.Manifest.SeedSunset
	// display_name (вариант 1): человекочитаемое имя до ~24 символов,
	// по умолчанию slug; показывается только внутри приложения и всегда
	// в паре с Eq-хешем
	dn := *displayName
	if dn == "" {
		dn = slugName
	}
	if len([]rune(dn)) > 24 {
		fmt.Printf("внимание: display_name длиннее ~24 символов: %q\n", dn)
	}
	gp.Manifest.DisplayName = dn
	if *genesisTime == 0 {
		*genesisTime = time.Now().Truncate(time.Hour).Unix()
	}
	gp.Genesis.GenesisTime = uint64(*genesisTime)

	// посевной круг (детерминированные метки — тестовая сеть; живая сеть
	// генерирует ключи на устройствах). Fingerprint-манифест (v9.11):
	// манифест хранит публичные ключи, адреса не хранятся — выводятся
	// из финального network_id, предварительная деривация не нужна
	var circle []wire.SeedMember
	seedLabels := strings.Split(*seeds, ",")
	for _, label := range seedLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		key := seedKey(label)
		circle = append(circle, wire.SeedMember{
			PublicKey: crypto.MarshalPubKey(&key.PublicKey),
		})
	}
	gp.Manifest.SeedCircle = circle
	// поставщик — fingerprint кошелька посевного круга (удобство: ключ
	// уже в манифесте; поставщик не обязан быть посевным участником —
	// цикла само-сертификации больше нет, v9.11); добавляется до
	// деривации network_id
	if *providerSeed != "" {
		idx := -1
		for i, lbl := range seedLabels {
			if strings.TrimSpace(lbl) == *providerSeed {
				idx = i
				break
			}
		}
		if idx < 0 || idx >= len(circle) {
			return fmt.Errorf("-provider-seed %q: нет такой метки в посевном круге", *providerSeed)
		}
		prof.Providers = [][]byte{crypto.Fingerprint(circle[idx].PublicKey)}
		gp.Manifest.Profile = &prof
	}
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
	// само-сертифицирующийся network_id = H(manifest ‖ genesis) (v9.10);
	// глобальная сеть — зарезервированная константа "global"
	wire.FinalizeGenesis(gp)
	if *global {
		gp.Manifest.NetworkID = append([]byte(nil), wire.GlobalNetworkID...)
		gp.Genesis.NetworkID = append([]byte(nil), wire.GlobalNetworkID...)
	}
	networkID := gp.Genesis.NetworkID
	// genesis-валидация комбинаций (v9.10, раздел 4): ошибочные комбинации
	// (recurring в глобальной сети, recurring + free, recurring + не-quota
	// unit, recurring + посторонний transfer_mode) отвергаются при сборке
	if bad := profiles.ValidateGenesis(gp); len(bad) > 0 {
		return fmt.Errorf("генезис не проходит валидацию комбинаций:\n  - %s", strings.Join(bad, "\n  - "))
	}
	if err := wire.VerifySelfCertified(gp); err != nil {
		return fmt.Errorf("генезис не само-сертифицирован: %v", err)
	}
	mh := gp.Genesis.ManifestHash

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
	// артефакты Android-сборки (этап 12, шаг 3): имя APK, applicationId,
	// label Eq-<hash8>, display_name, adaptive-иконка (вариант 3)
	slugComponent := slugName
	if *global {
		slugComponent = "global"
	}
	apk := APKName(slugComponent, prof.Recurring, *appVersion)
	appID := ApplicationID(networkID, slugComponent)
	label := wire.EqLabel(networkID)
	// точка конфигурации подписи (тема A8): вид ключа по типу сети
	// (std → tnp-std, rec → tnp-rec; глобальная сеть — std), signing-конфиг
	// генерируется в локальном каталоге вне git
	sgnDir := *signingDir
	if sgnDir == "" {
		sgnDir = signingDirDefault()
	}
	keyKind, sgnProps, sgnOK, sgnDiag := setupSigning(sgnDir, prof.Recurring)
	spec := iconSpec(networkID, *logo != "", prof.Recurring)
	if err := writeIconAssets(outDir+"/icon", networkID, spec, *logo); err != nil {
		return err
	}
	androidArtifacts := map[string]any{
		"apk_name":       apk,
		"application_id": appID,
		"label":          label,
		"display_name":   dn,
		"slug":           slugName,
		"recurring":      prof.Recurring,
		"network_id":     hex.EncodeToString(networkID),
		"key_kind":       keyKind,
		"icon": map[string]any{
			"background":      spec.Background,
			"palette_index":   spec.PaletteIndex,
			"foreground":      spec.Foreground,
			"badge":           spec.Badge,
			"adaptive_layers": []string{"icon/background.png", "icon/foreground.png"},
		},
	}
	if sgnOK {
		androidArtifacts["signing_properties"] = sgnProps
	}
	aj, err := json.MarshalIndent(androidArtifacts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/android.json", aj, 0o644); err != nil {
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
	if prof.Recurring {
		fmt.Printf(" recurring: on (периодная метка, сгорание после границы P+1) — коммерческая сборка (16.2)\n")
	}
	if prof.PeriodSeconds > 0 {
		fmt.Printf(" period_seconds: %d — период заданной длины (тестовая сеть, границы на реальном времени)\n", prof.PeriodSeconds)
	}
	fmt.Printf(" модули: %v\n посевной круг: %d\n артефакты: %s\n",
		moduleNames(prof.Modules), len(circle), outDir)
	// выведенные адреса посевных и поставщика — контроль организатором:
	// манифест хранит только публичные ключи (v9.11), адреса выводятся
	// из финального network_id и нигде не хранятся
	for i, m := range circle {
		lbl := fmt.Sprintf("seed#%d", i)
		if i < len(seedLabels) {
			if l := strings.TrimSpace(seedLabels[i]); l != "" {
				lbl = l
			}
		}
		fmt.Printf(" посевной %s: адрес %s (выведен из network_id)\n",
			lbl, hex.EncodeToString(crypto.AddressOfKey(m.PublicKey, networkID)))
	}
	for i, fp := range prof.Providers {
		fmt.Printf(" поставщик %d: fingerprint %s → адрес %s (выведен из network_id)\n",
			i, hex.EncodeToString(fp),
			hex.EncodeToString(crypto.DeriveAddress(fp, networkID)))
	}
	fmt.Printf(" APK: %s\n applicationId: %s\n label: %s\n display_name: %q · %s\n",
		apk, appID, label, dn, label)
	// подпись (тема A8): вид ключа и готовая команда release-сборки;
	// пароли не печатаются — они в signing.properties (chmod 600, вне git)
	if sgnOK {
		fmt.Printf(" release-подпись: tnp-%s — signing.properties: %s\n", keyKind, sgnProps)
		fmt.Printf(" сборка release: TNP_SIGNING_PROPERTIES=%s TNP_KEY_KIND=%s "+
			"ORG_GRADLE_PROJECT_TNP_APPLICATION_ID=%s ORG_GRADLE_PROJECT_TNP_APP_LABEL=%s flutter build apk --release\n",
			sgnProps, keyKind, appID, label)
	} else {
		fmt.Printf(" внимание (release-подпись): %s\n", sgnDiag)
	}
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
		return nil, fmt.Errorf("нужно -template (1–8 или имя; -list — список)")
	}
	var tpl *profiles.Template
	if n := len(s); n == 1 && s[0] >= '1' && s[0] <= '8' {
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
	case wire.TransferModePeerRedeem:
		return "peer-redeem"
	default:
		return "unspecified"
	}
}
