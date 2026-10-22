# Блокер прогона stage11-2: приложение не запускается на эмуляторах (установка проходит, activity не разрешается)

Дата: 21.10.2026. Ветка `stage11-2-guardians`, session-id: `stage11-2-run`.
Шаги мандата 1–3 (протокол/ядро, приложение, тесты) выполнены и закоммичены
(ee10019, ac14235, c0505f4): go-тесты (вкл. -race по задетым), flutter analyze
и flutter test — зелёные. Прогон шага 4 (проверочный, emu-run) остановлен на
подготовке: приложение невозможно запустить ни на одном из двух эмуляторов.

## Симптом

`adb install` проходит (`Success`), но запуск невозможен:

```
$ adb shell am start -n platform.tnp.tnp_app/platform.tnp.tnp_app.MainActivity
Error: Activity class {platform.tnp.tnp_app/platform.tnp.tnp_app.MainActivity} does not exist.
# logcat: ActivityTaskManager: START u0 {cmp=platform.tnp.tnp_app/.MainActivity}
#   ... result code=-92   (START_CLASS_NOT_FOUND)
$ adb shell cmd package resolve-activity --brief -c android.intent.category.LAUNCHER platform.tnp.tnp_app
No activity found
```

При этом `dumpsys package` показывает activity и фильтр MAIN/LAUNCHER на месте:
`cc75902 platform.tnp.tnp_app/.MainActivity filter 78bb213`.

## Проверено (факты)

1. **Оба эмулятора**: tnp_api35 (API 35, x86_64, google_apis_playstore_tablet)
   и Pixel_9 (API 36) — симптом одинаковый. Не AVD-специфика.
2. **APK цел**: md5 на устройстве совпадает с локальным
   (7566161ffb74acf97de1527218685d6d). Сборка `flutter build apk --debug`
   успешна.
3. **Манифест в APK содержит activity**: `aapt2 dump xmltree` по APK —
   `E: application`, `E: activity` (строки 52, 62), package
   `platform.tnp.tnp_app`.
4. **Класс в dex есть**: APK — multidex (12 dex-файлов); строки
   `platform/tnp/tnp_app/MainActivity` присутствуют в `classes6.dex`.
5. **Пакет ставится disabled**: после свежей установки
   `dumpsys package` → `User 0: ... enabled=0` (для свежего пакета это
   не норма). `pm enable` → `enabled=1`, но разрешение activity всё равно
   не работает.
6. **Каталог данных не создаётся**: `run-as platform.tnp.tnp_app ls files/` →
   `couldn't stat /data/user/0/platform.tnp.tnp_app` — приложение ни разу
   не запустилось.
7. logcat при установке: `Package [platform.tnp.tnp_app] reported as REPLACED,
   but missing application info. Assuming REMOVED` (многократно);
   `OatFileAssistant test for existing oat file .../base.dm: I/O error`.
8. `adb install --no-streaming` и полная переустановка
   (uninstall → install) — тот же результат.
9. Часы обоих эмуляторов верные (синхронизированы, Oct 21 2026);
   `device_provisioned=1`; device policy пуст; keyguard dismissing
   результата не меняет.

## Отличие от stage11-prep (21.10 утром) — рабочая гипотеза

Прогон stage11-prep сегодня утром запускал приложение на этих же AVD
(tnp_api35, Pixel_9) успешно. С тех пор в сборке изменилось:

- я пересобрал AAR ядра (`gomobile bind -target=android -androidapi 26
  -ldflags "-checklinkname=0" -o android/libs/tnp.aar ./mobile`) — в него
  добавились биндинги update_guardians (новые методы mobile/query.go);
- пересобран app-debug.apk (вчерашний билд от 20.10 в
  `app/build/app/outputs/flutter-apk/` работал в stage11-prep).

При этом сборка Gradle проходит без ошибок компиляции (только deprecation
warnings AGP 9: `android.builtInKotlin=false`, `android.newDsl=false`,
Kotlin 2.4 — «Deprecated org.jetbrains.kotlin.android plugin usage»).
Похоже на повреждение упаковки dex/установочного образа именно текущего
билда (multidex: класс платформенного слоя оказался в classes6.dex при
12 dex-файлах), либо на молчаливый отказ верификации при установке
(следствие — пакет disabled и «Assuming REMOVED»).

## Кандидаты на следующую итерацию (не выполнялись — жду решения владельца)

1. Бисект: поставить на эмулятор вчерашний `app-release.apk` (от 20.10,
   сборка stage11-prep) — отделить «среда сломалась» от «мой билд сломан».
2. `flutter clean` + пересборка APK (чистка кэша Gradle/Kotlin daemon).
3. Собрать debug-APK на коммите stage11-prep без моих изменений моста —
   если работает, добавлять изменения по одному (AAR / Kotlin-мост).
4. Проверить dex-файлы текущего APK dexdump-ом (валидность classes6.dex).
5. Fallback по мандату (п. 4.5): если среда блокирует — закрыть шаг
   прогоном по тестам + протокольной демонстрацией CLI-кошельками
   (хелпер /tmp/stage11-2/helper уже умеет sign/hash/propose/cancel для
   update_guardians; сеть stage11-2 сконфигурирована).

## Состояние инфраструктуры прогона (для продолжения)

- Сеть: `tnp-testnet configure -template 1 -network-id stage11-2
  -genesis-time <now−91сут> -seeds S122,S01,S128`, sandbox, эпоха 300 с,
  VouchK=2. network_id
  4f6ab03fb97c9d6eb452975ca4f1e70712e3c15d1a1d6fdda0abf363990226aa.
  Артефакты: /tmp/stage11-2/net (вне репо).
- Бинарники: /tmp/stage11-2/bin (tnp-node linux/amd64 — запускается внутри
  эмулятора; tnp-testnet; helper).
- Сценарий: подопечный A — кошелёк приложения (гардианы при register —
  посевные из реестра), перевыбор на нового гардиана S128 (посевной, уже
  active — mint-окна и vouch-ы не нужны); подпись согласия S128 — хелпер
  `sign -hash <хеш смены с экрана> -seed S128`; публикация и отмена — из
  приложения (PIN); вступление — сдвиг часов эмулятора рестартом с
  `-qemu -rtc base=+49h` (образ Play Store: adb root/su недоступны).
- Уборка на паузе выполнена: эмуляторы остановлены (`adb emu kill`),
  Gradle-демоны остановлены, `pgrep qemu java tnp-node tnp-testnet` — пусто.
