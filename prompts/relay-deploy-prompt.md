# Деплой воркера tnp-relay — часовая лента

Цель: обновить Cloudflare Worker tnp-relay новой версией из main репо mistral-harness-workflows. Изменения: scheduled-обработчик + cron-триггер `0 * * * *` — часовая лента состояния конвейера владельцу в Telegram (очередь CLI-задач, последние сессии, ветка a8-keys + CI, стадия диспетчера).

Шаги:
1. Перейти в клон диспетчера: `cd /home/projects/mistral-harness-workflows`. Убедиться, что он на ветке main (`git status`, `git branch --show-current`); затем `git pull --ff-only`. Если клон не на main или pull не проходит — записать причину и остановиться.
2. Убедиться, что wrangler авторизован: `wrangler whoami` (токены и секреты не печатать).
3. Деплой: `wrangler deploy` (конфиг wrangler.jsonc в корне клона, main = relay/worker.js). В выводе должна быть строка про cron-триггер (scheduled triggers / schedules).
4. Проверка: `curl -s https://tnp-relay.13sweetholly13.workers.dev` — ожидается ответ `tnp-relay ok`.
5. Сводка: строка деплоя из вывода wrangler (версия), наличие cron-триггера в выводе, результат health-check.

Правила: секреты (токены, пароли) не печатать; репозитории кода не менять; коммитов не делать; в конце — только сводка, без вопросов.
