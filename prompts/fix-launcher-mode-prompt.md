# Микрозадача: восстановить exec-bit launcher.sh в репозитории диспетчера

Контекст: пуши через GitHub API сбрасывают tree-mode до 100644. Локальный клон /home/projects/mistral-harness-workflows уже имеет launcher.sh с +x (chmod владельца), но git видит это как незакоммиченное изменение режима (M в git status, диффа содержимого нет). Нужно зафиксировать tree-mode 100755 в репозитории, чтобы будущие пуллы не снимали exec-bit.

Выполни строго по пунктам (работай в /home/projects/mistral-harness-workflows через git -C или cd):
1. git fetch origin; затем git pull --ff-only — должен пройти чисто (если нет — доложи причину и остановись).
2. git status — убедись: единственное изменение — cli-jobs/launcher.sh, только mode 100644→100755, содержимое не менялось (git diff пуст).
3. git add cli-jobs/launcher.sh и закоммить: git commit -m "cli-jobs: launcher.sh — восстановлен exec-bit (tree-mode 100755), урок API-пуша 03.11".
4. git push origin main. Если push отклонён (remote ушёл вперёд — коммиты state.json диспетчера): git pull --rebase --autostash && git push origin main.
5. Проверь: git status — рабочая копия чистая; git log --oneline -2.
6. Финал: только сводка — результат, вывод git log --oneline -2. Без вопросов.

Ограничения: содержимое launcher.sh не менять (только бит режима); никакие другие файлы не трогать; workflows диспетчера и dispatcher.py не трогать; секреты не печатать; текущий workdir (клон tnp) не изменять — только чтение этого промпта.