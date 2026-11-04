# Промпт темы release-v3 — релиз v3.0.0 (тег + GitHub Release)

Предусловия (проверить в первую очередь; при невыполнении ЛЮБОГО — СТОП, тег НЕ ставить, доложить в сводке results, что именно не сошлось):
1. В main tnp смержены все 9 фикс-тем — в reports/ есть отчёты: REPORT-FIX-SPEC-C11.md, REPORT-FIX-CP-THRESHOLD.md, REPORT-FIX-SYNC-CLAIM.md, REPORT-FIX-MINT-WINDOW.md, REPORT-FIX-RELEASE-KEYS.md, REPORT-FIX-D2-CLAMP.md, REPORT-FIX-D10-FUZZ.md, REPORT-FIX-D11-BUFFER.md, REPORT-FIX-A7-CLAIMS.md (точные имена сверить с фактом в reports/ main).
2. Тег v3.0.0 в ремуте ещё НЕ существует (git ls-remote --tags origin). Идемпотентность: если существует — СТОП и доклад.
3. CI main зелёный (gh run list --branch main --limit 3; если gh недоступен — по последнему статусу check-runs HEAD main, см. отчёты аудитов).

Рабочий клон: /home/projects/audit-equality-trust-networks (git fetch origin && git checkout main && git reset --hard origin/main). Личный клон /home/projects/equality-trust-networks НЕ трогать. Префикс audit* запрещён. Тег ставить ТОЛЬКО на origin/main (никаких веток).

Шаги:
1. Зафиксировать HEAD main: sha, краткий лог последних коммитов (фикс-мерджи должны быть в истории).
2. Аннотированный тег: git tag -a v3.0.0 -m "v3.0.0 — платформа сетей доверия; pre-live аудит пройден, блокеров нет".
3. git push origin v3.0.0.
4. GitHub Release: gh release create v3.0.0 --title "v3.0.0" --notes "<notes>" — в notes: краткое описание платформы (сети доверия без серверов и администратора: один человек = один кошелёк, равное периодическое право, публичная проверяемая история); вердикт pre-live аудита (блокеров нет, merge 07cb8a4); список закрытых тем фикс-этапа (fix-spec-c11, fix-cp-threshold, fix-sync-claim, fix-mint-window, fix-release-keys, fix-d2-clamp, fix-d10-fuzz, fix-d11-buffer, fix-a7-claims); protocol_version 3. Артефакты НЕ прикреплять (Zenodo — отдельно, за владельцем). Если gh недоступен — только тег, Release создаст Vibe через GitHub API; отметить это в сводке.
5. Проверка: gh release view v3.0.0 (или факт недоступности gh); git ls-remote --tags origin | grep v3.0.0.
6. Сводка в cli-jobs/results/ через временный клон в /tmp (как в фикс-темах): sha тега, URL релиза (или флаг «Release за Vibe»), результаты проверок, session-ID.

Запрещено: мержить что-либо, менять код/спеку, ставить другие теги, удалять ветки, трогать docs-репо и сим-репо. Чекпоинты событийные + ~60 мин (session-ID), чистое дерево.
