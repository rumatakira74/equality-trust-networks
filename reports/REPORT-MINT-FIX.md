# REPORT-MINT-FIX — фикс окна челленджа mint (дефект M7, вариант A) + protocolVersion-bump

Ветка: `mint-fix` (от `main` d5ab36e). Решение владельца 03.11.2026: дефект подтверждён
независимо трижды (симуляция A9, blind-аудит A9, внешний аудит Sonnet).
Коммиты ветки: 171a7df (фикс + bump + пути), 977d5b0 (тесты), 30a6626 (спека), отчёт.

## 1. Механика дефекта

Проверка M7 — `core/ledger/validate.go`, `applySigPowTime` (до фикса: строка 157):

```go
if tx.Type == wire.Mint && !l.Opts.SkipMintChallengeWindow {
    mp := tx.Payload.(*wire.MintPayload)
    if !l.CheckMintChallengeWindow(tx.Wallet, mp.Period, l.tt.now) {   // mp.Period!
        return reject("E_POW_HARD")
    }
}
```

Слот (`mintChallengeSlot`, validate.go:206): `period_start + H(addr‖period) mod period_secs`,
`period_start = genesis_time + period × period_secs` (периоды календарные от genesis, №39).
Слот равномерен внутри периода (при 30 сут = 720 ч). По M4 (validate.go, applyMint:
`mp.Period >= l.tt.period → E_MINT_PERIOD`) заявка периода P возможна только после конца P.
Поэтому окно ±48 ч вокруг слота заявленного периода P пересекается с допустимым временем
подачи только при слоте в последних 48 ч периода P: достижимость 48/720 = **1/15 = 6,67 %**
за период. Слоты независимы по периодам: 6 пропусков подряд с вероятностью (14/15)^6 ≈ 66 %
дают diff > accrual_window (6) навсегда — **поглощающее состояние**: кошелёк навсегда теряет
выпуск. Симуляция A9 (репо eq): честные −91,2 % дохода за 60 мес., к 60-му месяцу заблокированы
53 % честных кошельков; автоматизированная ферма не страдает — слот `SHA256(addr‖period)`
публично вычислим заранее.

## 2. Фикс (вариант A)

`core/ledger/validate.go:165` — якорь слота заменён на текущий период выпуска:

```go
if tx.Type == wire.Mint && !l.Opts.SkipMintChallengeWindow {
    if !l.CheckMintChallengeWindow(tx.Wallet, l.tt.period, l.tt.now) {
        return reject("E_POW_HARD")
    }
}
```

`l.tt.period = periodAt(txNow)`, txNow = `tx.TimestampClaim` (v9.13), периоды арифметически
от genesis (`core/ledger/ledger.go:169`, `periodAt`). Комментарии в `applySigPowTime`,
`CheckMintChallengeWindow` (validate.go:180) и `mintChallengeSlot` приведены к фактическому
поведению; из комментария `CheckMintChallengeWindow` убрана неверная ссылка на
checkpoint_hash (фактический хеш — `hashThree(addr, period)`, включение чекпоинт-хеша —
PENDING, см. п. 7).

### Доказательство свойства окна

После фикса слот лежит внутри ТЕКУЩЕГО периода C: slot ∈ [start, start+P), P = 720 ч,
W = 48 ч (окно). Пересечение [slot−W, slot+W] ∩ [start, start+P):

| положение слота | пересечение | длина |
|---|---|---|
| slot ∈ [start, start+W) | [start, slot+W] | slot−start+W ≥ W |
| slot ∈ [start+W, start+P−W] | [slot−W, slot+W] | 2W ≥ W |
| slot ∈ (start+P−W, start+P) | [slot−W, start+P] | W+(start+P−slot) > W |

Пересечение **всегда ≥ 48 ч** — у каждого кошелька каждый период есть достижимое окно
подачи (txNow внутри текущего периода попадает в окно на отрезке ≥ 48 ч). Поглощающее
состояние исчезает: пропущенный период компенсируется M6 (amount = issuance × diff)
в пределах accrual_window; осознанный пропуск 6+ периодов по-прежнему сгорает
(M5 не менялся). Проверяется тестом-свойством (a): 64 адреса × 8 периодов = 512 проверок,
все пересечения ≥ 48 ч; для коротких периодов (поле 61, 120 с) окно ±48 ч накрывает текущий
период целиком (24 проверки).

## 3. protocolVersion-bump (2 → 3)

Семантика проверки M7 изменилась — старые и новые узлы разошлись бы в вердиктах.
Проверка C2: `core/ledger/validate.go:65` (`tx.ProtocolVersion != l.State.Net.ProtocolVersion → E_VER`);
в сети — дублируется в `net/gossip.go:336` (cheapCheck, errForeignVersion). Обновлены
все места, где версия зашита:

| место | было → стало |
|---|---|
| `core/params/params.go:210` (TestGenesis — дефолт всех тестов) | 2 → 3 |
| `cmd/tnp-testnet/main.go:393` (genesis-шаблон живого прогона) | 2 → 3 |
| `cmd/tnp-testnet/configure.go:158` (конфигуратор сетей) | 2 → 3 |
| `vectors/*.json` (24 вхождения в 20 векторах; подписи/PoW материализуются harness'ом из плейсхолдеров — рассинхрона подписей нет) | 2 → 3 |
| `core/store/bench_test.go:20`, `core/store/sqlite_test.go:22` (фикстуры журнала) | 2/1 → 3 |

Сознательно НЕ поднято: `core/wire/proto_conformance_test.go` (ProtocolVersion: 1 в
нескольких фикстурах) — это тесты бинарного кодирования полей с произвольными значениями,
к сети не привязаны; изменение — шум без эффекта.

**Миграция не нужна:** живых сетей нет, тестовые сети пересоздаются с новым genesis
(как при v9.13 → 2). Зафиксировано здесь.

## 4. Единообразие M7 на всех путях применения

Все пути сходятся к одному механизму `Ledger.Apply` (сеттлемент к txNow → applySigPowTime
с `l.tt.period`), поэтому вердикт один и тот же от txNow:

- **headless**: `applySigPowTime` → `CheckMintChallengeWindow(l.tt.period)` — validate.go:165.
- **net**: `net/node.go:264` — `ledger.New(gp, ledger.Options{})`, окно включено
  (долг этапа 8 закрыт: SkipMintChallengeWindow в сетевом узле нет); gossip-валидатор
  (`net/gossip.go:336`) проверяет версию протокола до ретрансляции, окно — при apply
  тем же ядром; sync/replay — `cons/replay.go:116` (`l.Apply`), тот же вердикт.
- **mobile-мост**: песочница гасит окно легитимным оверрайдом
  `MintChallengeWindowHours = 0` (`mobile/mobile.go`, учебный режим; комментарий приведён
  к постфиксному факту). `MintWindowJSON` (`mobile/query.go:937`) теперь передаёт в
  `QueryMintWindowOpen` ТЕКУЩИЙ период живых часов — UI и apply сходятся к одному
  вердикту. Находка №2 прогона stage11-prep («передавать claim-период») отменена вместе
  с изменением семантики M7; тест `TestMintWindowClaimedPeriod` переписан в
  `TestMintWindowCurrentPeriod` (детерминированный подбор адресов вместо случайных ключей).
- `net/query.go` `QueryMintWindowOpen(addr, period, now)` — сигнатура сохранена
  (период-якорь передаёт вызывающий; комментарий обновлён).
- `cmd/tnp-testnet` `slots`-команда: формула-дубликат не менялась (правило 3, существующее
  дублирование — FINDING-02); комментарий уточнён: для живого прогона запрашивается слот
  текущего периода.
- **sim**: `sim/node.go:112`, `sim/sim.go:206` — `SkipMintChallengeWindow: true` сохранён
  (симуляция — песочница, окно не моделировала и раньше). FINDING-01.

`SkipMintChallengeWindow: true` остался только в: sim (2 места) и тестовых harness'ах
(cons, cmd/tnp-testnet тест, vector harness) — сетевых путей среди них нет.

## 5. Тесты (core/ledger/mint_fix_test.go, все — репродуцируемые)

Данные: ключи `crypto.DeterministicKey`, genesis фиксирован (1_700_000_000), период 30 сут,
окно ±48 ч, малые PoW-таргеты; адреса под условия подбираются детерминированным перебором
seed'ов (без вероятностных допущений в самих проверках).

| # | тест | проверка | результат |
|---|---|---|---|
| (a) | TestMintFixWindowIntersectionProperty | пересечение окна с текущим периодом ≥ 48 ч: 64 адреса × 8 периодов (512) + короткие периоды 8 × 5 × 3 точек | PASS |
| (b) | TestMintFixWindowInOut | txNow = слоту текущего периода → accept; txNow = слот+49 ч (тот же период) → E_POW_HARD | PASS |
| (c) | TestMintFixAccumulationInWindow | last=1, claim=4 (diff=3 ≤ 6) в окне текущего периода 5 → accept, когорты 3000 (M6: 1000×3), last_claimed=4 | PASS |
| (d) | TestMintFixAccrualWindowUnchanged | last=2, claim=9 (diff=7 > 6) в окне → E_ACCRUAL_WINDOW (M5 не изменился) | PASS |
| (e) | TestMintFixLiveReplayDeterminism | журнал из 2 выпусков (периоды 5 и 6) в live-ядро и в свежее (replay = тот же Apply): вердикты и дайджест состояния совпадают; отклонённая вне окна tx не меняет дайджест | PASS |
| (f) | TestMintFixSandboxSkip | SkipMintChallengeWindow: mint вне окна → accept (песочница) | PASS |
| (g) | TestMintFixOldProtocolVersion | валидная tx с protocolVersion 2 → E_VER | PASS |
| (h) | TestMintFixPreFixVector | два вектора (ниже) | PASS |

(h), дофиксные векторы — txNow через час после конца заявленного периода 4 (текущий 5):
- **поздний слот** (смена вердикта): слот заявленного периода в последних 24 ч — при
  дофиксной семантике окно заявленного периода накрывало txNow (было бы accept), слот
  текущего периода ≥ 49 ч от txNow → теперь E_POW_HARD. Вектор ловит откат фикса.
- **срединный слот** (буквальный вектор мандата): слот заявленного периода в средней
  трети периода → E_POW_HARD; вердикт определяется слотом текущего периода, заявленный
  в проверке больше не участвует.

Мобильный мост: `TestMintWindowCurrentPeriod` — окно текущего периода 2 открыто/закрыто
по слоту периода 2 (claim_period=1 в обоих случаях); сверка `open == QueryMintWindowOpen(addr, 2, now)`.

## 6. Спека (v9.13 → v9.14)

- `docs/KNOWLEDGE-3.md`, раздел 7.6 (хирургически, один пункт «Челлендж выпуска»):
  слот от текущего периода выпуска, окно ±48 ч, слот публично вычислим заранее —
  приложение может показать его пользователю; формула приведена к факту кода
  `H(address‖period)`; включение hash(чекпоинта предыдущей эпохи) — примечание об
  открытом вопросе (PENDING за владельцем). Добавлена запись ✅ v9.14 с механикой дефекта.
- `docs/KNOWLEDGE-ENG-3.md` — то же в переводе.
- `docs/KNOWLEDGE.md` / `docs/KNOWLEDGE-ENG.md`: версия документа 9.13 → 9.14, дата
  3 ноября 2026, запись «Версия 9.14» в истории версий (RU + ENG).
- Остальные разделы не тронуты.

## 7. PENDING (за владельцем)

1. **Чекпоинт-хеш в формуле слота**: спека прежде упоминала
   `H(address ‖ period ‖ hash(чекпоинта предыдущей эпохи))`; код всегда считал
   `H(address ‖ period)` (hashThree). Вопрос включения чекпоинт-хеша — открыт,
   НЕ правился (мандат п. 5: формулу слота не менять). Спека и комментарии приведены
   к факту кода с примечанием PENDING.
2. **UX-показ слота пользователю**: слот публично вычислим заранее, приложение может
   показать его; сам показ — вне мандата, не делался. Точный момент слота по-прежнему
   не раскрывается сетевыми запросами (net/query.go, PENDING отчёта этапа 7).

## 8. Расхождения со спекой (до фикса)

- Спека 7.6 (v9.13): «слот вычисляется от claim … заявляемый период обязан быть достижим
  в claim-окне» — описывала дофиксную семантику (якорь mp.Period); сама эта семантика и
  была дефектом M7. Исправлено в v9.14.
- Спека 7.6 утверждала «слот … непредсказуем заранее» — неверно даже до фикса:
  `SHA256(addr‖period)` детерминирован и публично вычислим (что и делало ферму
  неуязвимой к дефекту). Приведено к факту: «публично вычислим заранее».
- Комментарий `CheckMintChallengeWindow` в коде упоминал checkpoint_hash —
  расхождение код/комментарий; исправлено.

## 9. Прогоны (полный локальный набор)

- `gofmt -l .` — пусто.
- `go vet ./...` — чисто (GOMAXPROCS=6).
- `go test ./...` — зелёные все пакеты (core/ledger 43 c, net 60 c, sim 124 c и т.д.).
- `go test -race ./core/... ./cons/ ./mobile/ ./net/` — чисто (ledger 247 c, net 119 c).
- Память во время прогонов ≥ 7,6 ГБ доступной из 15,5 ГБ — стоп-условий (<2 ГБ) не было.
- Процессы: после прогонов pgrep — tnp/gradle/qemu/go-toolchain не осталось
 (goa-daemon и пр. — системные демоны GNOME, не относятся).

CI: статусы в сессии не ждались — CI pending.

## 10. Коммиты ветки mint-fix

- 171a7df — фикс M7 + protocolVersion 2→3 + единообразие путей (mobile/net/cmd).
- 977d5b0 — тесты (a)–(h).
- 30a6626 — спека 7.6 (RU/ENG) + версия 9.14 + история версий.
- (этот коммит) — отчёт и FINDINGS.

Пуш ветки — после полного локального набора (выполнен). Merge в main — только по
подтверждению владельца.
