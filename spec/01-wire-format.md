# 1. Wire-формат транзакций

Источник: разделы 9.1–9.3, 7.2, 7.6, 8.3, 10.1–10.3, 12.1, 13.
Схемы: `proto/core.proto`, `proto/transactions.proto`.

## 1.1. Конверт

Все транзакции — сообщение `tnp.v1.Transaction`:

```
Transaction {
  bytes  network_id        = 1;   // 32 байта, из genesis (решение D1)
  bytes  wallet            = 2;   // 32 байта, адрес отправителя
  uint64 seq               = 3;   // строго last_seq + 1
  bytes  prev              = 4;   // 32 байта, txid предыдущей; 0^32 для seq = 1
  TxType type              = 5;
  uint64 timestamp_claim   = 6;   // Unix sec, UTC, допуск ±30 мин
  uint32 protocol_version  = 7;   // раздел 13
  PowProof pow             = 8;
  bytes  signature         = 9;   // DER ECDSA P-256
  oneof  payload           = 10..24; // по type, см. 1.5
}
```

Правила:

- **Первая транзакция цепочки** — всегда `REGISTER`: `seq = 1`, `prev = 0^32`.
- `seq` строго +1 (9.1); нарушение — `E_SEQ`. `prev` — хеш предыдущей
  транзакции (9.1); нарушение — `E_PREV`.
- `type` обязан совпадать с активной веткой `payload`; иначе `E_TYPE`.
- Ориентир размера транзакции ~250 байт (12.5).

## 1.2. Вычисляемые величины

Не передаются в сообщении, вычисляются каждым узлом:

```
wallet_fingerprint = SHA-256(attested_public_key || protocol_id)   // 7.2
address            = f(wallet_fingerprint)            // PENDING: вопрос №2
signing_preimage   = canonical(Transaction, signature = пусто)
signing_hash       = SHA-256(signing_preimage || pow.nonce)
txid               = SHA-256(canonical(Transaction))
```

- Подпись `signature` — ECDSA P-256 над `signing_hash`; неверна — `E_SIG`.
- PoW включается в `signing_hash` → подпись привязана к конкретному решению
  челленджа; замена nonce без переподписи невозможна.
- `txid` вычисляется по полной подписанной транзакции → `prev` образует
  цепочку, а эквивокация (две транзакции с одним `seq`) даёт два разных `txid`
  и доказуема (7.5).

## 1.3. PoW

Разделы 7.6, 9.3, 17: лёгкий PoW — на каждую транзакцию; усиленный
memory-hard (Argon2id-подобный) — на `register` и выпуск (`mint`).

```
PowProof { PowAlgorithm algorithm = 1; bytes nonce = 2; }
```

- **POW_LIGHT** (все типы, кроме `register`/`mint`): принимается, если
  `SHA-256(signing_preimage || nonce) < target_light`.
- **POW_ARGON2ID** (`register`, `mint`): принимается, если
  `Argon2id(password = nonce, salt = signing_preimage,
  m = argon2_memory_kib, t = argon2_iterations, p = argon2_parallelism,
  len = 32) < target_hard`.
- Таргеты и параметры Argon2id — `ProfileParams.pow_params`; значения
  «под средний телефон» выбираются на этапах 1.2/2 — PENDING: вопрос №9.
- Повторяющийся `android_id` повышает таргет `register`
  (`repeat_android_id_target`) — слабый сигнал, не основание отказа (7.2).
- Для `mint` челлендж привязан к непредсказуемому слоту (7.6):
  `slot = H(address || period || checkpoint_hash(prev_epoch))`, окно ±48 ч
  (`mint_challenge_window_hours`). PoW принимается, только если сетевое время
  попадает в окно слота заявляемого периода; иначе `E_POW_HARD`.
  Для multi-period `mint` (нагон пропущенных месяцев) слот вычисляется по
  старшему заявляемому периоду — PENDING: вопрос №10.

## 1.4. Время

- `timestamp_claim` — заявленное время; допуск ±30 мин (9.2, 17) — `E_TIME`.
- На период выпуска `timestamp_claim` не влияет: период определяется эпохой
  публикации и полем `period` (9.2).
- Сетевое время не может сдвигаться между чекпоинтами более чем на заданный
  предел (9.2); предел — параметр консенсуса этапа 5 — PENDING: вопрос №16.

## 1.5. Payload по типам

### 1.5.1. `REGISTER` (7.2, 7.3, 11)

```
RegisterPayload {
  bytes   attested_public_key   = 1;
  bytes   attestation_chain     = 2;  // DER X.509, лист + промежуточные
  bytes   attestation_challenge = 3;
  string  android_id            = 4;
  repeated bytes personal_guardians = 5; // ≤ 3 адресов (решение D4)
  bool    synthetic             = 6;
}
```

- `walletFingerprint = SHA-256(attested_public_key ‖ protocolId)` (7.2);
  один фингерпринт = один адрес; повторная `register` с тем же фингерпринтом
  и другим адресом отвергается (7.2). Индекс узла: `фингерпринт → адрес`.
- Цепочка аттестации проверяется локально каждым узлом по корневым
  сертификатам Google, зашитым в APK: аппаратный уровень (StrongBox/TEE),
  `verifiedBootState = Verified`, `deviceLocked = true`, не просроченный
  уровень патча (7.2).
- `attestation_challenge` — значение `setAttestationChallenge`; узел проверяет
  связку с ключом и сетью. Точная формула — PENDING: вопрос №23.
- `personal_guardians` — назначение гардианов заранее, при создании кошелька
  (11); максимум 3 (`guardian_personal_count`). Обновление списка — вне
  закрытого перечня типов — PENDING: вопрос №6.
- `synthetic = true` допустим только при `sandbox = true` (6.5) — иначе `E_SBOX`.

### 1.5.2. `VOUCH` (7.4)

```
VouchPayload { bytes candidate = 1; uint64 epoch = 2; }
```

Подписывается поручителем; содержит адрес кандидата и эпоху (7.4).
Для активации кандидата нужно K = 3 `vouch` от разных кошельков.
Условия к поручителю и эффекты — spec/03, раздел 3.3; слоты — spec/06.

### 1.5.3. `MINT` (10.1, 7.6)

```
MintPayload { uint64 period = 1; uint64 amount = 2; }
```

- `period` — номер календарного месяца; «1-е число месяца» = первая эпоха,
  начавшаяся после 00:00 UTC 1-го числа по консенсусному времени (9.2).
- Правила: `period` строго больше последнего заявленного; `amount = issuance_amount × (period − last_claimed_period)`; узел пересчитывает сам — `E_MINT_AMOUNT`.
- Первый `mint` заявляет завершённые периоды, не раньше периода активации
  («без ретро-начислений», 10.1). Толкование «первый mint = period − 1» —
  PENDING: вопрос №24.
- Пропущенные месяцы начисляются в пределах `accrual_window` (6 мес. по
  умолчанию); сверх окна — сгорают, деньги не блокируются (10.1) —
  `E_ACCRUAL_WINDOW`.
- Доля R = 10% из первых 12 выпусков делится поровну между K поручителями
  (7.4); источник — выпуск самого поручаемого, суммарная эмиссия не растёт.
  Выплата прекращается с эпохи финализации `evidence` против поручаемого (7.5).

### 1.5.4. `TRANSFER` (10.2, 7.6, 8.3) — решение D2

```
TransferPayload {
  oneof action {
    TransferSend     send          = 1;
    TransferCancel   cancel        = 2;
    ThresholdUpdate  set_threshold = 3;
  }
}
TransferSend     { bytes recipient = 1; uint64 amount = 2; }
TransferCancel   { uint64 pending_seq = 1; }
ThresholdUpdate  { uint64 new_threshold = 1; }
```

- **send**: получатель получает ровно `amount`; комиссия
  `fee = max(fee_min, rate × amount)` сжигается сверх суммы (10.2).
  `rate` адаптивна, пересчитывается на границе эпохи детерминированно из
  данных чекпоинтов (10.2); границы `[rate_min, rate_max]` — этап 1.2.
- Перевод выше порога P → статус `pending` до `network_time + D` (D = 48 ч,
  фиксированная); в течение D владелец отменяет подписью с биометрией (8.3).
  Подпись с биометрией — платформенное требование (`signing_auth`), протокол
  проверяет только подпись.
- **cancel**: отмена `pending`-перевода; сумма разблокируется.
- **set_threshold**: смена личного P в пределах `P_max`; повышение вступает
  через 48 ч, понижение — сразу (8.3). Кодирование — PENDING: вопрос №3 (send/
  cancel — вопрос №4).
- Мораторий: `transfer` доступен через `transfer_moratorium_days` (7–30 дней,
  по умолчанию 14 — этап 1.2) после `register` (7.6, 17). Свежевыпущенные
  средства переводимы не сразу («часть моратория», 10.1): окно
  `fresh_transfer_window_periods` (по умолчанию 3 мес., вопрос №11 закрыт
  данными этапа 1.2) — выпуск периода t переводим с t + window, пер-юнит
  когортами, списание из старейших переводимых (см. spec/03 T8).

### 1.5.5. `MIGRATE` (11) — решение D3

```
MigratePayload {
  oneof action {
    MigratePropose propose = 1;  // публикуется в цепочке НОВОГО кошелька
    MigrateCancel  cancel  = 2;  // публикуется в цепочке СТАРОГО кошелька
  }
}
MigratePropose {
  bytes old_address = 1;
  repeated GuardianApproval approvals = 2;
  uint64 nonce = 3;
}
MigrateStatement { network_id, old_address, new_address, nonce, epoch }
GuardianApproval { bytes guardian = 1; bytes statement_sig = 2; }
MigrateCancel { bytes new_address = 1; uint64 propose_seq = 2; }
```

- Порог: 2 из 3 личных гардианов; при участии публичных — 3 из 5, любое
  сочетание (11).
- Новый кошелёк проходит ту же аттестацию, что при `register` (отдельная
  транзакция `REGISTER` в цепочке нового кошелька).
- Задержка 14 дней (`migrate_delay_days`); в течение неё старый ключ отменяет
  миграцию (`cancel`). После задержки без отмены: переносятся возраст, история
  выпусков, `period` и поручительства; старый фингерпринт навсегда в чёрном
  списке; награда публичным гардианам — из новой эмиссии, только подписавшим,
  только после задержки без отмены (11).

### 1.5.6. `EVIDENCE` (7.5) — решение D9

```
EvidencePayload {
  Transaction tx_a = 1;
  Transaction tx_b = 2;
  EvidenceKind kind = 3;  // EQUIVOCATION | DOUBLE_MINT
}
```

- Закрытый перечень доказательств: эквивокация (одинаковые `wallet` и
  `seq`/`prev`, разное содержание) и двойной выпуск за один период — частный
  случай (7.5). Всё остальное доказательством не является.
- Проверки каждого узла: обе подписи верны, кошелёк один, `seq`/`prev`
  совпадают, хеши разные, обе транзакции позже `register` (7.5). Проверка не
  зависит от времени и порядка прихода.
- Последствия (мягкая схема, 7.5 — PENDING: вопрос №5 = вопрос 5 раздела 18):
  постоянная метка `flagged`; обе конфликтующие ветки не принимаются, остаётся
  ветка финализации; выпуск приостанавливается на N месяцев (N — PENDING:
  вопрос №12), при повторе — навсегда; поручители перестают получать долю;
  денежных штрафов и потери слотов нет; награды за подачу нет.

### 1.5.7. `GUARDIAN_OFFER` (11)

```
GuardianOfferPayload { }  // пусто: offer = факт публикации
```

Кошелёк `active` с возрастом ≥ 12 мес. объявляет себя публичным гардианом.
Лимит подопечных G_max = 20. Отзыв предложения не определён — PENDING:
вопрос №25.

### 1.5.8. `POOL_CREATE` / `POOL_DONATE` / `POOL_CLOSE` / `POOL_CLAIM` (6.3)

```
PoolCreatePayload {
  bytes recipient = 1; uint64 goal = 2; uint64 deadline_epoch = 3;
  ReleaseMode release_mode = 4;  // ON_DEADLINE | ON_GOAL | CONTINUOUS
  RefundMode  refund_mode  = 5;  // IF_UNMET | NONE
}
PoolDonatePayload { bytes pool_id = 1; uint64 amount = 2; }
PoolClosePayload  { bytes pool_id = 1; }
PoolClaimPayload  { bytes pool_id = 1; }
```

- `pool_id` = txid транзакции `POOL_CREATE` (решение D5): типизированный счёт
  без адреса.
- Прогресс пула публичен (6.3). `pool_claim` — равные доли `active`-кошелькам.
- Право и момент закрытия не определены — PENDING: вопрос №26.

### 1.5.9. `REDEEM` / `RECEIPT` (6.2) — решение D10

```
RedeemPayload {
  bytes provider = 1; string resource_id = 2; TimeSlot slot = 3; uint64 amount = 4;
}
ReceiptPayload {
  bytes redeem_txid = 1; uint64 amount = 2; string resource_id = 3; TimeSlot slot = 4;
}
```

- `redeem` — перевод единицы кошельку-поставщику; адрес поставщика обязан
  быть указан в параметрах сети (6.2). Для физического доступа получатель
  обязан дождаться подтверждения сети (12.4).
- `receipt` публикуется поставщиком в своей цепочке и подтверждает погашение.
- Квота (`quota_reset`) не накапливается и сгорает в конце периода (6.2).

### 1.5.10. `CONTRIBUTION_CLAIM` / `CONTRIBUTION_CONFIRM` (6.4)

```
ContributionClaimPayload   { bytes description_hash = 1; uint64 size = 2; uint64 period = 3; }
ContributionConfirmPayload { bytes claim_txid = 1; }
```

- Подтверждение только `peer`: N независимых подтверждений от разных
  `active`-кошельков, не связанных с заявителем поручительствами; лимит
  подтверждений на человека в период (6.4). N, лимиты, потолки — PENDING:
  вопрос №13.
- Выплата из пула программы (рекомендуется) либо ограниченная эмиссия
  `on_contribution` с потолком на период и на человека (6.4) — потолки
  PENDING: вопрос №13.

## 1.6. Сообщения вне реестра

- **`Attestation`** (6.1) — предъявление: подписывается кошельком, к нему
  прилагаются Merkle-доказательство из чекпоинта и сам чекпоинт; проверяющему
  не нужно быть узлом. Схема — `proto/attest.proto`. Защита от повторного
  использования: `challenge` и срок годности. `app_subject_id =
  H(address ‖ app_id)` — публичен, тайные голосования в v1 не поддерживаются.
- **`MigrateStatement`** — каноническое утверждение, подписываемое гардианом
  (не транзакция; вкладывается в `MIGRATE/propose`).
- **Подписанные seed-списки** (12.3) и **список отозванных аттестационных
  сертификатов** (7.2) — подписанные данные, распространяемые через сеть;
  их wire-формат — вне этапа 0 (сетевой слой, этап 4).
