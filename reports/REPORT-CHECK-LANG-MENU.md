# Отчёт: тема check-lang-menu — меню выбора языка интерфейса на эмуляторе

Дата прогона: 05.11.2026. Ветка: `check-lang-menu` (от свежего `main`).
Session-ID: `03a779bc-f540-baf2-9f55-ba128678b94a`.
Характер работы: read-only по коду приложения — код не менялся, ветка содержит
только отчёт, скриншоты и находки.

## 1. Подготовка

| Пункт | Значение |
| --- | --- |
| SHA `main` на момент прогона | `58f76c192d48e69729b9589e861eb5a40117af07` |
| `git checkout main && git pull --ff-only` | `fc67f2f..58f76c1` fast-forward, дерево чистое |
| `flutter pub get` | `Got dependencies!` |
| `flutter analyze` | **`No issues found!`** (2.2 с) — нечистых предупреждений нет |
| Flutter / Dart | 3.47.9 stable / Dart 3.13.7 |

## 2. Сборка APK

| Пункт | Значение |
| --- | --- |
| Команда | `flutter build apk --debug` (в `app/`) |
| Результат | `✓ Built build/app/outputs/flutter-apk/app-debug.apk`, 60.5 с |
| Путь | `app/build/app/outputs/flutter-apk/app-debug.apk` |
| Размер | 357 078 167 байт (341 МиБ) |
| MD5 | `f418bd954f1baf4975e20696dae08683` |
| Релизная сборка | не собиралась (вне мандата) |

## 3. Эмулятор

| Пункт | Значение |
| --- | --- |
| Остатки прошлого прогона | `pgrep qemu/java/tnp-node/tnp-testnet` — пусто; `gradlew --stop` до и после |
| AVD | `stage12ga` (API 36, `google_apis`, x86_64, KVM) |
| Запуск | `emulator -avd stage12ga -gpu host -no-audio -no-boot-anim -no-snapshot` под `Xvfb :99` |
| Загрузка | `adb wait-for-device`, `sys.boot_completed=1` |
| Язык системы | `ro.product.locale=en-US` (для шага «д» это английский) |
| Установка | `adb install -r` → `Success` (15.8 с) |
| Пакет / activity | `platform.tnp.tnp_app` / `platform.tnp.tnp_app.MainActivity` |

Особенность среды (воспроизведена по прецеденту `reports/a8-keys/07-install-run.txt`):
headless `-no-window` падает SIGSEGV в SwiftShader, поэтому эмулятор запускался
с `-gpu host` под Xvfb. Скриншоты снимались через `adb exec-out screencap`.

## 4. Фактический набор языков (по ARB в `main`)

`app/lib/l10n/app_{ru,en,es,pt,zh,fr,de,it,ja}.arb` — **9 языков**, не 5.
Ожидаемое состояние в мандате («5 языков: ru, en, es, pt, zh») устарело:
мандат сам предписывает проверять по фактическому набору. Сверка ARB:

| Файл | Ключей | Расхождений с `app_en.arb` | Ключи подписей языков |
| --- | --- | --- | --- |
| `app_de.arb` | 285 | 0 | все на месте |
| `app_en.arb` (шаблон) | 285 | — | все на месте |
| `app_es.arb` | 285 | 0 | все на месте |
| `app_fr.arb` | 285 | 0 | все на месте |
| `app_it.arb` | 285 | 0 | все на месте |
| `app_ja.arb` | 285 | 0 | все на месте |
| `app_pt.arb` | 285 | 0 | все на месте |
| `app_ru.arb` | 285 | 0 | все на месте |
| `app_zh.arb` | 285 | 0 | все на месте |

Паритет полный: 285 ключей в каждом файле, ни одного пропущенного или лишнего.
`AppLocalizations.supportedLocales` содержит все девять локалей; fallback — `en`
(шаблон `app_en.arb`). В коде меню (`app/lib/screens/settings/settings_screen.dart`)
константа `languages = ['ru','en','es','pt','zh','fr','de','it','ja']` и чип
«Системный» (`''`) — совпадает с набором ARB один в один.

Подписи языков заданы самоназваниями (Русский, English, Español, Português, 中文,
Français, Deutsch, Italiano, 日本語) и одинаковы во всех локалях — переключение
не «переводит» названия языков, что для выбора языка правильно.

## 5. Сценарий проверки

Каждый шаг — скриншот в `reports/check-lang-menu/`. Текст интерфейса сверялся не
по картинке, а по семантическому дереву (`uiautomator dump`; семантика Flutter
включалась `settings put secure accessibility_enabled 1`) — дампы приведены рядом
с ключевыми скриншотами.

| Шаг | Скриншот | Результат |
| --- | --- | --- |
| (а) запуск приложения, стартовый экран | `01-app-start.png` | Приложение стартует, activity разрешается, онбординг: шаг биометрии с требованием отпечатка (BIOMETRIC_STRONG), сеть `TNP Test Genesis · Eq-dd500a27` |
| (а′) прохождение онбординга (тестовый путь) | `02-onboarding-biometrics-ready.png`, `03-onboarding-pin.png`, `04-onboarding-recovery-card.png` | Отпечаток зарегистрирован в системе эмулятора, PIN `1234` (без тревожного), гардианы пропущены, регистрация принята — карточка восстановления |
| (а″) экран блокировки и главный экран | `05-lock-screen-en.png`, `06-main-wallet-tab-blank-flag-secure.png` | Вход по PIN; вкладка кошелька под FLAG_SECURE — снимок полностью чёрный (1 цвет, 2 617 920 пикселей), это штатное поведение, а не сбой |
| (б) переход в настройки, пункт выбора языка | `07-settings-tab-en.png` | Вкладка «Settings» доступна; карточка «Language» — первый элемент списка, чипы кликабельны |
| (в) меню языков открыто | `07-settings-tab-en.png`, `07-settings-uia.xml` | Ровно 10 чипов: «System» + 9 языков — совпадает с фактическим набором ARB, лишних/недостающих нет |
| (г) выбран `es` | `08-settings-es.png` | Интерфейс перешёл на испанский целиком: «Ajustes», «Idioma», «Sistema», «Acerca de la red», вкладки «Monedero / Avales / Registro / Migración» |
| (д) выбран `System` | `09-settings-system.png` | Интерфейс вернулся к языку системы эмулятора (`en-US` → английский): «Settings», «Language», «System» |
| (е) перезапуск приложения | `10-restart-lock-es.png`, `11-settings-es-after-restart.png` | Перед рестартом выбран `es`; в `FlutterSharedPreferences.xml` записано `ui_locale=es`; после `force-stop` + запуска экран блокировки испанский («Redes de confianza», «Introduzca el PIN», «Entrar»), настройки испанские, чип «Español» — `selected=true` |
| (ж) навигация назад и прочие пункты | `12-settings-other-items-es.png`, `13-recovery-card-es.png`, `14-settings-back-es.png` | Прочие пункты доступны и переведены: «Declararse guardián público», «El nodo está funcionando», «Tutores», «Bloquear la aplicación», «Tarjeta de recuperación»; переход в подэкран и системный «назад» возвращают в настройки без потери языка |

### 5.1. Дополнительно: полный обход всех девяти языков

Сверх мандата проверен каждый язык набора (переключение чипом, чтение заголовков
экрана настроек):

| Язык | Скриншот | Заголовок / «Язык» / «Системный» |
| --- | --- | --- |
| ru | `15-settings-ru.png` | Настройки / Язык / Системный |
| ja | `16-settings-ja.png` | 設定 / 言語 / システム |
| de | `17-settings-de.png` | Einstellungen / Sprache / System |
| zh | `18-settings-zh.png` | 设置 / 语言 / 跟随系统 |
| pt | `19-settings-pt.png` | Ajustes / Idioma / Sistema |
| fr | `20-settings-fr.png` | Réglages / Langue / Système |
| it | `21-settings-it.png` | Impostazioni / Lingua / Sistema |
| es | `08-settings-es.png` | Ajustes / Idioma / Sistema |
| en (System) | `22-settings-system-final.png` | Settings / Language / System |

Все девять переключаются мгновенно, без перезапуска, включая нелатинские
(кириллица, CJK) — кодировки и шрифты в порядке.

## 6. Вердикт

**Меню выбора языка доступно; переключение работает.**

- Пункт выбора языка присутствует в настройках, доступен без прокрутки и без
  дополнительных условий.
- Перечень чипов точно соответствует фактическому набору ARB: 9 языков +
  «Системный».
- Переключение применяется сразу ко всему интерфейсу (включая нижнюю навигацию),
  выбор сохраняется в `SharedPreferences` (`ui_locale`) и переживает перезапуск
  приложения.
- «Системный» корректно возвращает язык системы эмулятора (`en-US` → английский).
- Навигация по настройкам и подэкранам работает, прочие пункты доступны.

Замечания, не влияющие на вердикт: см. находки ниже (устаревшие комментарии про
«пять языков», блокер первой регистрации, строки вне ARB).

## 7. Находки

| Файл | Кратко |
| --- | --- |
| `FINDING-CHECK-LANG-MENU-01.txt` | Ожидаемое состояние мандата (5 языков) устарело: в `main` 9 языков; в коде остались комментарии про «пять языков этапа 10» |
| `FINDING-CHECK-LANG-MENU-02.txt` | На чистой установке первая регистрация кошелька падает `NONODE`; со второй попытки проходит (порядок `startNode` / `buildRegister` / `ownWallet`) |
| `FINDING-CHECK-LANG-MENU-03.txt` | Строки вне ARB: биометрический промпт и ошибки моста — по-русски при любом языке интерфейса |

## 8. Уборка

`adb emu kill`, `gradlew --stop`, остановка Xvfb. Проверка после прогона:
`pgrep qemu`, `pgrep java`, `pgrep tnp-node`, `pgrep tnp-testnet`, `pgrep Xvfb` —
пусто; `adb devices` — пусто. Бинарники сборки остались только в
`app/build/` (под `.gitignore`), в корень репозитория ничего не попало.
Рабочее дерево содержит только `reports/`.

## 9. Ограничения прогона

- Проверялся debug-APK (по мандату); release-сборка и подпись — вне темы.
- Язык системы эмулятора — `en-US`; проверка «Системный» на нелатинской системной
  локали (ru/ja/zh) не выполнялась — это отдельный сценарий.
- Тревожный PIN не задавался (для доступа к настройкам не требуется).
- Сеть — песочница с фиксированным genesis; пиров нет (`peers: 0`), на меню языка
  это не влияет.
