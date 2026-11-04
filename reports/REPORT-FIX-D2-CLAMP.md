---
theme: fix-d2-clamp
task: "00000036"
branch: fix-d2-clamp
base: origin/main @ c7cfb02 (Merge branch 'fix-release-keys' into main)
session-id: 1be42beb-6d20-2a59-1cd6-6c589f1044a2
mandate: prompts/fix-d2-clamp-prompt.md (blob 7a2b8ae)
source: reports/REPORT-AUDIT-MINT-FIX.md §8, AUDIT-MINT-FIX-02 (P2, аудит pre-live — D2)
model:
  alias: deepseek-flash
  thinking: max
window:
  start: 2026-11-04T18:00:07Z
  timeout_min: 90
  checkpoints:
    - 2026-11-04T18:01:46Z  # шаг 2 (фикс), коммит d189335
    - 2026-11-04T18:04:46Z  # шаг 3 (тесты), коммит 2b03084
verdict: шаги 1–3 и 5 выполнены; go build/test ./... и flutter analyze/test зелёные; НЕ смержено
---

# REPORT-FIX-D2-CLAMP

AUDIT-MINT-FIX-02 (D2 аудита pre-live) — до релиза v3.0.0. Ветка
`fix-d2-clamp` от свежего `main` (`c7cfb02`), запушена, **не смержена**
(merge исполняет диспетчер при зелёном CI). Ядро (`core/`) и протокол не
менялись: правка — только mobile-мост (`mobile/query.go`) и его тесты.

Дефект: `MintWindowJSON` предлагал `claim = period − 1` без ограничения
окном накопления M5. Кошелёк, пропустивший ≥ `accrual_window` периодов,
получал «отравленный» claim: UI предлагал подачу, ядро отвечало
`E_ACCRUAL_WINDOW` — кнопка выпуска мертва навсегда (поглощающее
состояние). `accrual_window` при этом отдавался в JSON, но для клампа не
использовался.

## 1. План (как выполнен)

| Шаг мандата | Что сделано | Где |
| --- | --- | --- |
| 1. Долг | сверен с `reports/REPORT-AUDIT-MINT-FIX.md:324-336` (AUDIT-MINT-FIX-02): `claim = period-1` без клампа, `accrual_window` в JSON не используется | `mobile/query.go:936-942` на `main` (в аудите и мандате — `:929`, нумерация сдвинута) |
| 2. Фикс | `base` вынесен из блока суммы; `claim` клампится до `base + accrual_window` (кроме `quota_reset`); `amount` по M6 считается от клампа | `mobile/query.go:937-961`, `:972` |
| 3. Тесты | (а) свежий кошелёк, (б) пропуск < окна, (б2) граница `diff = accrual_window`, (в) пропуск 8 периодов, (в2) первый выпуск после отсутствия, воспроизведение долга, `quota_reset` | `mobile/mint_clamp_test.go:126`, `:190`, `:209` |
| 4. Смоук на эмуляторе | **не выполнялся** — опционален по мандату («только если остаётся время окна», «для merge не требуется»); см. §6 | — |
| 5. Отчёт | этот файл | `reports/REPORT-FIX-D2-CLAMP.md` |

## 2. Эталонная семантика ядра (сверено побуквенно)

`core/ledger/validate.go:603-657` (`applyMint`, M4/M5/M6; ниже — строки 624-654):

```go
base := w.LastClaimedPeriod
if base == 0 {
    base = w.ActivatedPeriod - 1
}
diff := mp.Period - base
// M4: mp.Period >= period (период txNow) → E_MINT_PERIOD
//     mp.Period <  w.ActivatedPeriod     → E_MINT_PERIOD
if prof.IssuanceMode != wire.IssuanceQuotaReset && diff > prof.AccrualWindowPeriods {
    return reject("E_ACCRUAL_WINDOW")     // validate.go:650
}
want := prof.IssuanceAmount * diff        // M6
```

Отсюда допустимый максимум заявляемого периода:

```
claim_max = base + accrual_window,  base = last_claimed, либо activated_period − 1 при last_claimed = 0
```

**Важно:** кламп — именно `base + accrual_window`, а не `last + accrual_window − 1`
из примера в мандате. Ядро отвергает только `diff > accrual_window`
(`validate.go:649`), т.е. `diff == accrual_window` принимается; пример в
мандате был бы на единицу строже ядра и оставил бы один достижимый период
недоступным из UI. Мандат требует «взять окончательную формулу из кода
ядра, чтобы мост и ядро совпадали побуквенно» — взята из ядра. То же
подтверждает `reports/REPORT-AUDIT-MINT-FIX.md:331-336` («протокол
допускает восстановление заявкой `last + accrual_window`», «клампить
`claim` до `last + accrual_window`»).

`quota_reset` — исключение: M5 для него не применяется (накопления нет,
заявляется один период), поэтому кламп не ставится — иначе `accrual_window = 0`
обнулял бы claim и квота не выпускалась бы вовсе.

## 3. Формула до/после

**До** (`mobile/query.go:936-942` на `main`, `c7cfb02`):

```
claim = period − 1
если claim <= last_claimed: claim = 0
amount = issuance × (claim − base)          # M6 от неклампленного claim
```

**После** (`mobile/query.go:937-964`):

```
base  = last_claimed, при last_claimed = 0 → base = activated_period − 1
claim = period − 1
если claim > 0 и режим ≠ quota_reset:
    claim = min(claim, base + accrual_window)     # кламп M5
если claim <= last_claimed: claim = 0
amount = issuance × (claim − base)                # M6 от клампа
```

`base` считается один раз и используется и клампом, и суммой — расхождения
между «периодом к заявлению» и «суммой к заявлению» больше нет. Формула
повторяет и uint64-перенос ядра при `activated_period = 0` (там арифметика
ядра даёт `claim_max = accrual_window − 1` — мост даёт то же).

## 4. Тесты и результаты

`mobile/mint_clamp_test.go` (пакет `tnp`, песочница с малым PoW, без сети).
Стенд: genesis в прошлом так, что живые часы узла стоят в периоде `period`;
кошелёк регистрируется через мост, затем состояние `activated/last/status`
инъектируется в реестр — тот же приём, что `injectActive` в core-тестах
(`core/ledger/mint_window_prod_test.go:62`), поскольку полный vouch-путь для
проверки клампа не нужен.

| Тест | Сценарий | claim | amount | Вердикт ядра |
| --- | --- | --- | --- | --- |
| `TestMintClampClaimByAccrualWindow/свежий_кошелёк` | (а) `activated=1, last=0, period=2` | 1 (= period−1, не изменился) | 1000 | accept/OK |
| `.../пропуск_меньше_окна` | (б) `last=1, period=5` (пропуск 3 < 6) | 4 (= period−1) | 3000 | accept/OK |
| `.../пропуск_равен_окну` | (б2) `last=1, period=8` (`diff = 6`) | 7 (= base+6) | 6000 | accept/OK |
| `.../пропуск_больше_окна` | (в) `last=1, period=10` (пропуск 8) | 7 (= base+6, до фикса 9) | 6000 | accept/OK |
| `.../первый_выпуск_после_пропуска` | (в2) `last=0, activated=1, period=10` | 6 (= base+6, до фикса 9) | 6000 | accept/OK |
| `TestMintClampOldClaimRejected` | долг D2: `claim = period−1 = 9` | 9 | 8000 | **reject/E_ACCRUAL_WINDOW** |
| `TestMintClampQuotaResetNoClamp` | `quota_reset`, `accrual_window = 0` | 9 (= period−1, клампа нет) | issuance | — |

Проверки в каждом кейсе (а)–(в2): `current_period`, `accrual_window_periods = 6`,
`claim_period`, `claim_amount`, `open = true` (в песочнице окно открыто весь
период); затем **вердикт живого ядра**, а не повтор формулы в тесте:
подача `claim + 1` (если `claim + 1 < period`) обязана дать
`reject/E_ACCRUAL_WINDOW`, подача `claim` — `accept/OK`, после неё
`last_claimed_period = claim`. Так тест фиксирует, что мост предлагает
именно максимум, допустимый ядром.

Обратный эксперимент (регрессионный страж): на коде до фикса
(`git show HEAD~1:mobile/query.go`) тесты красные —
`claim_period = 9, хочу 7` и `claim_period = 9, хочу 6`; с фиксом зелёные.

## 5. Прогоны

```
go build ./...                       rc=0
go vet ./mobile/                     rc=0
gofmt -l mobile/                     пусто
go test ./mobile/... -count=1        ok  mobile  2.897s
go test ./mobile/ -run TestMintClamp -count=1 -v
    --- PASS: TestMintClampClaimByAccrualWindow (1.40s)   [5 подкейсов]
    --- PASS: TestMintClampOldClaimRejected (0.26s)
    --- PASS: TestMintClampQuotaResetNoClamp (0.11s)
go test ./...                        EXIT=0 (15 пакетов ok)
flutter analyze (app/)               No issues found!  rc=0
flutter test (app/)                  00:13 +57: All tests passed!  rc=0
```

Flutter-часть не менялась (правка Go-only), прогоны сделаны по правилу
«перед финальным коммитом — зелёные»; эмуляторный прогон — см. §6.

## 6. Что не делалось и почему

- **Смоук на эмуляторе (шаг 4)** — опционален по мандату и для merge не
  требуется. Сценарий «пропуск ≥ accrual_window» в UI воспроизводится
  только инъекцией состояния кошелька в реестр (регистрация всегда даёт
  `activated = текущий период`), т.е. экранный смоук показал бы лишь
  неизменный claim свежего кошелька — ценности сверх Go-тестов на живом
  ядре не добавляет. Скриншотов `reports/fix-d2-clamp/` нет.
- **`core/` и протокол** — не менялись (мандат). `CheckMintChallengeWindow`
  (`core/ledger/validate.go:188-196`) к клампу отношения не имеет: это M7
  (окно слота), клампится M5; формула взята из `applyMint`
  (`validate.go:603-657`), как и требует п. 2 мандата по существу.

## 7. PENDING (за владельцем, вне темы)

1. **Claim свежего кошелька.** Для кошелька, активированного в текущем
   периоде (`activated = period`, `last = 0`), мост отдаёт
   `claim = period − 1`, а ядро на такую подачу отвечает `E_MINT_PERIOD`
   (M4: `mp.Period < ActivatedPeriod`) и `claim_amount = 0` (M6: `diff = 0`).
   Это дотема (не введено D2 и не покрыто AUDIT-MINT-FIX-02), поведение
   фиксом сознательно не менялось — п. 3(а) мандата требует «claim не
   изменился». Кандидат в отдельную находку: клампить claim снизу
   `activated_period` (или отдавать `claim = 0` при `period − 1 < activated`).
2. **AUDIT-MINT-FIX-05** (claim при sync) и AUDIT-MINT-FIX-03 (tooling
   прогона) — отдельные темы по §10 отчёта аудита, здесь не трогались.

## 8. Воспроизводимость

```bash
git fetch origin && git checkout -b fix-d2-clamp origin/main    # c7cfb02
gofmt -l mobile/                                                # пусто
go build ./... && go vet ./mobile/
go test ./mobile/... -count=1
go test ./mobile/ -run TestMintClamp -count=1 -v
# обратный эксперимент (тесты должны быть красными):
cp mobile/query.go /tmp/q.fixed && git show HEAD~1:mobile/query.go > mobile/query.go
go test ./mobile/ -run TestMintClamp ; cp /tmp/q.fixed mobile/query.go
```
