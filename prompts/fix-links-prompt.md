# Мандат fix-links — исправление 7 битых ссылок из аудита audit-links

## Контекст

Ты — zcode-агент, работаешь локально в каталоге `/home/projects/audit-equality-trust-networks`. Если каталога нет — создай и склонируй репо из списка ниже; если есть — сделай `git fetch --all --prune` в каждом.

Четыре репо (owner `rumatakira74`):

| Код | Репо |
|---|---|
| tnp | equality-trust-networks |
| eq | equality-anti-farm-simulation |
| programming-docs | equality-programming-docs |
| user-docs | equality-user-docs |

Аудит ссылок (ветка `audit-links` репо tnp, отчёт `REPORT-LINKS-AUDIT.md`) проверил 202 `.md`, нашёл 7 битых ссылок. Владелец принял решения по каждой; этот мандат их реализует. Ссылки на отчёт аудита в merge-сообщениях — текстом, ветка `audit-links` ещё не влита в main.

Контрольные SHA `origin/main` на момент составления мандата:

| Репо | SHA main |
|---|---|
| tnp | родитель пуша этого промпта (промпт добавляется коммитом в main поверх) |
| eq | `515b6b9` |
| programming-docs | `3dccc19` |
| user-docs | `0eea26e` |

Если HEAD `origin/main` репо eq / programming-docs / user-docs отличается от контрольного SHA — СТОП: перечисли расхождения и спроси владельца, продолжать ли.

## Правила

- В каждом репо — одна ветка `fix-links` от свежего `origin/main`.
- Правки строго по списку ниже. Никаких других изменений: без переформатирования, правок пробелов, сортировок, «улучшений» текста.
- Удаление строк — вместе с оговорёнными заголовками разделов, если раздел опустел (см. правки).
- Никаких файлов вне перечисленных: не создавать, не удалять.
- После влития ветки `fix-links` не удалять (удаление веток — за владельцем).
- Конвейер и прочие ветки не трогать.

## Правки

### Репо 1: equality-user-docs — 2 правки (битый якорь)

Причина: заголовок `### Карточка восстановления — заполните заранее` (docs/03-operations.md:60) содержит тире (em dash); GitHub-слаг превращает его в **двойной дефис**, а в ссылках стоит одинарный.

Файл `docs/02-getting-started.md`, строка 68. Заменить вхождение:

```
#карточка-восстановления-заполните-заранее
```

на:

```
#карточка-восстановления--заполните-заранее
```

Файл `docs/04-security.md`, строка 49. Та же замена того же якоря.

Самопроверка: `grep -rn 'карточка-восстановления-заполните' .` — старый якорь больше нигде не встречается; новый якорь соответствует слагу заголовка в `docs/03-operations.md`.

### Репо 2: equality-programming-docs — 1 правка (мёртвые ссылки на удалённую ветку аудита)

Причина: две ссылки на ветку `audit` репозитория equality-trust-networks, удалённую с ремута. Решение владельца: гиперсылки убрать, упоминание аудита сохранить текстом.

Файл `docs/05-network-merge-rejected.md`, строка 12. Полная строка сейчас:

```
- Аудит: независимый внешний аудит дизайна — [`audit/results/AUDIT-network-merge.md`](https://github.com/rumatakira74/equality-trust-networks/blob/audit/audit/results/AUDIT-network-merge.md) (ветка [`audit`](https://github.com/rumatakira74/equality-trust-networks/tree/audit) репозитория [equality-trust-networks](https://github.com/rumatakira74/equality-trust-networks)). Вердикт: **«Переработать»** — 4 находки CRITICAL, 9 HIGH, 7 MEDIUM; из 10 проектных вопросов аудита 5 получили ответ «проблема», 1 — «неоднозначно».
```

Заменить целиком на:

```
- Аудит: независимый внешний аудит дизайна — `audit/results/AUDIT-network-merge.md` (выполнялся в отдельной ветке репозитория [equality-trust-networks](https://github.com/rumatakira74/equality-trust-networks), удалённой после закрытия темы). Вердикт: **«Переработать»** — 4 находки CRITICAL, 9 HIGH, 7 MEDIUM; из 10 проектных вопросов аудита 5 получили ответ «проблема», 1 — «неоднозначно».
```

### Репо 3: equality-anti-farm-simulation — 1 правка, 2 файла (фантомный naming-conventions.md)

Причина: ссылки на `naming-conventions.md` — файл никогда не существовал. Правила хранения артефактов — личные, живут в базе знаний владельца и не публикуются. Строка — единственный пункт своего раздела, раздел закрывает файл.

Файл `KNOWLEDGE.md`. Удалить целиком хвост:

```
## Файлы темы (служебное)

- [Правила именования и хранения артефактов](naming-conventions.md) — план разработки всегда `development-plan`; в облаках всё кладётся в папку `mistral`, не в корень.
```

Файл `KNOWLEDGE-ENG.md`. Удалить целиком хвост:

```
## Theme files (service)

- [Naming and artifact-storage rules](naming-conventions.md) — the development plan is always `development-plan`; in the clouds, everything goes into the `mistral` folder, not the root.
```

В обоих файлах: не оставлять висячий разделитель `---` и двойные пустые строки в конце; файл заканчивается последним содержательным пунктом предыдущего раздела и одним переводом строки.

### Репо 4: equality-trust-networks — 1 правка (тот же фантом)

Файл `docs/KNOWLEDGE-ENG-5.md`. Удалить целиком хвост (включая висячий разделитель перед разделом):

```
---

## Theme files (service)

- [Naming and artifact-storage rules](naming-conventions.md) — the development plan is always `development-plan`; in the clouds, everything goes into the `mistral` folder, not the root.
```

Файл заканчивается последним содержательным пунктом предыдущего раздела и одним переводом строки.

## Самопроверка (обязательно, в каждом репо после правок)

1. `git diff origin/main..fix-links` — только заявленные строки (плюс оговорённые заголовки/`---` при удалении хвостов). Любая лишняя строка — СТОП и вопрос владельцу.
2. Повторная проверка правленых файлов по методике аудита: все оставшиеся markdown-ссылки и якоря резолвятся, битых нет.
3. Чекпоинт: коммит правок в `fix-links` + push ветки. Формат сообщения коммита: `fix-links: <суть правки> (repo <код>)`.

## Merge — решение владельца, входит в мандат

После самопроверки всех четырёх репо: влить `fix-links` → `main` в каждом репо, **no-ff**, готовыми сообщениями (дословно), push `main`. Порядок: user-docs → programming-docs → eq → tnp.

**user-docs:**

```
Merge branch 'fix-links': битые якоря «Карточка восстановления» — двойной дефис слага

docs/02-getting-started.md:68 и docs/04-security.md:49: якорь #карточка-восстановления-заполните-заранее → #карточка-восстановления--заполните-заранее (заголовок в docs/03-operations.md содержит тире, GitHub-слаг даёт двойной дефис). Найдено аудитом audit-links.

Generated by Mistral Vibe.
Co-Authored-By: Mistral Vibe <vibe@mistral.ai>
```

**programming-docs:**

```
Merge branch 'fix-links': сняты мёртвые ссылки на удалённую ветку аудита network-merge

docs/05-network-merge-rejected.md:12: гиперсылки на ветку audit заменены текстом, упоминание аудита сохранено; ветка аудита удалена с ремута после закрытия темы. Найдено аудитом audit-links.

Generated by Mistral Vibe.
Co-Authored-By: Mistral Vibe <vibe@mistral.ai>
```

**eq:**

```
Merge branch 'fix-links': удалены строки со ссылкой на фантомный naming-conventions.md

KNOWLEDGE.md и KNOWLEDGE-ENG.md: единственные пункты разделов «Файлы темы (служебное)» / "Theme files (service)" со ссылкой на несуществующий naming-conventions.md удалены вместе с опустевшими разделами; правила хранения артефактов — личные, в базе знаний владельца, не публикуются. Найдено аудитом audit-links.

Generated by Mistral Vibe.
Co-Authored-By: Mistral Vibe <vibe@mistral.ai>
```

**tnp:**

```
Merge branch 'fix-links': удалена строка со ссылкой на фантомный naming-conventions.md

docs/KNOWLEDGE-ENG-5.md: единственный пункт раздела "Theme files (service)" со ссылкой на несуществующий naming-conventions.md удалён вместе с опустевшим разделом; правила хранения артефактов — личные, в базе знаний владельца, не публикуются. Найдено аудитом audit-links.

Generated by Mistral Vibe.
Co-Authored-By: Mistral Vibe <vibe@mistral.ai>
```

## Порядок работы

1. Подготовка: fetch всех репо, ветка `fix-links` от свежего `origin/main` в каждом. Проверка контрольных SHA (см. выше).
2. Правки по списку: user-docs → programming-docs → eq → tnp. После правок каждого репо — самопроверка, чекпоинт-коммит, push ветки.
3. Серия merge: 4 × no-ff merge в `main` с готовыми сообщениями, push `main` каждого репо.
4. Финальная сводка: таблица «репо → SHA merge-коммита в main → правки», плюс подтверждение повторной проверки ссылок.

## Параметры сессии

- Модель: DeepSeek Chat (V3.2), Flash; thinking: low.
- Лимит времени: ≤90 минут.
- Один сеанс, без перерывов между репо; стоп только по стоп-вопросам.

## Стоп-вопросы

Стоп и вопрос владельцу в случаях:

- HEAD `origin/main` любого репо отличается от контрольного SHA (кроме tnp, куда добавлен этот промпт).
- В `git diff` появились строки, не заявленные мандатом.
- Конфликт слияния или любая неоднозначность в правках.
- Любая находка, меняющая выводы аудита audit-links.
