// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Trust Networks';

  @override
  String get navWallet => 'Wallet';

  @override
  String get navVouching => 'Vouching';

  @override
  String get navRegistry => 'Registry';

  @override
  String get navMigration => 'Migration';

  @override
  String get navSettings => 'Settings';

  @override
  String get btnNext => 'Next';

  @override
  String get btnCancel => 'Cancel';

  @override
  String get btnConfirm => 'Confirm';

  @override
  String get btnRefresh => 'Refresh';

  @override
  String get btnSkip => 'Skip';

  @override
  String get btnLogin => 'Unlock';

  @override
  String get btnFingerprint => 'Fingerprint';

  @override
  String get btnSend => 'Send';

  @override
  String get btnSendBt => 'Send via Bluetooth';

  @override
  String get btnVouch => 'Vouch';

  @override
  String get btnMyQr => 'My QR';

  @override
  String get btnHideQr => 'Hide my QR';

  @override
  String get btnBtReceive => 'Receive via BT';

  @override
  String get btnMint => 'Mint (challenge)';

  @override
  String get btnMintDo => 'Mint (heavy PoW)';

  @override
  String get btnTransfer => 'Transfer';

  @override
  String get btnPublishAwaiting => 'Publish (network available)';

  @override
  String get btnChangeThreshold => 'Change threshold';

  @override
  String get btnCancelTransfer => 'Cancel';

  @override
  String get btnRegister => 'Register wallet';

  @override
  String get btnStartMigration => 'Start migration (new device)';

  @override
  String get btnApproveMigration =>
      'Approve someone\'s migration (I am a guardian)';

  @override
  String get btnRestart => 'Restart wallet';

  @override
  String get btnUnderstandRestart => 'I understand, restart';

  @override
  String get btnAddSig => 'Add signature';

  @override
  String get btnPublishProposal => 'Publish proposal';

  @override
  String get btnSignBiometric => 'Sign (biometrics)';

  @override
  String get enterPin => 'Enter PIN';

  @override
  String get errWrongPin => 'Wrong PIN';

  @override
  String get confirmOperation => 'Confirm operation';

  @override
  String get keyUnlockTitle => 'Unlock key';

  @override
  String rejectedWithCode(String code) {
    return 'Rejected: $code';
  }

  @override
  String errorWith(String e) {
    return 'Error: $e';
  }

  @override
  String get obTitle => 'Creating a wallet';

  @override
  String get obBioRequired =>
      'The wallet requires a hardware key with biometrics (BIOMETRIC_STRONG). Registration is impossible without a configured fingerprint.';

  @override
  String get obBioNotSetup =>
      'No fingerprint configured. Open the system settings screen and add a fingerprint.';

  @override
  String get obBioSetup => 'Set up fingerprint';

  @override
  String get obPinIntro =>
      'PIN protects the app. A duress PIN (optional) opens a decoy mode: the same interface, but a plausible small balance; the signing key is not used.';

  @override
  String get obPinNormal => 'PIN (normal)';

  @override
  String get obWantDuress => 'Set a duress PIN';

  @override
  String get obDuressPin => 'Duress PIN';

  @override
  String get obDecoyLegend => 'Decoy mode legend (what the coercer will see):';

  @override
  String get obDecoyBalance => 'Plausible balance (not 0)';

  @override
  String get obDecoyHistory => 'History (lines: date — amount — recipient)';

  @override
  String get pinMinDigits => 'minimum 4 digits';

  @override
  String get duressPinMinDigits => 'duress PIN: minimum 4 digits';

  @override
  String get pinsMatch => 'PINs match';

  @override
  String get obGuardiansIntro =>
      'Assign personal guardians (2 of 3). They will be able to approve migration to a new device. Adding a new fingerprint invalidates the key — without guardians access to the wallet is lost.';

  @override
  String get obGuardiansEmpty =>
      'There are no other members in the network yet. You may skip guardians for now, but the offer is mandatory: the list cannot be changed later (only at creation).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, age $age days';
  }

  @override
  String obSelectedCount(int count) {
    return 'Selected: $count';
  }

  @override
  String get regStatusReady => 'Ready to register';

  @override
  String get regStatusPow => 'Attestation and PoW (may take a while)…';

  @override
  String get regConfirmTitle => 'Confirm registration';

  @override
  String get regStatusDone => 'Wallet registered';

  @override
  String pendingCount(int count) {
    return 'Delayed transfers: $count';
  }

  @override
  String get pendingHint =>
      'Large transfers execute after delay D; they can be cancelled within D';

  @override
  String get historyTitle => 'History';

  @override
  String get historyEmpty => 'Nothing yet';

  @override
  String get effBalance => 'effective balance';

  @override
  String get effBalanceDem => 'effective balance (demurrage applied)';

  @override
  String lockedReserve(int amount) {
    return 'reserved for delayed transfers: $amount';
  }

  @override
  String get flaggedNotice =>
      'Wallet is flagged — the mark is public, no verdicts';

  @override
  String get registeredNeedK =>
      'Status registered: activation requires K vouches (the \"Vouching\" tab)';

  @override
  String get btAwaitingTitle => 'Received via Bluetooth — awaiting publication';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount from $hash';
  }

  @override
  String btStatus(String status) {
    return 'status: $status';
  }

  @override
  String get txRegister => 'Wallet registration';

  @override
  String get txVouch => 'Vouching';

  @override
  String txMint(int period) {
    return 'Mint (period $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Transfer $amount → $hash';
  }

  @override
  String get mintTitle => 'Mint';

  @override
  String mintPeriod(int period) {
    return 'Period: $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'Challenge window is open (±$hours h around the slot). You can mint.';
  }

  @override
  String get mintWindowOff =>
      'Challenge window check is disabled (sandbox). You can mint.';

  @override
  String mintWindowClosed(int periods) {
    return 'Challenge window is closed. The slot inside the month is unpredictable in advance; a miss is carried over within the accrual window ($periods periods), funds are not blocked.';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Last claimed period: $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Period to claim: $period (completed periods, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Amount to mint: $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Issuance per period: $amount';
  }

  @override
  String get mintConfirmTitle => 'Confirm mint';

  @override
  String get mintPowStatus => 'Heavy PoW (Argon2id)…';

  @override
  String get mintNoPeriods => 'No periods to claim';

  @override
  String get mintPublished => 'Mint published';

  @override
  String get transferTitle => 'Transfer';

  @override
  String get segNetwork => 'Via network';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Recipient address (hex)';

  @override
  String get recipientHint => '64 hex characters';

  @override
  String get amountLabel => 'Amount (the recipient gets exactly this)';

  @override
  String feeHelper(int fee) {
    return 'Fee: $fee (on top of the amount, burned) — paid by the sender';
  }

  @override
  String delayedWarn(int hours) {
    return 'The amount is above threshold P: the transfer will be delayed. Execution in $hours h; it can be cancelled during the delay (the \"Delayed\" screen).';
  }

  @override
  String get btModeHint =>
      'Bluetooth offline: the recipient will see \"received via Bluetooth — awaiting publication\". Until the network confirms, it is not a payment (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Moratorium: transfers are available $days days after registration.';
  }

  @override
  String get transferConfirmTitle => 'Confirm transfer';

  @override
  String get btConfirmTitle => 'Confirm Bluetooth payment';

  @override
  String get transferSent => 'Transfer sent';

  @override
  String transferDelayed(int hours) {
    return 'Transfer delayed: execution in $hours h, it can be cancelled during the delay';
  }

  @override
  String get signNotConfirmed => 'Signature not confirmed';

  @override
  String get btNoDevices => 'No paired devices';

  @override
  String get btSent =>
      'Sent via Bluetooth. The recipient must wait for network confirmation — until publication it is not a payment';

  @override
  String btError(String error) {
    return 'Bluetooth: $error';
  }

  @override
  String get pendingTitle => 'Delayed transfers';

  @override
  String thresholdP(int value) {
    return 'Threshold P: $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'Increase to $value takes effect after $time';
  }

  @override
  String get newThresholdLabel => 'New threshold P';

  @override
  String newThresholdHelper(int max) {
    return 'limit P_max = $max; increase — after 48 h, decrease — immediately';
  }

  @override
  String get pendingEmpty => 'No active delayed transfers';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'execution in $time (fee $fee)';
  }

  @override
  String get cancelConfirmTitle => 'Confirm transfer cancellation';

  @override
  String get canceledMsg => 'Cancelled';

  @override
  String get invalidThreshold => 'Invalid threshold';

  @override
  String get thresholdRaised =>
      'The increase takes effect in 48 h (coercion protection)';

  @override
  String get thresholdLowered => 'Threshold lowered immediately';

  @override
  String timeHm(int h, int m) {
    return '$h h $m min';
  }

  @override
  String get vouchingTitle => 'Vouching';

  @override
  String get segMyStatus => 'My status';

  @override
  String get segVouch => 'Vouch';

  @override
  String get segMySlots => 'My slots';

  @override
  String get walletActive => 'Wallet is active';

  @override
  String vouchProgress(int received, int k) {
    return 'Vouches: $received of $k';
  }

  @override
  String get vouchFindHelpers =>
      'Find K vouchers in person: show them the QR code of your address on the \"Vouch\" tab (the \"My QR\" button).';

  @override
  String get vouchFlagged =>
      'Flagged mark: nobody can vouch for you (vouching from a flagged wallet is forbidden, decision No. 14)';

  @override
  String get vouchConfirmTitle => 'Confirm vouching';

  @override
  String get vouchPublished => 'Vouch published';

  @override
  String get btWaiting => 'Waiting for address via Bluetooth (60 s)…';

  @override
  String get btAddrReceived => 'Address received via Bluetooth';

  @override
  String get vouchWarning =>
      'In-person vouching (QR or Bluetooth) does NOT prove the person\'s presence. You are responsible for the choice: the slot is limited, and if evidence is presented against the vouched person you lose the reward (not money).';

  @override
  String get vouchOnlyActive =>
      'Only an active wallet aged ≥ 3 months with ≥ 2 completed mint periods can vouch (7.4).';

  @override
  String get candidateLabel => 'Candidate address (hex)';

  @override
  String get candidateHint => '64 hex characters — from QR or Bluetooth';

  @override
  String get myQrHint => 'My address — show it to the voucher';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Slots: $occupied occupied of $capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Free: $free. Voucher statistics: vouched for $total, active $active (public reputation, 7.4)';
  }

  @override
  String get slotsEmpty => 'Slots are free';

  @override
  String slotActiveSub(int periods) {
    return 'candidate is active: the slot frees after $periods mint periods without evidence';
  }

  @override
  String slotInactiveSub(int days) {
    return 'candidate is not active: the slot frees in $days days';
  }

  @override
  String get registryTitle => 'Network registry';

  @override
  String registryRowSub(String status, int age) {
    return '$status • age $age days';
  }

  @override
  String chipAge(int age) {
    return 'age $age days';
  }

  @override
  String get chipSeed => 'seed circle';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'public guardian';

  @override
  String vouchDepthLabel(int depth) {
    return 'Vouching graph depth: $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Vouching graph depth: not linked to the seed circle';

  @override
  String rewardLine(int periods) {
    return 'Reward to vouchers: $periods completed mint periods';
  }

  @override
  String asVoucher(int count, int free) {
    return 'As voucher: for $count (free slots: $free)';
  }

  @override
  String get publicHistory => 'Public history';

  @override
  String get migrationTitle => 'Migration';

  @override
  String get migrationIntro =>
      'Wallet recovery: guardians are assigned in advance (at creation). Personal — 2 of 3; with public ones — 3 of 5 (thresholds combine).';

  @override
  String migrationTo(String hash) {
    return 'Migration to $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'execution in $days days (delay $delay days); guardian signatures: $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Public guardians of the network (guardian_offer, 11)';

  @override
  String get noGuardians =>
      'Nobody has declared themselves a public guardian yet';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'age $age days, wards: $wards';
  }

  @override
  String get restartIntro =>
      'If there are no guardians: a restart is a regular registration with a new key and new K vouches. The old wallet remains in the registry untouched and inaccessible.';

  @override
  String get cancelMigrationTitle => 'Confirm migration cancellation (old key)';

  @override
  String get migrationCanceled => 'Migration cancelled';

  @override
  String get restartDialogTitle => 'Loss of funds';

  @override
  String get restartDialogBody =>
      'A restart gives nothing beyond registering a new wallet: funds at the old address are lost, history is not transferred. Migration with guardians transfers age, mint history and vouches.';

  @override
  String get restartFlowNote =>
      'Restart: delete the app data and go through wallet creation again (new key, new K vouches).';

  @override
  String get startMigrationTitle => 'Starting migration';

  @override
  String get startMigrationIntro =>
      'You are on the NEW device. Enter the old wallet address and guardian signatures (each guardian signs the statement in their own app — the \"Approve someone\'s migration\" tab).';

  @override
  String get oldAddressLabel => 'Old address (hex)';

  @override
  String get newAddressLabel => 'New address (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Signatures collected: $count (personal threshold $personal of $personalCount; with public — $public of $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Guardian address (hex)';

  @override
  String get guardianSigLabel => 'Guardian signature (hex)';

  @override
  String get migrationConfirmTitle => 'Confirm migration';

  @override
  String migrationStarted(int days) {
    return 'Migration started: execution in $days days; the old key can cancel during the delay';
  }

  @override
  String get approveTitle => 'Migration approval (guardian)';

  @override
  String get approveIntro =>
      'Verify the ward\'s identity through your own channels. The signature binds the pair of addresses and the nonce; hand it to the ward in person.';

  @override
  String get nonceLabel => 'Nonce (from the ward)';

  @override
  String get wardSignTitle =>
      'Confirm the ward\'s migration (you are the guardian)';

  @override
  String get sigToCopy => 'Signature (hand it to the ward):';

  @override
  String get nonceToCopy => 'Migration nonce (hand it to the guardians):';

  @override
  String get settingsTitle => 'Settings';

  @override
  String networkLine(String name) {
    return 'Network: $name';
  }

  @override
  String get paramUnit => 'Unit';

  @override
  String get paramTransfer => 'Transfer';

  @override
  String get paramIssuance => 'Issuance';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount per $months months';
  }

  @override
  String get paramVouchK => 'K (vouching)';

  @override
  String get paramThresholdP => 'Threshold P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Delay D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours h';
  }

  @override
  String get paramMoratorium => 'Moratorium';

  @override
  String paramMoratoriumValue(String days) {
    return '$days days';
  }

  @override
  String get paramDemurrage => 'Demurrage';

  @override
  String get demurrageOn => 'enabled';

  @override
  String get demurrageOff => 'disabled';

  @override
  String get paramAccrual => 'Accrual window';

  @override
  String paramAccrualValue(String periods) {
    return '$periods periods';
  }

  @override
  String get paramGuardians => 'Guardians';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'personal $pt of $pc; public $pubt of $pubc';
  }

  @override
  String get paramMigration => 'Migration';

  @override
  String paramMigrationValue(String days) {
    return 'delay $days days';
  }

  @override
  String get paramEpoch => 'Checkpoint epoch';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds s';
  }

  @override
  String get sandboxNote => 'Sandbox (synthetic attestation)';

  @override
  String get guardianOfferTitle => 'Declare yourself a public guardian';

  @override
  String guardianAlready(int wards) {
    return 'You are already a public guardian (wards: $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Requirements: active, age ≥ $days days';
  }

  @override
  String get guardianConfirmTitle => 'Confirm becoming a public guardian';

  @override
  String get guardianPublished => 'Declaration published';

  @override
  String get nodeRunning => 'Node is running';

  @override
  String get nodeStopped => 'Node is stopped';

  @override
  String get nodeLevelHint =>
      'Full level — when charging and on Wi-Fi, otherwise light (12.2)';

  @override
  String get lockApp => 'Lock the app';

  @override
  String get relockNote => 'Locked. Restart the app.';

  @override
  String get languageTitle => 'Language';

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
  String get recoveryTitle => 'Recovery card';

  @override
  String get recoveryIntro =>
      'Save this data outside the phone: it is needed to recover the wallet if the device is lost (migration with guardians, section 11).';

  @override
  String get recoveryAddress => 'Wallet address';

  @override
  String get recoveryNetwork => 'Network';

  @override
  String get recoveryGuardians => 'Your guardians';

  @override
  String get recoveryNoGuardians =>
      'No guardians assigned: recovery is impossible, only a restart is available (regular registration with new vouches).';

  @override
  String get recoveryHowToSave =>
      'How to save: write it on paper or keep it in a safe place outside the phone; do not store it in the cloud. The card image can be printed via \"Share\".';

  @override
  String get recoveryNeverShare =>
      'Never share migration data (the nonce and guardian signatures) — only directly to your guardian and only for the duration of the migration; delete it from your messages afterwards. Only the address and this card\'s QR may be shown to others.';

  @override
  String get recoverySavedDone => 'Saved — continue';

  @override
  String get settingsRecovery => 'Recovery card';

  @override
  String get recoveryApkNote =>
      'Save the installable file (APK) of the same network in advance: without it, recovery on a new device may be impossible.';

  @override
  String get recoveryShareBtn => 'Share card (image)';

  @override
  String get guardiansTitle => 'Guardians';

  @override
  String get guardiansIntro =>
      'Personal guardians confirm migration (2 of 3). You can re-elect them: the change takes effect after 48 h; until then the old list applies.';

  @override
  String get guardiansEntrySub =>
      'Personal guardians: re-election, 48 h window';

  @override
  String get guardiansCurrent => 'Current guardians';

  @override
  String get guardiansNone => 'No guardians assigned';

  @override
  String get guardiansReelect => 'Re-elect';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return 'Guardian changes used: $used of $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return 'The new list takes effect in $hours h';
  }

  @override
  String get guardiansCancelBtn => 'Cancel change';

  @override
  String get guardiansStatusCanceled => 'Canceled';

  @override
  String get guardiansStatusEffective => 'Took effect';

  @override
  String get guardiansStatusExpired =>
      'Expired: the guardian\'s ward limit is exceeded';

  @override
  String get reelectTitle => 'Re-elect guardians';

  @override
  String get reelectIntro =>
      'Enter the new guardians (up to 3, hex, one per line). Pass the change hash to each new guardian: they sign it in their app (\"Guardian consent\") and give you the signature hex.';

  @override
  String get newGuardiansLabel => 'New guardians (hex, one per line)';

  @override
  String get changeHashToCopy => 'Change hash (pass it to the new guardians):';

  @override
  String guardiansApprovals(int count, int need) {
    return 'Consents collected: $count of $need';
  }

  @override
  String get btnPublishChange => 'Publish change';

  @override
  String get changePublished => 'Change published: takes effect in 48 h';

  @override
  String get changeConfirmTitle => 'Confirm the guardian change (PIN)';

  @override
  String get cancelChangeConfirmTitle => 'Confirm cancelling the change (PIN)';

  @override
  String get changeCanceled => 'Change canceled';

  @override
  String get consentTitle => 'Guardian consent';

  @override
  String get consentIntro =>
      'You are confirming becoming a guardian of the ward. Enter the ward\'s address and the change hash, and sign the hash. Give the signature hex back to the ward.';

  @override
  String get wardAddressLabel => 'Ward\'s address (hex)';

  @override
  String get changeHashLabel => 'Change hash (hex)';

  @override
  String get consentSignTitle =>
      'Confirm the ward\'s guardian change (you are the guardian)';

  @override
  String get biometricTimeout =>
      'Biometrics not responding (timeout) — try again';

  @override
  String hexChunksCount(int n, int total) {
    return 'Entered $n of $total';
  }

  @override
  String hexChunksSigCount(int n) {
    return 'Characters entered: $n';
  }

  @override
  String hexLenError(int total) {
    return 'Expected exactly $total hex characters';
  }

  @override
  String get hexSigEvenError =>
      'Signature must have an even number of hex characters';

  @override
  String get hexCharsDropped => 'Invalid characters removed (only 0-9, a-f)';
}
