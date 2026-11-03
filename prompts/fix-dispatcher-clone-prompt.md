# Разовая задача обслуживания конвейера (не тема разработки)

Контекст: клон диспетчера в текущем workdir разошёлся с origin/main — локально есть незапушенный коммит переноса задачи 00000011-a8-keys в processed/ (суффикс .done), а remote ушёл вперёд коммитами state.json от GitHub Actions диспетчера. Из-за этого cron-тики лаунчера с 10:10 UTC падают на git pull --ff-only (см. logs/launcher-pull.err: «Not possible to fast-forward»).

Выполни строго по пунктам:
1. git fetch origin.
2. git rebase origin/main — локальный коммит переноса статуса перебазируется поверх входящих коммитов state.json.
3. Если rebase конфликтует: state.json — принять версию origin/main (theirs); cli-jobs/ — сохранить перенос processed/00000011-a8-keys.md.done и отсутствие файла задачи в queue/.
4. git push origin main.
5. Проверь: git status — рабочая копия чистая; git log --oneline -3 — перенос статуса на вершине поверх коммитов диспетчера.
6. Проверь: git pull --ff-only проходит без ошибок (это условие оживления тиков).
7. Финал: только сводка — результат rebase, результат push, вывод git log --oneline -3. Без вопросов.

Ограничения: репозиторий /home/projects/audit-equality-trust-networks не трогать; секреты и содержимое credentials не печатать; кроме пунктов 1–6 ничего не делать; откатов истории remote не делать.