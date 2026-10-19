# Тестовые векторы спецификации v1.0-draft

Формат: `tnp-vector-1`. Каждый файл — фиксированная точка оракула проверки
(spec/03): параметры сети + пред-состояние + транзакции + ожидаемый результат.

## Поля

| Поле | Смысл |
| --- | --- |
| `network.params` | локальные значения параметров профиля (векторы фиксируют их локально; глобально fee/P/демередж — «этап 1.2») |
| `pre_state.epoch`, `pre_state.network_time`, `pre_state.current_period` | сетевое время: эпоха, Unix-секунды, номер календарного периода выпуска |
| `pre_state.wallets.<ID>` | статус (`REGISTERED`/`ACTIVE`/`EXPIRED`), `registered_at`, `activated_at`, `balance`, `seq` (последний принятый), `last_mint_period`, `slots_used`, `flags`, `vouch_count`, метки гардианов |
| `pre_state.pools` | состояние пулов: `pool_id`, `creator`, `recipient`, `goal`, `deadline_epoch`, `balance`, `mode`, `state` |
| `transactions[]` | конверт `Transaction` (spec/01) + payload; поле `id` — локальная ссылка |
| `expected[]` | `outcome` (`accept`/`reject`), `reason` (`OK`/`E_*` из spec/00), `effects` — пост-изменения |

## Плейсхолдеры (материализуются эталонной реализацией на этапе 2)

| Плейсхолдер | Чем заменяется |
| --- | --- |
| `SIG_PLACEHOLDER` | DER ECDSA P-256 над `signing_hash` |
| `POW_NONCE_PLACEHOLDER` | решение челленджа (лёгкий/Argon2id) |
| `txid:<WALLET>:<seq>` | реальный `txid` транзакции `<seq>` цепочки `<WALLET>` |
| `0^32` | 32 нулевых байта (`prev` первой транзакции) |
| `W_*` | реальные 32-байтные адреса |
| `ATTEST_*` | цепочка X.509 (реальная или синтетическая — этап 2) |

## Правило пройденности

Вектор пройден, если для каждой транзакции ядро выдаёт ожидаемые `outcome` +
`reason`, а пост-состояние совпадает с `effects` (балансы, статусы, метки,
слоты). Подписи/PoW/хеши при материализации обязаны сходиться; иначе —
дефект эталонной реализации, а не вектора.

## Перечень

- Валидные, по одному на каждый тип раздела 9.3: `valid_*.json` (15 файлов).
- Негативные/потоковые: `bad_equivocation.json`, `bad_double_mint.json`,
  `bad_moratorium.json`, `bad_accrual_window.json`, `flow_delayed_cancel.json`.
