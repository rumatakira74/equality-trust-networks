# Разовая задача: вернуть клон tnp на main и зафиксировать два фикса launcher.sh (содержимое + exec-bit)

Контекст (уроки 03.11):
(1) Клон /home/projects/audit-equality-trust-networks остался на ветке a8-keys после сессии A8 — лаунчер делает git pull --ff-only ТЕКУЩЕЙ ветки клона, поэтому промпты из main не находятся: задачи 00000012-fix-launcher-mode и 00000013-audit-a8-keys упали «файл промпта не найден».
(2) commit_and_push лаунчера при неудаче статусного пуша делает pull --ff-only — при расходящихся ветках (коммиты state.json диспетчера во время полёта сессии) это падает бесконечно (эпизод 10:10–11:50, тики молча умирали до ручной починки).
Очередь задач пуста, тики холостые — правка launcher.sh безопасна (правило конвейера: менять launcher.sh только при пустой очереди).

Выполни строго по пунктам:
1. Клон tnp: git -C /home/projects/audit-equality-trust-networks checkout main; затем git -C /home/projects/audit-equality-trust-networks pull --ff-only. Проверь: git -C /home/projects/audit-equality-trust-networks status — рабочая копия чистая.
2. В /home/projects/mistral-harness-workflows/cli-jobs/launcher.sh внеси ровно две правки (комментарии в коде — по-русски):
   а) В блоке «# 5) Клон target_repo…» ПЕРЕД существующим git pull --ff-only добавь строку:
      git -C "$target_clone" checkout main --quiet 2>>"$log_file" || fail_job "git checkout main клона $target_clone не удался"
      с комментарием: сессии оставляют клон на рабочих ветках, промпты читаются только из main (урок 03.11: 00000012/00000013 упали «файл промпта не найден»).
   б) В функции commit_and_push, в ретрай-цикле (for i in 1 2 3) замени «git -C "$PRIV_CLONE" pull --ff-only --quiet …» на «git -C "$PRIV_CLONE" pull --rebase --autostash --quiet …» — при неудаче пуша статус-коммит перебазируется на коммиты диспетчера (урок 03.11: diverged от state.json во время полёта).
3. Проверь синтаксис: bash -n /home/projects/mistral-harness-workflows/cli-jobs/launcher.sh — без ошибок.
4. Закоммить: git -C /home/projects/mistral-harness-workflows add cli-jobs/launcher.sh (add захватит и правку содержимого, и бит режима — файл на диске уже +x, tree-mode в коммите станет 100755); git commit -m "cli-jobs: launcher.sh — checkout main клона target_repo перед промптом; rebase-ретрай статусного пуша; exec-bit 100755 (уроки 03.11)".
5. git -C /home/projects/mistral-harness-workflows push origin main; при неудаче: git -C /home/projects/mistral-harness-workflows pull --rebase --autostash && git -C /home/projects/mistral-harness-workflows push origin main.
6. Проверь: git -C /home/projects/mistral-harness-workflows status — чисто; git log --oneline -2; затем /home/projects/mistral-harness-workflows/cli-jobs/launcher.sh --dry-run — должен отработать без ошибок (ожидаемо «dry-run: очередь пуста»).
7. Финал: только сводка — результаты пунктов, git log --oneline -2, вывод dry-run. Без вопросов.

Ограничения: в launcher.sh больше ничего не менять; workflows диспетчера, dispatcher.py, state.json не трогать; cli-jobs/queue/ и processed/ не трогать; ветку a8-keys репо tnp не менять (в клоне tnp — только checkout main); секреты не печатать.