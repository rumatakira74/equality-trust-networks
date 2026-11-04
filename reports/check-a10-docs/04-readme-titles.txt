# АРТЕФАКТ 04 — README, титулы, версия спеки, DISCLAIMER (blind)
# дата: 2026-11-04T00:52:44Z  session: 52e752d3-b60f-c3a3-61f5-f8b251bf51de

## 1. README 7.7 против спеки 7.7 (RU) — побайтовое сравнение
ИТОГ: разделы 7.7 идентичны побайтово

## 2. README 7.6 против спеки 7.6 (RU)
5,6c5,6
< - **Устройства:** аппаратная аттестация до встроенных корней Google с привязкой листа к регистрируемому ключу (v9.12): программные фермы и эмуляторы отсечены, рут на таких устройствах сжигает Knox. Но одно устройство может выпустить несколько attested-ключей, поэтому «ферма = ферма дорогих устройств» выполняется не автоматически: несколько кошельков на устройстве остаются возможны, и барьер держат поручительства и стоимость (7.1).
< - **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч (настраивается полем 51 `mint_challenge_window_hours`). Слот `period_start + H(address ‖ period) mod period_seconds` публично вычислим заранее — приложение может показать его пользователю; включение `hash(чекпоинта предыдущей эпохи)` в формулу — открытый вопрос (PENDING за владельцем; реализация — `H(address ‖ period)`). ✅ Якорь слота (№39, этап 9): периоды сети — от genesis-времени (раздел 9.2), не от 1970 г.; приведение mintChallengeSlot (подэтап 8.0) подтверждено симуляцией 8.1-2 — окна у якоря эры 2: выпуски участников 28/28, вне окна E_POW_HARD, приложение блокирует выпуск в UI. ✅ v9.13 (этап фикса времени): окно сверяется с `timestamp_claim` самой tx (как и все часозависимые правила, 9.2); слот масштабируется длиной периода `period_seconds` (поле 61, 0 — конвенция 30 сут). ✅ v9.14 (тема mint-fix, решение владельца 03.11.2026): якорь слота — ТЕКУЩИЙ период выпуска (период txNow), не заявленный: заявка периода P возможна только после конца P (10.1), поэтому окно слота заявленного периода пересекалось с допустимым временем подачи лишь при слоте в последних 48 ч периода — достижимость 1/15 за период, шесть пропусков подряд давали поглощающее состояние (симуляция A9: честные −91,2 % дохода за 60 мес.). Слот текущего периода лежит внутри него: пересечение окна ±48 ч с текущим периодом всегда ≥ 48 ч — у каждого кошелька каждый период есть достижимое окно, пропущенный период компенсируется в пределах `accrual_window` (10.1). Достижимость первых заявляемых периодов посевного круга при коротких `period_seconds` (квоты с периода 1 у genesis, блокер 2 прогона 12-testnet) задаёт поле 51: окно шаблона ±48 ч их не накрывает, тестовая сеть задаёт большее окно. PoW memory-hard (Argon2id-подобный, параметры под средний телефон): ограничивает выигрыш GPU/ASIC, но полного паритета не даёт.
---
> - **Устройства:** аппаратная аттестация (программные фермы и эмуляторы отсечены, рут на таких устройствах сжигает Knox); но одно устройство может выпустить несколько attested-ключей — несколько кошельков на устройстве возможны, барьер держат поручительства и стоимость (7.1).
> - **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч. Слот `period_start + H(address ‖ period) mod period_seconds` публично вычислим заранее (включение `hash(чекпоинта предыдущей эпохи)` в формулу — открытый вопрос); якорь слота — текущий период выпуска (v9.14): окно ±48 ч всегда пересекается с текущим периодом на ≥ 48 ч — у каждого кошелька каждый период есть достижимое окно. PoW memory-hard (Argon2id-подобный, параметры под средний телефон): ограничивает выигрыш GPU/ASIC, но полного паритета не даёт.
ИТОГ: 7.6 в README — сокращённая редакция (как и до правки); см. разбор в отчёте

## 3. README-ENG.md — тронут?
(не тронут — ожидаемо, PENDING-3)
строки README-ENG.md со старыми числами:
364:- **The security metric:** "the farm's cost per wallet" (device + electricity + time + the price of a vouch: social risk, limited slots, 10% of the first ye
374:2. **The farm's share is temporary.** The peak is ~40% in month 27; by month 60 the honest network overtakes the farm (12.1%). A growing honest network is i
375:3. **K = 3 confirmed.** The farm's share at 60 months: K = 2 → 9.6%, K = 3 → 12.1%, K = 4 → 15.2%, K = 5 → 23.2%, while honest growth at K = 5 is 2.2 times 
377:5. **A seed circle ≥ 300.** The farm's share at 60 months: seed 50 → 21.4%, 100 → 12.1%, 300 → 5.4%, 1000 → 5.4% (the effect saturates after ~300). Recommen
378:6. **The farm's scale changes the share, not the economics.** A budget of 5 devices/month → a 3.3% farm share, 100 devices/month → 40.5%, yet the break-even
379:7. ⚠️ **Honest about the payback time.** The farm's break-even price falls monotonically and crosses $0.01 in month 38: at a unit price of $0.01 the base fa

## 4. DISCLAIMER.md — тронут? числовые утверждения?
(не тронут)
(числовых утверждений о PoW/ферме/окупаемости нет)

## 5. Титулы и версия спеки
8:Версия документа: 9.15. Рабочее название — Equality Project.
9:Дата документа: 4 ноября 2026 г.
11:Document version: 9.15. Working title — Equality Project.  
12:Document date: November 4, 2026.  
запись в истории версий RU:
46:Версия 9.15 (тема a10-docs, документация к факту после A9 — PENDING-2 blind-аудита A9): раздел 7.7 приведён к фактам 
запись в истории версий ENG:
41:Version 9.15 (the a10-docs topic, documentation to the facts after A9 — PENDING-2 of the A9 blind audit): Section 7.7
титул README.md:
12:Версия документа: 9.3. Рабочее название — Equality Project.  
13:Дата документа: 18 октября 2026 г.  
титул README-ENG.md:
15:Document version: 9.3. Working title — Equality Project.  
16:Document date: October 18, 2026.  

## 6. Слот челленджа в README 7.6 и спеке 7.6 (по коду main)
README.md:356:- **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч. Слот `period_start + H(address ‖ period) mod period_seconds` публично вычислим заранее (включение `h
docs/KNOWLEDGE-3.md:108:- **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч (настраивается полем 51 `mint_challenge_window_hours`). Слот `period_start + H(address ‖ pe
docs/KNOWLEDGE-ENG-3.md:109:- **The issuance challenge:** reinforced PoW in a random slot within the period with a ±48 h window (configured by field 51 `mint_challenge_window_hours`). The slot `period
356:- **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч. Слот `period_start + H(address ‖ period) mod period_seconds` публично вычислим заранее (включение `hash(чекпои
README.md:356:- **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч. Слот `period_start + H(address ‖ period) mod period_seconds` публично вычислим заранее (включение `h
docs/KNOWLEDGE-3.md:108:- **Челлендж выпуска:** усиленный PoW в рандомном слоте внутри периода с окном ±48 ч (настраивается полем 51 `mint_challenge_window_hours`). Слот `period_start + H(address ‖ pe

## 7. Остаточные устаревшие числа в docs/ (вне правки темы)
docs/farm-attack-economics-ENG.md:22:- **The maximum pit is ~$36k in month 23.** The farm only turns a profit in month 38, having invested ~$90k by then.
docs/farm-attack-economics-ENG.md:28:**Bottom line.** The farm is not "a profitable venture with a long start", but a speculative bet: tens of thousands of dollars, three years of losses, a publicly visible scheme, and a
docs/farm-attack-economics.md:39:  на 38-м месяце, вложив к тому времени ~$90 тыс.
docs/farm-attack-economics.md:57:окупается на 38-м месяце» читается как «нужно 3 года держать в минусе десятки
docs/KNOWLEDGE-4.md:89:✅ Зафиксировано (этапы 1.2, 6–7; floor профиля 5 — v9.9): демередж 1%/мес, floor = 3 месячных выпуска (профиль 7 — 3000, профиль 5 — 30); сжигаемая комиссия `fee_min` = 0,01, rate ∈ [0,5; 2]%; P = 
