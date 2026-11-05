// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

  @override
  String get appTitle => 'Vertrauensnetzwerke';

  @override
  String get navWallet => 'Wallet';

  @override
  String get navVouching => 'Bürgschaft';

  @override
  String get navRegistry => 'Register';

  @override
  String get navMigration => 'Migration';

  @override
  String get navSettings => 'Einstellungen';

  @override
  String get btnNext => 'Weiter';

  @override
  String get btnCancel => 'Abbrechen';

  @override
  String get btnConfirm => 'Bestätigen';

  @override
  String get btnRefresh => 'Aktualisieren';

  @override
  String get btnSkip => 'Überspringen';

  @override
  String get btnLogin => 'Entsperren';

  @override
  String get btnFingerprint => 'Fingerabdruck';

  @override
  String get btnSend => 'Senden';

  @override
  String get btnSendBt => 'Per Bluetooth senden';

  @override
  String get btnVouch => 'Bürgen';

  @override
  String get btnMyQr => 'Mein QR';

  @override
  String get btnHideQr => 'Mein QR verbergen';

  @override
  String get btnBtReceive => 'Per BT empfangen';

  @override
  String get btnMint => 'Emission (Challenge)';

  @override
  String get btnMintDo => 'Prägen (schweres PoW)';

  @override
  String get btnTransfer => 'Überweisen';

  @override
  String get btnPublishAwaiting => 'Veröffentlichen (Netz verfügbar)';

  @override
  String get btnChangeThreshold => 'Schwelle ändern';

  @override
  String get btnCancelTransfer => 'Abbrechen';

  @override
  String get btnRegister => 'Wallet registrieren';

  @override
  String get btnStartMigration => 'Migration starten (neues Gerät)';

  @override
  String get btnApproveMigration =>
      'Migration eines anderen genehmigen (ich bin Vormund)';

  @override
  String get btnRestart => 'Wallet zurücksetzen';

  @override
  String get btnUnderstandRestart => 'Ich verstehe, zurücksetzen';

  @override
  String get btnAddSig => 'Signatur hinzufügen';

  @override
  String get btnPublishProposal => 'Vorschlag veröffentlichen';

  @override
  String get btnSignBiometric => 'Signieren (Biometrie)';

  @override
  String get enterPin => 'PIN eingeben';

  @override
  String get errWrongPin => 'Falsche PIN';

  @override
  String lockTooManyAttempts(int seconds) {
    return 'Zu viele fehlgeschlagene Versuche. Erneut in $seconds s versuchen';
  }

  @override
  String get confirmOperation => 'Operation bestätigen';

  @override
  String get keyUnlockTitle => 'Schlüssel entsperren';

  @override
  String rejectedWithCode(String code) {
    return 'Abgelehnt: $code';
  }

  @override
  String errorWith(String e) {
    return 'Fehler: $e';
  }

  @override
  String get obTitle => 'Wallet-Erstellung';

  @override
  String get obBioRequired =>
      'Die Wallet erfordert einen Hardware-Schlüssel mit Biometrie (BIOMETRIC_STRONG). Ohne konfigurierten Fingerabdruck ist die Registrierung unmöglich.';

  @override
  String get obBioNotSetup =>
      'Kein Fingerabdruck konfiguriert. Öffnen Sie den Systemeinstellungs-Bildschirm und fügen Sie einen Fingerabdruck hinzu.';

  @override
  String get obBioSetup => 'Fingerabdruck einrichten';

  @override
  String get obPinIntro =>
      'Die PIN schützt die App. Eine Zwangs-PIN (optional) öffnet den Ködermodus: dieselbe Oberfläche, aber ein plausibler kleiner Saldo; der Signaturschlüssel wird nicht verwendet.';

  @override
  String get obPinNormal => 'PIN (normal)';

  @override
  String get obWantDuress => 'Zwangs-PIN festlegen';

  @override
  String get obDuressPin => 'Zwangs-PIN';

  @override
  String get obDecoyLegend =>
      'Legende des Ködermodus (was die zwingende Person sehen wird):';

  @override
  String get obDecoyBalance => 'Plausibler Saldo (nicht 0)';

  @override
  String get obDecoyHistory => 'Verlauf (Zeilen: Datum — Betrag — Empfänger)';

  @override
  String get pinMinDigits => 'mindestens 4 Ziffern';

  @override
  String get duressPinMinDigits => 'Zwangs-PIN: mindestens 4 Ziffern';

  @override
  String get pinsMatch => 'Die PINs stimmen überein';

  @override
  String get obGuardiansIntro =>
      'Benennen Sie persönliche Vormünde (2 von 3). Sie können die Migration auf ein neues Gerät genehmigen. Das Hinzufügen eines neuen Fingerabdrucks macht den Schlüssel ungültig — ohne Vormünde geht der Zugriff auf die Wallet verloren.';

  @override
  String get obGuardiansEmpty =>
      'Es gibt noch keine anderen Mitglieder im Netzwerk. Sie können Vormünde vorerst weglassen, aber das Angebot ist obligatorisch: Die Liste lässt sich später nicht mehr ändern (nur bei der Erstellung).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, Alter $age Tage';
  }

  @override
  String obSelectedCount(int count) {
    return 'Ausgewählt: $count';
  }

  @override
  String get regStatusReady => 'Bereit zur Registrierung';

  @override
  String get regStatusPow => 'Attestierung und PoW (kann dauern)…';

  @override
  String get regConfirmTitle => 'Registrierung bestätigen';

  @override
  String get regStatusDone => 'Wallet registriert';

  @override
  String pendingCount(int count) {
    return 'Aufgeschobene Transfers: $count';
  }

  @override
  String get pendingHint =>
      'Große Transfers werden nach der Verzögerung D ausgeführt; innerhalb von D können sie abgebrochen werden';

  @override
  String get historyTitle => 'Verlauf';

  @override
  String get historyEmpty => 'Noch nichts';

  @override
  String get effBalance => 'effektiver Saldo';

  @override
  String get effBalanceDem => 'effektiver Saldo (Demurrage angewendet)';

  @override
  String lockedReserve(int amount) {
    return 'für aufgeschobene reserviert: $amount';
  }

  @override
  String get flaggedNotice =>
      'Die Wallet ist markiert (flagged) — die Marke ist öffentlich, keine Urteile';

  @override
  String get registeredNeedK =>
      'Status registered: Die Aktivierung erfordert K Bürgschaften (Tab „Bürgschaft“)';

  @override
  String get btAwaitingTitle =>
      'Per Bluetooth empfangen — wartet auf Veröffentlichung';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount von $hash';
  }

  @override
  String btStatus(String status) {
    return 'Status: $status';
  }

  @override
  String get txRegister => 'Wallet-Registrierung';

  @override
  String get txVouch => 'Bürgschaft';

  @override
  String txMint(int period) {
    return 'Emission (Periode $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Transfer $amount → $hash';
  }

  @override
  String get mintTitle => 'Emission';

  @override
  String mintPeriod(int period) {
    return 'Periode: $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'Das Challenge-Fenster ist offen (±$hours h um den Slot). Sie können prägen.';
  }

  @override
  String get mintWindowOff =>
      'Die Prüfung des Challenge-Fensters ist deaktiviert (Sandbox). Sie können prägen.';

  @override
  String mintWindowClosed(int periods) {
    return 'Das Challenge-Fenster ist geschlossen. Der Slot im Monat ist im Voraus unvorhersehbar; ein Ausfall wird innerhalb des Nachholfensters ($periods Perioden) nachgeholt, die Mittel werden nicht blockiert.';
  }

  @override
  String mintWindowJitter(int minutes) {
    return 'Das Fenster ist offen; der Client plant die Einreichung in $minutes min (der Jitter glättet den Lastspitzenwert).';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Zuletzt beanspruchte Periode: $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Zu beanspruchende Periode: $period (abgeschlossene Perioden, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Zu prägender Betrag: $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Emission pro Periode: $amount';
  }

  @override
  String get mintConfirmTitle => 'Prägung bestätigen';

  @override
  String get mintPowStatus => 'Schweres PoW (Argon2id)…';

  @override
  String get mintNoPeriods => 'Keine Perioden zu beanspruchen';

  @override
  String get mintPublished => 'Emission veröffentlicht';

  @override
  String get transferTitle => 'Transfer';

  @override
  String get segNetwork => 'Über das Netz';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Empfängeradresse (hex)';

  @override
  String get recipientHint => '64 Hex-Zeichen';

  @override
  String get amountLabel => 'Betrag (der Empfänger erhält genau dies)';

  @override
  String feeHelper(int fee) {
    return 'Gebühr: $fee (zusätzlich zum Betrag, wird verbrannt) — vom Absender gezahlt';
  }

  @override
  String delayedWarn(int hours) {
    return 'Der Betrag übersteigt die Schwelle P: Der Transfer wird aufgeschoben. Ausführung in $hours h; während der Verzögerung kann er abgebrochen werden (Bildschirm „Aufgeschobene“).';
  }

  @override
  String get btModeHint =>
      'Bluetooth offline: Der Empfänger sieht „per Bluetooth empfangen — wartet auf Veröffentlichung“. Bis zur Bestätigung des Netzes ist es keine Zahlung (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Moratorium: Transfers sind $days Tage nach der Registrierung verfügbar.';
  }

  @override
  String get transferConfirmTitle => 'Transfer bestätigen';

  @override
  String get btConfirmTitle => 'Bluetooth-Zahlung bestätigen';

  @override
  String get transferSent => 'Transfer gesendet';

  @override
  String get transferAwaiting => 'Transfer gesendet — wartet auf Bestätigung';

  @override
  String transferDelayed(int hours) {
    return 'Transfer aufgeschoben: Ausführung in $hours h, während der Verzögerung abbrechbar';
  }

  @override
  String get signNotConfirmed => 'Signatur nicht bestätigt';

  @override
  String get btNoDevices => 'Keine gekoppelten Geräte';

  @override
  String get btSent =>
      'Per Bluetooth gesendet. Der Empfänger muss auf die Bestätigung des Netzes warten — bis zur Veröffentlichung ist es keine Zahlung';

  @override
  String btError(String error) {
    return 'Bluetooth: $error';
  }

  @override
  String get pendingTitle => 'Aufgeschobene Transfers';

  @override
  String thresholdP(int value) {
    return 'Schwelle P: $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'Die Erhöhung auf $value wirkt nach $time';
  }

  @override
  String get newThresholdLabel => 'Neue Schwelle P';

  @override
  String newThresholdHelper(int max) {
    return 'Grenze P_max = $max; Erhöhung — nach 48 h, Senkung — sofort';
  }

  @override
  String get pendingEmpty => 'Keine aktiven aufgeschobenen Transfers';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'Ausführung in $time (Gebühr $fee)';
  }

  @override
  String get cancelConfirmTitle => 'Abbruch des Transfers bestätigen';

  @override
  String get canceledMsg => 'Abgebrochen';

  @override
  String get invalidThreshold => 'Ungültige Schwelle';

  @override
  String get thresholdRaised => 'Die Erhöhung wirkt in 48 h (Schutz vor Zwang)';

  @override
  String get thresholdLowered => 'Schwelle sofort gesenkt';

  @override
  String timeHm(int h, int m) {
    return '$h h $m min';
  }

  @override
  String get vouchingTitle => 'Bürgschaft';

  @override
  String get segMyStatus => 'Mein Status';

  @override
  String get segVouch => 'Bürgen';

  @override
  String get segMySlots => 'Meine Plätze';

  @override
  String get walletActive => 'Die Wallet ist aktiv';

  @override
  String vouchProgress(int received, int k) {
    return 'Bürgschaften: $received von $k';
  }

  @override
  String get vouchFindHelpers =>
      'Finden Sie K Bürgen persönlich: Zeigen Sie ihnen den QR-Code Ihrer Adresse im Tab „Bürgen“ (Schaltfläche „Mein QR“).';

  @override
  String get vouchFlagged =>
      'Flagged-Marke: Niemand kann für Sie bürgen (Bürgschaft aus einer markierten Wallet ist verboten, Beschluss Nr. 14)';

  @override
  String get vouchConfirmTitle => 'Bürgschaft bestätigen';

  @override
  String get vouchPublished => 'Bürgschaft veröffentlicht';

  @override
  String get btWaiting => 'Warte auf Adresse per Bluetooth (60 s)…';

  @override
  String get btAddrReceived => 'Adresse per Bluetooth empfangen';

  @override
  String get vouchWarning =>
      'Persönliche Bürgschaft (QR oder Bluetooth) beweist NICHT die Anwesenheit der Person. Sie antworten für Ihre Wahl: Der Platz ist begrenzt, und wenn gegen den Gebürgten eine Nachweisung vorgelegt wird, verlieren Sie die Belohnung (kein Geld).';

  @override
  String get vouchOnlyActive =>
      'Nur eine aktive Wallet mit Alter ≥ 3 Monate und ≥ 2 abgeschlossenen Emissionsperioden kann bürgen (7.4).';

  @override
  String get candidateLabel => 'Kandidatenadresse (hex)';

  @override
  String get candidateHint => '64 Hex-Zeichen — vom QR oder Bluetooth';

  @override
  String get myQrHint => 'Meine Adresse — zeigen Sie sie dem Bürgen';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Plätze: $occupied von $capacity belegt';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Frei: $free. Statistik des Bürgen: für $total gebürgt, aktiv $active (öffentliches Ansehen, 7.4)';
  }

  @override
  String get slotsEmpty => 'Die Plätze sind frei';

  @override
  String slotActiveSub(int periods) {
    return 'der Kandidat ist aktiv: der Platz wird nach $periods Emissionsperioden ohne Nachweisung frei';
  }

  @override
  String slotInactiveSub(int days) {
    return 'der Kandidat ist nicht aktiv: der Platz wird in $days Tagen frei';
  }

  @override
  String get registryTitle => 'Netzregister';

  @override
  String registryRowSub(String status, int age) {
    return '$status • Alter $age Tage';
  }

  @override
  String chipAge(int age) {
    return 'Alter $age Tage';
  }

  @override
  String get chipSeed => 'Seed-Kreis';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'öffentlicher Vormund';

  @override
  String vouchDepthLabel(int depth) {
    return 'Tiefe des Bürgschaftsgraphen: $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Tiefe des Bürgschaftsgraphen: keine Verbindung zum Seed-Kreis';

  @override
  String rewardLine(int periods) {
    return 'Belohnung an Bürgen: $periods abgeschlossene Emissionsperioden';
  }

  @override
  String asVoucher(int count, int free) {
    return 'Als Bürge: für $count (freie Plätze: $free)';
  }

  @override
  String get publicHistory => 'Öffentlicher Verlauf';

  @override
  String get migrationTitle => 'Migration';

  @override
  String get migrationIntro =>
      'Wallet-Wiederherstellung: Die Vormünde werden im Voraus benannt (bei der Erstellung). Persönliche — 2 von 3; mit öffentlichen — 3 von 5 (die Schwellen kombinieren sich).';

  @override
  String migrationTo(String hash) {
    return 'Migration zu $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'Ausführung in $days Tagen (Verzögerung $delay Tage); Vormund-Signaturen: $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Öffentliche Vormünde des Netzes (guardian_offer, 11)';

  @override
  String get noGuardians =>
      'Niemand hat sich bisher als öffentlicher Vormund erklärt';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'Alter $age Tage, Mündel: $wards';
  }

  @override
  String get restartIntro =>
      'Wenn es keine Vormünde gibt: Das Zurücksetzen ist eine gewöhnliche Registrierung mit neuem Schlüssel und neuen K Bürgschaften. Die alte Wallet bleibt im Register unberührt und unzugänglich.';

  @override
  String get cancelMigrationTitle =>
      'Abbruch der Migration bestätigen (alter Schlüssel)';

  @override
  String get migrationCanceled => 'Migration abgebrochen';

  @override
  String get restartDialogTitle => 'Verlust der Mittel';

  @override
  String get restartDialogBody =>
      'Das Zurücksetzen bringt nichts über die Registrierung einer neuen Wallet hinaus: Die Mittel der alten Adresse sind verloren, der Verlauf wird nicht übertragen. Die Migration mit Vormünden überträgt Alter, Emissionsverlauf und Bürgschaften.';

  @override
  String get restartFlowNote =>
      'Zurücksetzen: Löschen Sie die App-Daten und durchlaufen Sie die Wallet-Erstellung erneut (neuer Schlüssel, neue K Bürgschaften).';

  @override
  String get startMigrationTitle => 'Start der Migration';

  @override
  String get startMigrationIntro =>
      'Sie sind auf dem NEUEN Gerät. Geben Sie die alte Wallet-Adresse und die Vormund-Signaturen ein (jeder Vormund signiert die Erklärung in seiner eigenen App — Tab „Migration eines anderen genehmigen“).';

  @override
  String get oldAddressLabel => 'Alte Adresse (hex)';

  @override
  String get newAddressLabel => 'Neue Adresse (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Signaturen gesammelt: $count (persönliche Schwelle $personal von $personalCount; mit öffentlichen — $public von $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Vormund-Adresse (hex)';

  @override
  String get guardianSigLabel => 'Vormund-Signatur (hex)';

  @override
  String get migrationConfirmTitle => 'Migration bestätigen';

  @override
  String migrationStarted(int days) {
    return 'Migration gestartet: Ausführung in $days Tagen; der alte Schlüssel kann während der Verzögerung abbrechen';
  }

  @override
  String get approveTitle => 'Genehmigung der Migration (Vormund)';

  @override
  String get approveIntro =>
      'Prüfen Sie die Identität des Mündels über Ihre eigenen Kanäle. Die Signatur bindet das Adresspaar und den Nonce; übergeben Sie sie dem Mündel persönlich.';

  @override
  String get nonceLabel => 'Nonce (vom Mündel)';

  @override
  String get wardSignTitle =>
      'Migration des Mündels bestätigen (Sie sind der Vormund)';

  @override
  String get sigToCopy => 'Signatur (dem Mündel übergeben):';

  @override
  String get nonceToCopy => 'Migrations-Nonce (den Vormünden übergeben):';

  @override
  String get settingsTitle => 'Einstellungen';

  @override
  String networkLine(String name) {
    return 'Netz: $name';
  }

  @override
  String get paramUnit => 'Einheit';

  @override
  String get paramTransfer => 'Transfer';

  @override
  String get paramIssuance => 'Emission';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount alle $months Monate';
  }

  @override
  String get paramVouchK => 'K (Bürgschaften)';

  @override
  String get paramThresholdP => 'Schwelle P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Verzögerung D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours h';
  }

  @override
  String get paramMoratorium => 'Moratorium';

  @override
  String paramMoratoriumValue(String days) {
    return '$days Tage';
  }

  @override
  String get paramDemurrage => 'Demurrage';

  @override
  String get demurrageOn => 'aktiviert';

  @override
  String get demurrageOff => 'deaktiviert';

  @override
  String get paramAccrual => 'Nachholfenster';

  @override
  String paramAccrualValue(String periods) {
    return '$periods Perioden';
  }

  @override
  String get paramGuardians => 'Vormünde';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'persönliche $pt von $pc; öffentliche $pubt von $pubc';
  }

  @override
  String get paramMigration => 'Migration';

  @override
  String paramMigrationValue(String days) {
    return 'Verzögerung $days Tage';
  }

  @override
  String get paramEpoch => 'Checkpoint-Epoche';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds s';
  }

  @override
  String get sandboxNote => 'Sandbox (synthetische Attestierung)';

  @override
  String get guardianOfferTitle => 'Als öffentlicher Vormund erklären';

  @override
  String guardianAlready(int wards) {
    return 'Sie sind bereits öffentlicher Vormund (Mündel: $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Anforderungen: active, Alter ≥ $days Tage';
  }

  @override
  String get guardianConfirmTitle =>
      'Bestätigen, öffentlicher Vormund zu werden';

  @override
  String get guardianPublished => 'Erklärung veröffentlicht';

  @override
  String get nodeRunning => 'Der Knoten läuft';

  @override
  String get nodeStopped => 'Der Knoten ist gestoppt';

  @override
  String get nodeLevelHint =>
      'Level full — beim Laden und im WLAN, sonst light (12.2)';

  @override
  String get lockApp => 'App sperren';

  @override
  String get relockNote => 'Gesperrt. Starten Sie die App neu.';

  @override
  String get languageTitle => 'Sprache';

  @override
  String get languageSystem => 'System';

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
  String get languageFr => 'Français';

  @override
  String get languageDe => 'Deutsch';

  @override
  String get languageIt => 'Italiano';

  @override
  String get languageJa => '日本語';

  @override
  String get recoveryTitle => 'Wiederherstellungskarte';

  @override
  String get recoveryIntro =>
      'Bewahren Sie diese Daten außerhalb des Telefons auf: Sie werden benötigt, um die Wallet zu wiederherstellen, wenn das Gerät verloren geht (Migration mit Vormünden, Abschnitt 11).';

  @override
  String get recoveryAddress => 'Wallet-Adresse';

  @override
  String get recoveryNetwork => 'Netz';

  @override
  String get recoveryGuardians => 'Ihre Vormünde';

  @override
  String get recoveryNoGuardians =>
      'Keine Vormünde benannt: Die Wiederherstellung ist unmöglich, es bleibt nur das Zurücksetzen (gewöhnliche Registrierung mit neuen Bürgschaften).';

  @override
  String get recoveryHowToSave =>
      'So bewahren: Auf Papier kopieren oder an einem sicheren Ort außerhalb des Telefons aufbewahren; nicht in der Cloud speichern. Das Kartenbild lässt sich über „Teilen“ drucken.';

  @override
  String get recoveryNeverShare =>
      'Teilen Sie die Migrationsdaten (den Nonce und die Vormund-Signaturen) niemals — nur direkt an Ihren Vormund und nur während der Migration; löschen Sie sie danach aus Ihren Nachrichten. Dritten zeigen Sie nur die Adresse und den QR dieser Karte.';

  @override
  String get recoverySavedDone => 'Gespeichert — weiter';

  @override
  String get settingsRecovery => 'Wiederherstellungskarte';

  @override
  String get recoveryApkNote =>
      'Bewahren Sie die Installationsdatei (APK) desselben Netzes im Voraus auf: Ohne sie kann die Wiederherstellung auf einem neuen Gerät unmöglich sein.';

  @override
  String get recoveryShareBtn => 'Karte teilen (Bild)';

  @override
  String get guardiansTitle => 'Vormünde';

  @override
  String get guardiansIntro =>
      'Persönliche Vormünde bestätigen die Migration (2 von 3). Sie können neu gewählt werden: Die Änderung wirkt nach 48 h; bis dahin gilt die alte Liste.';

  @override
  String get guardiansEntrySub => 'Persönliche Vormünde: Neuwahl, 48-h-Fenster';

  @override
  String get guardiansCurrent => 'Aktuelle Vormünde';

  @override
  String get guardiansNone => 'Keine Vormünde benannt';

  @override
  String get guardiansReelect => 'Neu wählen';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return 'Vormundswechsel genutzt: $used von $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return 'Die neue Liste wirkt in $hours h';
  }

  @override
  String get guardiansCancelBtn => 'Änderung abbrechen';

  @override
  String get guardiansStatusCanceled => 'Abgebrochen';

  @override
  String get guardiansStatusEffective => 'Wirksam';

  @override
  String get guardiansStatusExpired =>
      'Abgelaufen: das Mündel-Limit des Vormunds ist überschritten';

  @override
  String get reelectTitle => 'Vormünde neu wählen';

  @override
  String get reelectIntro =>
      'Geben Sie die neuen Vormünde ein (bis zu 3, hex, einer pro Zeile). Übergeben Sie den Hash der Änderung jedem neuen Vormund: Er signiert ihn in seiner App („Zustimmung des Vormunds“) und gibt Ihnen den Hex der Signatur zurück.';

  @override
  String get newGuardiansLabel => 'Neue Vormünde (hex, einer pro Zeile)';

  @override
  String get changeHashToCopy =>
      'Änderungs-Hash (an die neuen Vormünde übergeben):';

  @override
  String guardiansApprovals(int count, int need) {
    return 'Zustimmungen gesammelt: $count von $need';
  }

  @override
  String get btnPublishChange => 'Änderung veröffentlichen';

  @override
  String get changePublished => 'Änderung veröffentlicht: wirkt in 48 h';

  @override
  String get changeConfirmTitle => 'Vormundswechsel bestätigen (PIN)';

  @override
  String get cancelChangeConfirmTitle =>
      'Abbruch der Änderung bestätigen (PIN)';

  @override
  String get changeCanceled => 'Änderung abgebrochen';

  @override
  String get consentTitle => 'Zustimmung des Vormunds';

  @override
  String get consentIntro =>
      'Sie bestätigen, Vormund des Mündels zu werden. Geben Sie die Adresse des Mündels und den Hash der Änderung ein und signieren Sie den Hash. Geben Sie den Hex der Signatur an das Mündel zurück.';

  @override
  String get wardAddressLabel => 'Adresse des Mündels (hex)';

  @override
  String get changeHashLabel => 'Änderungs-Hash (hex)';

  @override
  String get consentSignTitle =>
      'Vormundswechsel des Mündels bestätigen (Sie sind der Vormund)';

  @override
  String get biometricTimeout =>
      'Die Biometrie antwortet nicht (Zeitüberschreitung) — versuchen Sie es erneut';

  @override
  String hexChunksCount(int n, int total) {
    return 'Eingegeben $n von $total';
  }

  @override
  String hexChunksSigCount(int n) {
    return 'Eingegebene Zeichen: $n';
  }

  @override
  String hexLenError(int total) {
    return 'Genau $total Hex-Zeichen erwartet';
  }

  @override
  String get hexSigEvenError =>
      'Die Signatur muss eine gerade Anzahl von Hex-Zeichen haben';

  @override
  String get hexCharsDropped => 'Ungültige Zeichen entfernt (nur 0-9, a-f)';

  @override
  String get aboutNetwork => 'Über das Netz';

  @override
  String networkSlug(String slug) {
    return 'slug: $slug';
  }

  @override
  String get paramRecurring => 'Periodische Einheiten (recurring)';

  @override
  String get recurringOn => 'aktiviert (Erlöschen an der Periodengrenze)';

  @override
  String get recurringOff => 'deaktiviert';

  @override
  String get recurringNote =>
      'Jede Einheit trägt die Periodenmarke: Der Transfer einer Einheit der Periode P wird nach der Grenze der Periode P+1 abgelehnt. Der Saldo wird nicht zurückgesetzt — der Verlauf bleibt öffentlich und überprüfbar.';

  @override
  String get burnedUnitError =>
      'Die Einheit ist erloschen: Die Grenze der Periode P+1 ist verstrichen. Transfer und Einlösung sind unmöglich, aber der Saldo wird nicht zurückgesetzt — der Verlauf bleibt erhalten (E_BURNED).';

  @override
  String obNetworkLine(String pair) {
    return 'Netz: $pair';
  }
}
