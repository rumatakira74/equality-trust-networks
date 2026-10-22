// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get appTitle => 'Сети доверия';

  @override
  String get navWallet => 'Кошелёк';

  @override
  String get navVouching => 'Поручительства';

  @override
  String get navRegistry => 'Реестр';

  @override
  String get navMigration => 'Миграция';

  @override
  String get navSettings => 'Настройки';

  @override
  String get btnNext => 'Далее';

  @override
  String get btnCancel => 'Отмена';

  @override
  String get btnConfirm => 'Подтвердить';

  @override
  String get btnRefresh => 'Обновить';

  @override
  String get btnSkip => 'Пропустить';

  @override
  String get btnLogin => 'Войти';

  @override
  String get btnFingerprint => 'Отпечаток';

  @override
  String get btnSend => 'Отправить';

  @override
  String get btnSendBt => 'Передать по Bluetooth';

  @override
  String get btnVouch => 'Поручиться';

  @override
  String get btnMyQr => 'Мой QR';

  @override
  String get btnHideQr => 'Скрыть мой QR';

  @override
  String get btnBtReceive => 'Получить по BT';

  @override
  String get btnMint => 'Выпуск (челлендж)';

  @override
  String get btnMintDo => 'Выпустить (усиленный PoW)';

  @override
  String get btnTransfer => 'Перевод';

  @override
  String get btnPublishAwaiting => 'Опубликовать (сеть доступна)';

  @override
  String get btnChangeThreshold => 'Сменить порог';

  @override
  String get btnCancelTransfer => 'Отменить';

  @override
  String get btnRegister => 'Зарегистрировать кошелёк';

  @override
  String get btnStartMigration => 'Запустить миграцию (новое устройство)';

  @override
  String get btnApproveMigration => 'Подтвердить чужую миграцию (я — гардиан)';

  @override
  String get btnRestart => 'Перезапустить кошелёк';

  @override
  String get btnUnderstandRestart => 'Понимаю, перезапустить';

  @override
  String get btnAddSig => 'Добавить подпись';

  @override
  String get btnPublishProposal => 'Опубликовать предложение';

  @override
  String get btnSignBiometric => 'Подписать (биометрия)';

  @override
  String get enterPin => 'Введите PIN';

  @override
  String get errWrongPin => 'Неверный PIN';

  @override
  String get confirmOperation => 'Подтвердите операцию';

  @override
  String get keyUnlockTitle => 'Разблокировка ключа';

  @override
  String rejectedWithCode(String code) {
    return 'Отклонено: $code';
  }

  @override
  String errorWith(String e) {
    return 'Ошибка: $e';
  }

  @override
  String get obTitle => 'Создание кошелька';

  @override
  String get obBioRequired =>
      'Кошелёк требует аппаратного ключа с биометрией (BIOMETRIC_STRONG). Без настроенного отпечатка регистрация невозможна.';

  @override
  String get obBioNotSetup =>
      'Отпечаток не настроен. Откройте системный экран настройки и добавьте отпечаток.';

  @override
  String get obBioSetup => 'Настроить отпечаток';

  @override
  String get obPinIntro =>
      'PIN защищает приложение. Тревожный PIN (по желанию) открывает режим-приманку: тот же интерфейс, но правдоподобный небольшой остаток; ключ подписи при этом не используется.';

  @override
  String get obPinNormal => 'PIN (обычный)';

  @override
  String get obWantDuress => 'Задать тревожный PIN';

  @override
  String get obDuressPin => 'Тревожный PIN';

  @override
  String get obDecoyLegend =>
      'Легенда режима-приманки (что увидит принуждающий):';

  @override
  String get obDecoyBalance => 'Правдоподобный остаток (не 0)';

  @override
  String get obDecoyHistory => 'История (строки: дата — сумма — получатель)';

  @override
  String get pinMinDigits => 'минимум 4 цифры';

  @override
  String get duressPinMinDigits => 'тревожный PIN: минимум 4 цифры';

  @override
  String get pinsMatch => 'PIN-ы совпадают';

  @override
  String get obGuardiansIntro =>
      'Назначьте личных гардианов (2 из 3). Они смогут подтвердить миграцию на новое устройство. Добавление нового отпечатка инвалидирует ключ — без гардианов доступ к кошельку теряется.';

  @override
  String get obGuardiansEmpty =>
      'В сети пока нет других участников. Гардианов можно не назначать сейчас, но предложение обязательно: позже список не меняется (только при создании).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, возраст $age дн.';
  }

  @override
  String obSelectedCount(int count) {
    return 'Выбрано: $count';
  }

  @override
  String get regStatusReady => 'Готово к регистрации';

  @override
  String get regStatusPow => 'Аттестация и PoW (может занять время)…';

  @override
  String get regConfirmTitle => 'Подтвердите регистрацию';

  @override
  String get regStatusDone => 'Кошелёк зарегистрирован';

  @override
  String pendingCount(int count) {
    return 'Отложенных переводов: $count';
  }

  @override
  String get pendingHint =>
      'Крупные переводы исполняются через задержку D; отменить можно в течение D';

  @override
  String get historyTitle => 'История';

  @override
  String get historyEmpty => 'Пока пусто';

  @override
  String get effBalance => 'эффективный остаток';

  @override
  String get effBalanceDem => 'эффективный остаток (демередж применён)';

  @override
  String lockedReserve(int amount) {
    return 'в резерве отложенных: $amount';
  }

  @override
  String get flaggedNotice =>
      'Кошелёк помечен (flagged) — метка публична, вердиктов нет';

  @override
  String get registeredNeedK =>
      'Статус registered: для активации нужно K поручительств (вкладка «Поручительства»)';

  @override
  String get btAwaitingTitle => 'Получено по Bluetooth — ожидает публикации';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount от $hash';
  }

  @override
  String btStatus(String status) {
    return 'статус: $status';
  }

  @override
  String get txRegister => 'Регистрация кошелька';

  @override
  String get txVouch => 'Поручительство';

  @override
  String txMint(int period) {
    return 'Выпуск (период $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Перевод $amount → $hash';
  }

  @override
  String get mintTitle => 'Выпуск';

  @override
  String mintPeriod(int period) {
    return 'Период: $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'Окно челленджа открыто (±$hours ч вокруг слота). Можно выпускать.';
  }

  @override
  String get mintWindowOff =>
      'Проверка окна челленджа отключена (песочница). Можно выпускать.';

  @override
  String mintWindowClosed(int periods) {
    return 'Окно челленджа закрыто. Слот внутри месяца непредсказуем заранее; пропуск переносится в пределах окна накопления ($periods периодов), средства не блокируются.';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Последний заявленный период: $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Период к заявлению: $period (завершённые периоды, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Сумма к выпуску: $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Эмиссия за период: $amount';
  }

  @override
  String get mintConfirmTitle => 'Подтвердите выпуск';

  @override
  String get mintPowStatus => 'Усиленный PoW (Argon2id)…';

  @override
  String get mintNoPeriods => 'Нет периодов к заявлению';

  @override
  String get mintPublished => 'Выпуск опубликован';

  @override
  String get transferTitle => 'Перевод';

  @override
  String get segNetwork => 'Через сеть';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Адрес получателя (hex)';

  @override
  String get recipientHint => '64 hex-символа';

  @override
  String get amountLabel => 'Сумма (получатель получит ровно столько)';

  @override
  String feeHelper(int fee) {
    return 'Комиссия: $fee (сверх суммы, сжигается) — платит отправитель';
  }

  @override
  String delayedWarn(int hours) {
    return 'Сумма выше порога P: перевод будет отложен. Исполнение через $hours ч; отменить можно в течение задержки (экран «Отложенные»).';
  }

  @override
  String get btModeHint =>
      'Bluetooth офлайн: получатель увидит «получено по Bluetooth — ожидает публикации». До подтверждения сети это не оплата (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Мораторий: переводы доступны через $days дн. после регистрации.';
  }

  @override
  String get transferConfirmTitle => 'Подтвердите перевод';

  @override
  String get btConfirmTitle => 'Подтвердите Bluetooth-платёж';

  @override
  String get transferSent => 'Перевод отправлен';

  @override
  String transferDelayed(int hours) {
    return 'Перевод отложен: исполнение через $hours ч, отменить можно в течение задержки';
  }

  @override
  String get signNotConfirmed => 'Подпись не подтверждена';

  @override
  String get btNoDevices => 'Нет сопряжённых устройств';

  @override
  String get btSent =>
      'Передано по Bluetooth. Получатель обязан дождаться подтверждения сети — до публикации это не оплата';

  @override
  String btError(String error) {
    return 'Bluetooth: $error';
  }

  @override
  String get pendingTitle => 'Отложенные переводы';

  @override
  String thresholdP(int value) {
    return 'Порог P: $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'Повышение до $value вступает после $time';
  }

  @override
  String get newThresholdLabel => 'Новый порог P';

  @override
  String newThresholdHelper(int max) {
    return 'предел P_max = $max; повышение — через 48 ч, понижение — сразу';
  }

  @override
  String get pendingEmpty => 'Нет активных отложенных переводов';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'исполнение через $time (комиссия $fee)';
  }

  @override
  String get cancelConfirmTitle => 'Подтвердите отмену перевода';

  @override
  String get canceledMsg => 'Отменено';

  @override
  String get invalidThreshold => 'Некорректный порог';

  @override
  String get thresholdRaised =>
      'Повышение порога вступит в силу через 48 ч (защита от принуждения)';

  @override
  String get thresholdLowered => 'Порог понижен немедленно';

  @override
  String timeHm(int h, int m) {
    return '$h ч $m мин';
  }

  @override
  String get vouchingTitle => 'Поручительства';

  @override
  String get segMyStatus => 'Мой статус';

  @override
  String get segVouch => 'Поручиться';

  @override
  String get segMySlots => 'Мои слоты';

  @override
  String get walletActive => 'Кошелёк активен';

  @override
  String vouchProgress(int received, int k) {
    return 'Поручительств: $received из $k';
  }

  @override
  String get vouchFindHelpers =>
      'Найдите K поручителей при личной встрече: покажите им QR-код своего адреса на вкладке «Поручиться» (кнопка «Мой QR»).';

  @override
  String get vouchFlagged =>
      'Метка flagged: поручиться за вас нельзя (vouch от flagged-кошелька запрещён, решение №14)';

  @override
  String get vouchConfirmTitle => 'Подтвердите поручительство';

  @override
  String get vouchPublished => 'Поручительство опубликовано';

  @override
  String get btWaiting => 'Ожидание адреса по Bluetooth (60 с)…';

  @override
  String get btAddrReceived => 'Адрес получен по Bluetooth';

  @override
  String get vouchWarning =>
      'Поручительство при личной встрече (QR или Bluetooth) НЕ доказывает присутствие человека. Вы отвечаете за выбор: слот ограничен, при evidence против поручаемого вы теряете вознаграждение (не деньги).';

  @override
  String get vouchOnlyActive =>
      'Поручителем может быть только active-кошелёк с возрастом ≥ 3 мес. и ≥ 2 выполненными периодами выпуска (7.4).';

  @override
  String get candidateLabel => 'Адрес кандидата (hex)';

  @override
  String get candidateHint => '64 hex-символа — из QR или Bluetooth';

  @override
  String get myQrHint => 'Мой адрес — покажите поручителю';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Слоты: $occupied занято из $capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Свободно: $free. Статистика поручителя: поручился за $total, активны $active (публичная репутация, 7.4)';
  }

  @override
  String get slotsEmpty => 'Слоты свободны';

  @override
  String slotActiveSub(int periods) {
    return 'кандидат активен: слот освободится после $periods периодов выпуска без evidence';
  }

  @override
  String slotInactiveSub(int days) {
    return 'кандидат не активен: слот освободится через $days дн.';
  }

  @override
  String get registryTitle => 'Реестр сети';

  @override
  String registryRowSub(String status, int age) {
    return '$status • возраст $age дн.';
  }

  @override
  String chipAge(int age) {
    return 'возраст $age дн.';
  }

  @override
  String get chipSeed => 'посевной круг';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'публичный гардиан';

  @override
  String vouchDepthLabel(int depth) {
    return 'Глубина графа поручительств: $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Глубина графа поручительств: не связан с посевным кругом';

  @override
  String rewardLine(int periods) {
    return 'Вознаграждение поручителям: $periods выполненных периодов выпуска';
  }

  @override
  String asVoucher(int count, int free) {
    return 'Как поручитель: за $count (свободно слотов: $free)';
  }

  @override
  String get publicHistory => 'Публичная история';

  @override
  String get migrationTitle => 'Миграция';

  @override
  String get migrationIntro =>
      'Восстановление кошелька: гардианы назначаются заранее (при создании). Личные — 2 из 3; с публичными — 3 из 5 (пороги комбинируются).';

  @override
  String migrationTo(String hash) {
    return 'Миграция на $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'исполнение через $days дн. (задержка $delay дн.); подписей гардианов: $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Публичные гардианы сети (guardian_offer, 11)';

  @override
  String get noGuardians => 'Пока никто не объявил себя публичным гардианом';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'возраст $age дн., подопечных: $wards';
  }

  @override
  String get restartIntro =>
      'Если гардианов нет: перезапуск — обычная регистрация с новым ключом и новыми K поручительствами. Старый кошелёк остаётся в реестре нетронутым и недоступным.';

  @override
  String get cancelMigrationTitle =>
      'Подтвердите отмену миграции (старый ключ)';

  @override
  String get migrationCanceled => 'Миграция отменена';

  @override
  String get restartDialogTitle => 'Потеря средств';

  @override
  String get restartDialogBody =>
      'Перезапуск не даёт ничего сверх регистрации нового кошелька: средства на старом адресе теряются, история не переносится. Миграция с гардианами переносит возраст, историю выпусков и поручительства.';

  @override
  String get restartFlowNote =>
      'Перезапуск: удалите данные приложения и пройдите создание кошелька заново (новый ключ, новые K поручительств).';

  @override
  String get startMigrationTitle => 'Запуск миграции';

  @override
  String get startMigrationIntro =>
      'Вы на НОВОМ устройстве. Введите адрес старого кошелька и подписи гардианов (каждый гардиан подписывает утверждение в своём приложении — вкладка «Подтвердить чужую миграцию»).';

  @override
  String get oldAddressLabel => 'Старый адрес (hex)';

  @override
  String get newAddressLabel => 'Новый адрес (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Подписей собрано: $count (личный порог $personal из $personalCount; с публичными — $public из $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Адрес гардиана (hex)';

  @override
  String get guardianSigLabel => 'Подпись гардиана (hex)';

  @override
  String get migrationConfirmTitle => 'Подтвердите миграцию';

  @override
  String migrationStarted(int days) {
    return 'Миграция запущена: исполнение через $days дн.; старый ключ может отменить в течение задержки';
  }

  @override
  String get approveTitle => 'Подтверждение миграции (гардиан)';

  @override
  String get approveIntro =>
      'Проверьте личность подопечного по своим каналам. Подпись связывает пару адресов и nonce; передайте её подопечному лично.';

  @override
  String get nonceLabel => 'Nonce (от подопечного)';

  @override
  String get wardSignTitle => 'Подтвердите миграцию подопечного (вы — гардиан)';

  @override
  String get sigToCopy => 'Подпись (передайте подопечному):';

  @override
  String get nonceToCopy => 'Nonce миграции (передайте гардианам):';

  @override
  String get settingsTitle => 'Настройки';

  @override
  String networkLine(String name) {
    return 'Сеть: $name';
  }

  @override
  String get paramUnit => 'Единица';

  @override
  String get paramTransfer => 'Передача';

  @override
  String get paramIssuance => 'Выпуск';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount за $months мес.';
  }

  @override
  String get paramVouchK => 'K (поручительства)';

  @override
  String get paramThresholdP => 'Порог P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Задержка D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours ч';
  }

  @override
  String get paramMoratorium => 'Мораторий';

  @override
  String paramMoratoriumValue(String days) {
    return '$days дн.';
  }

  @override
  String get paramDemurrage => 'Демередж';

  @override
  String get demurrageOn => 'включён';

  @override
  String get demurrageOff => 'выключен';

  @override
  String get paramAccrual => 'Окно накопления';

  @override
  String paramAccrualValue(String periods) {
    return '$periods периодов';
  }

  @override
  String get paramGuardians => 'Гардианы';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'личные $pt из $pc; публичные $pubt из $pubc';
  }

  @override
  String get paramMigration => 'Миграция';

  @override
  String paramMigrationValue(String days) {
    return 'задержка $days дн.';
  }

  @override
  String get paramEpoch => 'Эпоха чекпоинта';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds с';
  }

  @override
  String get sandboxNote => 'Песочница (синтетическая аттестация)';

  @override
  String get guardianOfferTitle => 'Объявить себя публичным гардианом';

  @override
  String guardianAlready(int wards) {
    return 'Вы уже публичный гардиан (подопечных: $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Требования: active, возраст ≥ $days дн.';
  }

  @override
  String get guardianConfirmTitle =>
      'Подтвердите объявление публичным гардианом';

  @override
  String get guardianPublished => 'Объявление опубликовано';

  @override
  String get nodeRunning => 'Узел работает';

  @override
  String get nodeStopped => 'Узел остановлен';

  @override
  String get nodeLevelHint =>
      'Уровень full — при зарядке и Wi-Fi, иначе light (12.2)';

  @override
  String get lockApp => 'Заблокировать приложение';

  @override
  String get relockNote => 'Заблокировано. Перезапустите приложение.';

  @override
  String get languageTitle => 'Язык';

  @override
  String get languageSystem => 'Системный';

  @override
  String get languageRu => 'Русский';

  @override
  String get languageEn => 'English';

  @override
  String get languageEs => 'Español';

  @override
  String get languagePt => 'Português';

  @override
  String get languageZh => '中文';

  @override
  String get recoveryTitle => 'Карточка восстановления';

  @override
  String get recoveryIntro =>
      'Сохраните эти данные вне телефона: они нужны для восстановления кошелька при утрате устройства (миграция с гардианами, раздел 11).';

  @override
  String get recoveryAddress => 'Адрес кошелька';

  @override
  String get recoveryNetwork => 'Сеть';

  @override
  String get recoveryGuardians => 'Ваши гардианы';

  @override
  String get recoveryNoGuardians =>
      'Гардианы не назначены: восстановление невозможно, доступен только перезапуск (обычная регистрация с новыми поручительствами).';

  @override
  String get recoveryHowToSave =>
      'Как сохранить: перепишите на бумагу или сохраните в надёжном месте вне телефона; не храните в облаке. Изображение карточки можно распечатать через «Поделиться».';

  @override
  String get recoveryNeverShare =>
      'Данные миграции (nonce, подписи гардианов) никому не передавайте — только напрямую вашему гардиану и только на время миграции; после завершения удалите их из переписок. Посторонним можно показывать только адрес и QR этой карточки.';

  @override
  String get recoverySavedDone => 'Сохранено — продолжить';

  @override
  String get settingsRecovery => 'Карточка восстановления';

  @override
  String get recoveryApkNote =>
      'Заранее сохраните и установочный файл (APK) той же сети: без него восстановление на новом устройстве может оказаться невозможным.';

  @override
  String get recoveryShareBtn => 'Поделиться карточкой (изображение)';

  @override
  String get guardiansTitle => 'Гардианы';

  @override
  String get guardiansIntro =>
      'Личные гардианы подтверждают миграцию (2 из 3). Их можно перевыбрать: смена вступает в силу через 48 ч, до этого действует старый список.';

  @override
  String get guardiansEntrySub => 'Личные гардианы: перевыбор, окно 48 ч';

  @override
  String get guardiansCurrent => 'Текущие гардианы';

  @override
  String get guardiansNone => 'Гардианы не назначены';

  @override
  String get guardiansReelect => 'Перевыбрать';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return 'Использовано смен гардианов: $used из $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return 'Новый список вступит в силу через $hours ч';
  }

  @override
  String get guardiansCancelBtn => 'Отменить смену';

  @override
  String get guardiansStatusCanceled => 'Отменена';

  @override
  String get guardiansStatusEffective => 'Вступила в силу';

  @override
  String get guardiansStatusExpired =>
      'Истекла: у гардиана превышен лимит подопечных';

  @override
  String get reelectTitle => 'Перевыбор гардианов';

  @override
  String get reelectIntro =>
      'Введите новых гардианов (до 3, hex, по одному в строке). Передайте каждому новому гардиану хеш смены: он подписывает его в своём приложении («Согласие гардиана») и передаёт вам hex подписи.';

  @override
  String get newGuardiansLabel => 'Новые гардианы (hex, по одному в строке)';

  @override
  String get changeHashToCopy => 'Хеш смены (передайте новым гардианам):';

  @override
  String guardiansApprovals(int count, int need) {
    return 'Согласий собрано: $count из $need';
  }

  @override
  String get btnPublishChange => 'Опубликовать смену';

  @override
  String get changePublished => 'Смена опубликована: вступит в силу через 48 ч';

  @override
  String get changeConfirmTitle => 'Подтвердите смену гардианов (PIN)';

  @override
  String get cancelChangeConfirmTitle => 'Подтвердите отмену смены (PIN)';

  @override
  String get changeCanceled => 'Смена отменена';

  @override
  String get consentTitle => 'Согласие гардиана';

  @override
  String get consentIntro =>
      'Вы подтверждаете согласие стать гардианом подопечного. Введите адрес подопечного и хеш смены, подпишите хеш. Передайте hex подписи подопечному.';

  @override
  String get wardAddressLabel => 'Адрес подопечного (hex)';

  @override
  String get changeHashLabel => 'Хеш смены (hex)';

  @override
  String get consentSignTitle =>
      'Подтвердите смену гардианов подопечного (вы — гардиан)';

  @override
  String get biometricTimeout =>
      'Биометрия не отвечает (таймаут) — попробуйте ещё раз';

  @override
  String hexChunksCount(int n, int total) {
    return 'Введено $n из $total';
  }

  @override
  String hexChunksSigCount(int n) {
    return 'Введено символов: $n';
  }

  @override
  String hexLenError(int total) {
    return 'Ожидается ровно $total hex-символов';
  }

  @override
  String get hexSigEvenError => 'Подпись — чётное число hex-символов';

  @override
  String get hexCharsDropped =>
      'Недопустимые символы удалены (только 0–9, a–f)';
}
