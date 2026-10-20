import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_es.dart';
import 'app_localizations_pt.dart';
import 'app_localizations_ru.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('es'),
    Locale('pt'),
    Locale('ru'),
    Locale('zh'),
  ];

  /// No description provided for @appTitle.
  ///
  /// In en, this message translates to:
  /// **'Trust Networks'**
  String get appTitle;

  /// No description provided for @navWallet.
  ///
  /// In en, this message translates to:
  /// **'Wallet'**
  String get navWallet;

  /// No description provided for @navVouching.
  ///
  /// In en, this message translates to:
  /// **'Vouching'**
  String get navVouching;

  /// No description provided for @navRegistry.
  ///
  /// In en, this message translates to:
  /// **'Registry'**
  String get navRegistry;

  /// No description provided for @navMigration.
  ///
  /// In en, this message translates to:
  /// **'Migration'**
  String get navMigration;

  /// No description provided for @navSettings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get navSettings;

  /// No description provided for @btnNext.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get btnNext;

  /// No description provided for @btnCancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get btnCancel;

  /// No description provided for @btnConfirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get btnConfirm;

  /// No description provided for @btnRefresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get btnRefresh;

  /// No description provided for @btnSkip.
  ///
  /// In en, this message translates to:
  /// **'Skip'**
  String get btnSkip;

  /// No description provided for @btnLogin.
  ///
  /// In en, this message translates to:
  /// **'Unlock'**
  String get btnLogin;

  /// No description provided for @btnFingerprint.
  ///
  /// In en, this message translates to:
  /// **'Fingerprint'**
  String get btnFingerprint;

  /// No description provided for @btnSend.
  ///
  /// In en, this message translates to:
  /// **'Send'**
  String get btnSend;

  /// No description provided for @btnSendBt.
  ///
  /// In en, this message translates to:
  /// **'Send via Bluetooth'**
  String get btnSendBt;

  /// No description provided for @btnVouch.
  ///
  /// In en, this message translates to:
  /// **'Vouch'**
  String get btnVouch;

  /// No description provided for @btnMyQr.
  ///
  /// In en, this message translates to:
  /// **'My QR'**
  String get btnMyQr;

  /// No description provided for @btnHideQr.
  ///
  /// In en, this message translates to:
  /// **'Hide my QR'**
  String get btnHideQr;

  /// No description provided for @btnBtReceive.
  ///
  /// In en, this message translates to:
  /// **'Receive via BT'**
  String get btnBtReceive;

  /// No description provided for @btnMint.
  ///
  /// In en, this message translates to:
  /// **'Mint (challenge)'**
  String get btnMint;

  /// No description provided for @btnMintDo.
  ///
  /// In en, this message translates to:
  /// **'Mint (heavy PoW)'**
  String get btnMintDo;

  /// No description provided for @btnTransfer.
  ///
  /// In en, this message translates to:
  /// **'Transfer'**
  String get btnTransfer;

  /// No description provided for @btnPublishAwaiting.
  ///
  /// In en, this message translates to:
  /// **'Publish (network available)'**
  String get btnPublishAwaiting;

  /// No description provided for @btnChangeThreshold.
  ///
  /// In en, this message translates to:
  /// **'Change threshold'**
  String get btnChangeThreshold;

  /// No description provided for @btnCancelTransfer.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get btnCancelTransfer;

  /// No description provided for @btnRegister.
  ///
  /// In en, this message translates to:
  /// **'Register wallet'**
  String get btnRegister;

  /// No description provided for @btnStartMigration.
  ///
  /// In en, this message translates to:
  /// **'Start migration (new device)'**
  String get btnStartMigration;

  /// No description provided for @btnApproveMigration.
  ///
  /// In en, this message translates to:
  /// **'Approve someone\'s migration (I am a guardian)'**
  String get btnApproveMigration;

  /// No description provided for @btnRestart.
  ///
  /// In en, this message translates to:
  /// **'Restart wallet'**
  String get btnRestart;

  /// No description provided for @btnUnderstandRestart.
  ///
  /// In en, this message translates to:
  /// **'I understand, restart'**
  String get btnUnderstandRestart;

  /// No description provided for @btnAddSig.
  ///
  /// In en, this message translates to:
  /// **'Add signature'**
  String get btnAddSig;

  /// No description provided for @btnPublishProposal.
  ///
  /// In en, this message translates to:
  /// **'Publish proposal'**
  String get btnPublishProposal;

  /// No description provided for @btnSignBiometric.
  ///
  /// In en, this message translates to:
  /// **'Sign (biometrics)'**
  String get btnSignBiometric;

  /// No description provided for @enterPin.
  ///
  /// In en, this message translates to:
  /// **'Enter PIN'**
  String get enterPin;

  /// No description provided for @errWrongPin.
  ///
  /// In en, this message translates to:
  /// **'Wrong PIN'**
  String get errWrongPin;

  /// No description provided for @confirmOperation.
  ///
  /// In en, this message translates to:
  /// **'Confirm operation'**
  String get confirmOperation;

  /// No description provided for @keyUnlockTitle.
  ///
  /// In en, this message translates to:
  /// **'Unlock key'**
  String get keyUnlockTitle;

  /// No description provided for @rejectedWithCode.
  ///
  /// In en, this message translates to:
  /// **'Rejected: {code}'**
  String rejectedWithCode(String code);

  /// No description provided for @errorWith.
  ///
  /// In en, this message translates to:
  /// **'Error: {e}'**
  String errorWith(String e);

  /// No description provided for @obTitle.
  ///
  /// In en, this message translates to:
  /// **'Creating a wallet'**
  String get obTitle;

  /// No description provided for @obBioRequired.
  ///
  /// In en, this message translates to:
  /// **'The wallet requires a hardware key with biometrics (BIOMETRIC_STRONG). Registration is impossible without a configured fingerprint.'**
  String get obBioRequired;

  /// No description provided for @obBioNotSetup.
  ///
  /// In en, this message translates to:
  /// **'No fingerprint configured. Open the system settings screen and add a fingerprint.'**
  String get obBioNotSetup;

  /// No description provided for @obBioSetup.
  ///
  /// In en, this message translates to:
  /// **'Set up fingerprint'**
  String get obBioSetup;

  /// No description provided for @obPinIntro.
  ///
  /// In en, this message translates to:
  /// **'PIN protects the app. A duress PIN (optional) opens a decoy mode: the same interface, but a plausible small balance; the signing key is not used.'**
  String get obPinIntro;

  /// No description provided for @obPinNormal.
  ///
  /// In en, this message translates to:
  /// **'PIN (normal)'**
  String get obPinNormal;

  /// No description provided for @obWantDuress.
  ///
  /// In en, this message translates to:
  /// **'Set a duress PIN'**
  String get obWantDuress;

  /// No description provided for @obDuressPin.
  ///
  /// In en, this message translates to:
  /// **'Duress PIN'**
  String get obDuressPin;

  /// No description provided for @obDecoyLegend.
  ///
  /// In en, this message translates to:
  /// **'Decoy mode legend (what the coercer will see):'**
  String get obDecoyLegend;

  /// No description provided for @obDecoyBalance.
  ///
  /// In en, this message translates to:
  /// **'Plausible balance (not 0)'**
  String get obDecoyBalance;

  /// No description provided for @obDecoyHistory.
  ///
  /// In en, this message translates to:
  /// **'History (lines: date — amount — recipient)'**
  String get obDecoyHistory;

  /// No description provided for @pinMinDigits.
  ///
  /// In en, this message translates to:
  /// **'minimum 4 digits'**
  String get pinMinDigits;

  /// No description provided for @duressPinMinDigits.
  ///
  /// In en, this message translates to:
  /// **'duress PIN: minimum 4 digits'**
  String get duressPinMinDigits;

  /// No description provided for @pinsMatch.
  ///
  /// In en, this message translates to:
  /// **'PINs match'**
  String get pinsMatch;

  /// No description provided for @obGuardiansIntro.
  ///
  /// In en, this message translates to:
  /// **'Assign personal guardians (2 of 3). They will be able to approve migration to a new device. Adding a new fingerprint invalidates the key — without guardians access to the wallet is lost.'**
  String get obGuardiansIntro;

  /// No description provided for @obGuardiansEmpty.
  ///
  /// In en, this message translates to:
  /// **'There are no other members in the network yet. You may skip guardians for now, but the offer is mandatory: the list cannot be changed later (only at creation).'**
  String get obGuardiansEmpty;

  /// No description provided for @guardianRowStatus.
  ///
  /// In en, this message translates to:
  /// **'{status}, age {age} days'**
  String guardianRowStatus(String status, int age);

  /// No description provided for @obSelectedCount.
  ///
  /// In en, this message translates to:
  /// **'Selected: {count}'**
  String obSelectedCount(int count);

  /// No description provided for @regStatusReady.
  ///
  /// In en, this message translates to:
  /// **'Ready to register'**
  String get regStatusReady;

  /// No description provided for @regStatusPow.
  ///
  /// In en, this message translates to:
  /// **'Attestation and PoW (may take a while)…'**
  String get regStatusPow;

  /// No description provided for @regConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm registration'**
  String get regConfirmTitle;

  /// No description provided for @regStatusDone.
  ///
  /// In en, this message translates to:
  /// **'Wallet registered'**
  String get regStatusDone;

  /// No description provided for @pendingCount.
  ///
  /// In en, this message translates to:
  /// **'Delayed transfers: {count}'**
  String pendingCount(int count);

  /// No description provided for @pendingHint.
  ///
  /// In en, this message translates to:
  /// **'Large transfers execute after delay D; they can be cancelled within D'**
  String get pendingHint;

  /// No description provided for @historyTitle.
  ///
  /// In en, this message translates to:
  /// **'History'**
  String get historyTitle;

  /// No description provided for @historyEmpty.
  ///
  /// In en, this message translates to:
  /// **'Nothing yet'**
  String get historyEmpty;

  /// No description provided for @effBalance.
  ///
  /// In en, this message translates to:
  /// **'effective balance'**
  String get effBalance;

  /// No description provided for @effBalanceDem.
  ///
  /// In en, this message translates to:
  /// **'effective balance (demurrage applied)'**
  String get effBalanceDem;

  /// No description provided for @lockedReserve.
  ///
  /// In en, this message translates to:
  /// **'reserved for delayed transfers: {amount}'**
  String lockedReserve(int amount);

  /// No description provided for @flaggedNotice.
  ///
  /// In en, this message translates to:
  /// **'Wallet is flagged — the mark is public, no verdicts'**
  String get flaggedNotice;

  /// No description provided for @registeredNeedK.
  ///
  /// In en, this message translates to:
  /// **'Status registered: activation requires K vouches (the \"Vouching\" tab)'**
  String get registeredNeedK;

  /// No description provided for @btAwaitingTitle.
  ///
  /// In en, this message translates to:
  /// **'Received via Bluetooth — awaiting publication'**
  String get btAwaitingTitle;

  /// No description provided for @btAwaitingRow.
  ///
  /// In en, this message translates to:
  /// **'{amount} from {hash}'**
  String btAwaitingRow(int amount, String hash);

  /// No description provided for @btStatus.
  ///
  /// In en, this message translates to:
  /// **'status: {status}'**
  String btStatus(String status);

  /// No description provided for @txRegister.
  ///
  /// In en, this message translates to:
  /// **'Wallet registration'**
  String get txRegister;

  /// No description provided for @txVouch.
  ///
  /// In en, this message translates to:
  /// **'Vouching'**
  String get txVouch;

  /// No description provided for @txMint.
  ///
  /// In en, this message translates to:
  /// **'Mint (period {period})'**
  String txMint(int period);

  /// No description provided for @txTransfer.
  ///
  /// In en, this message translates to:
  /// **'Transfer {amount} → {hash}'**
  String txTransfer(int amount, String hash);

  /// No description provided for @mintTitle.
  ///
  /// In en, this message translates to:
  /// **'Mint'**
  String get mintTitle;

  /// No description provided for @mintPeriod.
  ///
  /// In en, this message translates to:
  /// **'Period: {period}'**
  String mintPeriod(int period);

  /// No description provided for @mintWindowOpen.
  ///
  /// In en, this message translates to:
  /// **'Challenge window is open (±{hours} h around the slot). You can mint.'**
  String mintWindowOpen(int hours);

  /// No description provided for @mintWindowOff.
  ///
  /// In en, this message translates to:
  /// **'Challenge window check is disabled (sandbox). You can mint.'**
  String get mintWindowOff;

  /// No description provided for @mintWindowClosed.
  ///
  /// In en, this message translates to:
  /// **'Challenge window is closed. The slot inside the month is unpredictable in advance; a miss is carried over within the accrual window ({periods} periods), funds are not blocked.'**
  String mintWindowClosed(int periods);

  /// No description provided for @mintLastClaimed.
  ///
  /// In en, this message translates to:
  /// **'Last claimed period: {period}'**
  String mintLastClaimed(int period);

  /// No description provided for @mintClaimPeriod.
  ///
  /// In en, this message translates to:
  /// **'Period to claim: {period} (completed periods, M4)'**
  String mintClaimPeriod(String period);

  /// No description provided for @mintClaimAmount.
  ///
  /// In en, this message translates to:
  /// **'Amount to mint: {amount}'**
  String mintClaimAmount(int amount);

  /// No description provided for @mintIssuance.
  ///
  /// In en, this message translates to:
  /// **'Issuance per period: {amount}'**
  String mintIssuance(int amount);

  /// No description provided for @mintConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm mint'**
  String get mintConfirmTitle;

  /// No description provided for @mintPowStatus.
  ///
  /// In en, this message translates to:
  /// **'Heavy PoW (Argon2id)…'**
  String get mintPowStatus;

  /// No description provided for @mintNoPeriods.
  ///
  /// In en, this message translates to:
  /// **'No periods to claim'**
  String get mintNoPeriods;

  /// No description provided for @mintPublished.
  ///
  /// In en, this message translates to:
  /// **'Mint published'**
  String get mintPublished;

  /// No description provided for @transferTitle.
  ///
  /// In en, this message translates to:
  /// **'Transfer'**
  String get transferTitle;

  /// No description provided for @segNetwork.
  ///
  /// In en, this message translates to:
  /// **'Via network'**
  String get segNetwork;

  /// No description provided for @segBluetooth.
  ///
  /// In en, this message translates to:
  /// **'Bluetooth'**
  String get segBluetooth;

  /// No description provided for @recipientLabel.
  ///
  /// In en, this message translates to:
  /// **'Recipient address (hex)'**
  String get recipientLabel;

  /// No description provided for @recipientHint.
  ///
  /// In en, this message translates to:
  /// **'64 hex characters'**
  String get recipientHint;

  /// No description provided for @amountLabel.
  ///
  /// In en, this message translates to:
  /// **'Amount (the recipient gets exactly this)'**
  String get amountLabel;

  /// No description provided for @feeHelper.
  ///
  /// In en, this message translates to:
  /// **'Fee: {fee} (on top of the amount, burned) — paid by the sender'**
  String feeHelper(int fee);

  /// No description provided for @delayedWarn.
  ///
  /// In en, this message translates to:
  /// **'The amount is above threshold P: the transfer will be delayed. Execution in {hours} h; it can be cancelled during the delay (the \"Delayed\" screen).'**
  String delayedWarn(int hours);

  /// No description provided for @btModeHint.
  ///
  /// In en, this message translates to:
  /// **'Bluetooth offline: the recipient will see \"received via Bluetooth — awaiting publication\". Until the network confirms, it is not a payment (12.4).'**
  String get btModeHint;

  /// No description provided for @moratoriumNote.
  ///
  /// In en, this message translates to:
  /// **'Moratorium: transfers are available {days} days after registration.'**
  String moratoriumNote(int days);

  /// No description provided for @transferConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm transfer'**
  String get transferConfirmTitle;

  /// No description provided for @btConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm Bluetooth payment'**
  String get btConfirmTitle;

  /// No description provided for @transferSent.
  ///
  /// In en, this message translates to:
  /// **'Transfer sent'**
  String get transferSent;

  /// No description provided for @transferDelayed.
  ///
  /// In en, this message translates to:
  /// **'Transfer delayed: execution in {hours} h, it can be cancelled during the delay'**
  String transferDelayed(int hours);

  /// No description provided for @signNotConfirmed.
  ///
  /// In en, this message translates to:
  /// **'Signature not confirmed'**
  String get signNotConfirmed;

  /// No description provided for @btNoDevices.
  ///
  /// In en, this message translates to:
  /// **'No paired devices'**
  String get btNoDevices;

  /// No description provided for @btSent.
  ///
  /// In en, this message translates to:
  /// **'Sent via Bluetooth. The recipient must wait for network confirmation — until publication it is not a payment'**
  String get btSent;

  /// No description provided for @btError.
  ///
  /// In en, this message translates to:
  /// **'Bluetooth: {error}'**
  String btError(String error);

  /// No description provided for @pendingTitle.
  ///
  /// In en, this message translates to:
  /// **'Delayed transfers'**
  String get pendingTitle;

  /// No description provided for @thresholdP.
  ///
  /// In en, this message translates to:
  /// **'Threshold P: {value}'**
  String thresholdP(int value);

  /// No description provided for @thresholdPendingRaise.
  ///
  /// In en, this message translates to:
  /// **'Increase to {value} takes effect after {time}'**
  String thresholdPendingRaise(int value, String time);

  /// No description provided for @newThresholdLabel.
  ///
  /// In en, this message translates to:
  /// **'New threshold P'**
  String get newThresholdLabel;

  /// No description provided for @newThresholdHelper.
  ///
  /// In en, this message translates to:
  /// **'limit P_max = {max}; increase — after 48 h, decrease — immediately'**
  String newThresholdHelper(int max);

  /// No description provided for @pendingEmpty.
  ///
  /// In en, this message translates to:
  /// **'No active delayed transfers'**
  String get pendingEmpty;

  /// No description provided for @pendingRow.
  ///
  /// In en, this message translates to:
  /// **'{amount} → {hash}'**
  String pendingRow(int amount, String hash);

  /// No description provided for @pendingRowSub.
  ///
  /// In en, this message translates to:
  /// **'execution in {time} (fee {fee})'**
  String pendingRowSub(String time, int fee);

  /// No description provided for @cancelConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm transfer cancellation'**
  String get cancelConfirmTitle;

  /// No description provided for @canceledMsg.
  ///
  /// In en, this message translates to:
  /// **'Cancelled'**
  String get canceledMsg;

  /// No description provided for @invalidThreshold.
  ///
  /// In en, this message translates to:
  /// **'Invalid threshold'**
  String get invalidThreshold;

  /// No description provided for @thresholdRaised.
  ///
  /// In en, this message translates to:
  /// **'The increase takes effect in 48 h (coercion protection)'**
  String get thresholdRaised;

  /// No description provided for @thresholdLowered.
  ///
  /// In en, this message translates to:
  /// **'Threshold lowered immediately'**
  String get thresholdLowered;

  /// No description provided for @timeHm.
  ///
  /// In en, this message translates to:
  /// **'{h} h {m} min'**
  String timeHm(int h, int m);

  /// No description provided for @vouchingTitle.
  ///
  /// In en, this message translates to:
  /// **'Vouching'**
  String get vouchingTitle;

  /// No description provided for @segMyStatus.
  ///
  /// In en, this message translates to:
  /// **'My status'**
  String get segMyStatus;

  /// No description provided for @segVouch.
  ///
  /// In en, this message translates to:
  /// **'Vouch'**
  String get segVouch;

  /// No description provided for @segMySlots.
  ///
  /// In en, this message translates to:
  /// **'My slots'**
  String get segMySlots;

  /// No description provided for @walletActive.
  ///
  /// In en, this message translates to:
  /// **'Wallet is active'**
  String get walletActive;

  /// No description provided for @vouchProgress.
  ///
  /// In en, this message translates to:
  /// **'Vouches: {received} of {k}'**
  String vouchProgress(int received, int k);

  /// No description provided for @vouchFindHelpers.
  ///
  /// In en, this message translates to:
  /// **'Find K vouchers in person: show them the QR code of your address on the \"Vouch\" tab (the \"My QR\" button).'**
  String get vouchFindHelpers;

  /// No description provided for @vouchFlagged.
  ///
  /// In en, this message translates to:
  /// **'Flagged mark: nobody can vouch for you (vouching from a flagged wallet is forbidden, decision No. 14)'**
  String get vouchFlagged;

  /// No description provided for @vouchConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm vouching'**
  String get vouchConfirmTitle;

  /// No description provided for @vouchPublished.
  ///
  /// In en, this message translates to:
  /// **'Vouch published'**
  String get vouchPublished;

  /// No description provided for @btWaiting.
  ///
  /// In en, this message translates to:
  /// **'Waiting for address via Bluetooth (60 s)…'**
  String get btWaiting;

  /// No description provided for @btAddrReceived.
  ///
  /// In en, this message translates to:
  /// **'Address received via Bluetooth'**
  String get btAddrReceived;

  /// No description provided for @vouchWarning.
  ///
  /// In en, this message translates to:
  /// **'In-person vouching (QR or Bluetooth) does NOT prove the person\'s presence. You are responsible for the choice: the slot is limited, and if evidence is presented against the vouched person you lose the reward (not money).'**
  String get vouchWarning;

  /// No description provided for @vouchOnlyActive.
  ///
  /// In en, this message translates to:
  /// **'Only an active wallet aged ≥ 3 months with ≥ 2 completed mint periods can vouch (7.4).'**
  String get vouchOnlyActive;

  /// No description provided for @candidateLabel.
  ///
  /// In en, this message translates to:
  /// **'Candidate address (hex)'**
  String get candidateLabel;

  /// No description provided for @candidateHint.
  ///
  /// In en, this message translates to:
  /// **'64 hex characters — from QR or Bluetooth'**
  String get candidateHint;

  /// No description provided for @myQrHint.
  ///
  /// In en, this message translates to:
  /// **'My address — show it to the voucher'**
  String get myQrHint;

  /// No description provided for @slotsSummary.
  ///
  /// In en, this message translates to:
  /// **'Slots: {occupied} occupied of {capacity}'**
  String slotsSummary(int occupied, int capacity);

  /// No description provided for @slotsFreeStats.
  ///
  /// In en, this message translates to:
  /// **'Free: {free}. Voucher statistics: vouched for {total}, active {active} (public reputation, 7.4)'**
  String slotsFreeStats(int free, int total, int active);

  /// No description provided for @slotsEmpty.
  ///
  /// In en, this message translates to:
  /// **'Slots are free'**
  String get slotsEmpty;

  /// No description provided for @slotActiveSub.
  ///
  /// In en, this message translates to:
  /// **'candidate is active: the slot frees after {periods} mint periods without evidence'**
  String slotActiveSub(int periods);

  /// No description provided for @slotInactiveSub.
  ///
  /// In en, this message translates to:
  /// **'candidate is not active: the slot frees in {days} days'**
  String slotInactiveSub(int days);

  /// No description provided for @registryTitle.
  ///
  /// In en, this message translates to:
  /// **'Network registry'**
  String get registryTitle;

  /// No description provided for @registryRowSub.
  ///
  /// In en, this message translates to:
  /// **'{status} • age {age} days'**
  String registryRowSub(String status, int age);

  /// No description provided for @chipAge.
  ///
  /// In en, this message translates to:
  /// **'age {age} days'**
  String chipAge(int age);

  /// No description provided for @chipSeed.
  ///
  /// In en, this message translates to:
  /// **'seed circle'**
  String get chipSeed;

  /// No description provided for @chipFlagged.
  ///
  /// In en, this message translates to:
  /// **'flagged'**
  String get chipFlagged;

  /// No description provided for @chipPublicGuardian.
  ///
  /// In en, this message translates to:
  /// **'public guardian'**
  String get chipPublicGuardian;

  /// No description provided for @vouchDepthLabel.
  ///
  /// In en, this message translates to:
  /// **'Vouching graph depth: {depth}'**
  String vouchDepthLabel(int depth);

  /// No description provided for @vouchDepthUnlinked.
  ///
  /// In en, this message translates to:
  /// **'Vouching graph depth: not linked to the seed circle'**
  String get vouchDepthUnlinked;

  /// No description provided for @rewardLine.
  ///
  /// In en, this message translates to:
  /// **'Reward to vouchers: {periods} completed mint periods'**
  String rewardLine(int periods);

  /// No description provided for @asVoucher.
  ///
  /// In en, this message translates to:
  /// **'As voucher: for {count} (free slots: {free})'**
  String asVoucher(int count, int free);

  /// No description provided for @publicHistory.
  ///
  /// In en, this message translates to:
  /// **'Public history'**
  String get publicHistory;

  /// No description provided for @migrationTitle.
  ///
  /// In en, this message translates to:
  /// **'Migration'**
  String get migrationTitle;

  /// No description provided for @migrationIntro.
  ///
  /// In en, this message translates to:
  /// **'Wallet recovery: guardians are assigned in advance (at creation). Personal — 2 of 3; with public ones — 3 of 5 (thresholds combine).'**
  String get migrationIntro;

  /// No description provided for @migrationTo.
  ///
  /// In en, this message translates to:
  /// **'Migration to {hash}'**
  String migrationTo(String hash);

  /// No description provided for @migrationRowSub.
  ///
  /// In en, this message translates to:
  /// **'execution in {days} days (delay {delay} days); guardian signatures: {sigs}'**
  String migrationRowSub(int days, int delay, int sigs);

  /// No description provided for @publicGuardiansTitle.
  ///
  /// In en, this message translates to:
  /// **'Public guardians of the network (guardian_offer, 11)'**
  String get publicGuardiansTitle;

  /// No description provided for @noGuardians.
  ///
  /// In en, this message translates to:
  /// **'Nobody has declared themselves a public guardian yet'**
  String get noGuardians;

  /// No description provided for @guardianOfferRow.
  ///
  /// In en, this message translates to:
  /// **'age {age} days, wards: {wards}'**
  String guardianOfferRow(int age, int wards);

  /// No description provided for @restartIntro.
  ///
  /// In en, this message translates to:
  /// **'If there are no guardians: a restart is a regular registration with a new key and new K vouches. The old wallet remains in the registry untouched and inaccessible.'**
  String get restartIntro;

  /// No description provided for @cancelMigrationTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm migration cancellation (old key)'**
  String get cancelMigrationTitle;

  /// No description provided for @migrationCanceled.
  ///
  /// In en, this message translates to:
  /// **'Migration cancelled'**
  String get migrationCanceled;

  /// No description provided for @restartDialogTitle.
  ///
  /// In en, this message translates to:
  /// **'Loss of funds'**
  String get restartDialogTitle;

  /// No description provided for @restartDialogBody.
  ///
  /// In en, this message translates to:
  /// **'A restart gives nothing beyond registering a new wallet: funds at the old address are lost, history is not transferred. Migration with guardians transfers age, mint history and vouches.'**
  String get restartDialogBody;

  /// No description provided for @restartFlowNote.
  ///
  /// In en, this message translates to:
  /// **'Restart: delete the app data and go through wallet creation again (new key, new K vouches).'**
  String get restartFlowNote;

  /// No description provided for @startMigrationTitle.
  ///
  /// In en, this message translates to:
  /// **'Starting migration'**
  String get startMigrationTitle;

  /// No description provided for @startMigrationIntro.
  ///
  /// In en, this message translates to:
  /// **'You are on the NEW device. Enter the old wallet address and guardian signatures (each guardian signs the statement in their own app — the \"Approve someone\'s migration\" tab).'**
  String get startMigrationIntro;

  /// No description provided for @oldAddressLabel.
  ///
  /// In en, this message translates to:
  /// **'Old address (hex)'**
  String get oldAddressLabel;

  /// No description provided for @newAddressLabel.
  ///
  /// In en, this message translates to:
  /// **'New address (hex)'**
  String get newAddressLabel;

  /// No description provided for @sigsCollected.
  ///
  /// In en, this message translates to:
  /// **'Signatures collected: {count} (personal threshold {personal} of {personalCount}; with public — {public} of {publicCount})'**
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  );

  /// No description provided for @guardianAddrLabel.
  ///
  /// In en, this message translates to:
  /// **'Guardian address (hex)'**
  String get guardianAddrLabel;

  /// No description provided for @guardianSigLabel.
  ///
  /// In en, this message translates to:
  /// **'Guardian signature (hex)'**
  String get guardianSigLabel;

  /// No description provided for @migrationConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm migration'**
  String get migrationConfirmTitle;

  /// No description provided for @migrationStarted.
  ///
  /// In en, this message translates to:
  /// **'Migration started: execution in {days} days; the old key can cancel during the delay'**
  String migrationStarted(int days);

  /// No description provided for @approveTitle.
  ///
  /// In en, this message translates to:
  /// **'Migration approval (guardian)'**
  String get approveTitle;

  /// No description provided for @approveIntro.
  ///
  /// In en, this message translates to:
  /// **'Verify the ward\'s identity through your own channels. The signature binds the pair of addresses and the nonce; hand it to the ward in person.'**
  String get approveIntro;

  /// No description provided for @nonceLabel.
  ///
  /// In en, this message translates to:
  /// **'Nonce (from the ward)'**
  String get nonceLabel;

  /// No description provided for @wardSignTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm the ward\'s migration (you are the guardian)'**
  String get wardSignTitle;

  /// No description provided for @sigToCopy.
  ///
  /// In en, this message translates to:
  /// **'Signature (hand it to the ward):'**
  String get sigToCopy;

  /// No description provided for @settingsTitle.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTitle;

  /// No description provided for @networkLine.
  ///
  /// In en, this message translates to:
  /// **'Network: {name}'**
  String networkLine(String name);

  /// No description provided for @paramUnit.
  ///
  /// In en, this message translates to:
  /// **'Unit'**
  String get paramUnit;

  /// No description provided for @paramTransfer.
  ///
  /// In en, this message translates to:
  /// **'Transfer'**
  String get paramTransfer;

  /// No description provided for @paramIssuance.
  ///
  /// In en, this message translates to:
  /// **'Issuance'**
  String get paramIssuance;

  /// No description provided for @paramIssuanceValue.
  ///
  /// In en, this message translates to:
  /// **'{amount} per {months} months'**
  String paramIssuanceValue(String amount, String months);

  /// No description provided for @paramVouchK.
  ///
  /// In en, this message translates to:
  /// **'K (vouching)'**
  String get paramVouchK;

  /// No description provided for @paramThresholdP.
  ///
  /// In en, this message translates to:
  /// **'Threshold P'**
  String get paramThresholdP;

  /// No description provided for @paramThresholdPValue.
  ///
  /// In en, this message translates to:
  /// **'{p} (P_max {max})'**
  String paramThresholdPValue(String p, String max);

  /// No description provided for @paramDelayD.
  ///
  /// In en, this message translates to:
  /// **'Delay D'**
  String get paramDelayD;

  /// No description provided for @paramDelayDValue.
  ///
  /// In en, this message translates to:
  /// **'{hours} h'**
  String paramDelayDValue(String hours);

  /// No description provided for @paramMoratorium.
  ///
  /// In en, this message translates to:
  /// **'Moratorium'**
  String get paramMoratorium;

  /// No description provided for @paramMoratoriumValue.
  ///
  /// In en, this message translates to:
  /// **'{days} days'**
  String paramMoratoriumValue(String days);

  /// No description provided for @paramDemurrage.
  ///
  /// In en, this message translates to:
  /// **'Demurrage'**
  String get paramDemurrage;

  /// No description provided for @demurrageOn.
  ///
  /// In en, this message translates to:
  /// **'enabled'**
  String get demurrageOn;

  /// No description provided for @demurrageOff.
  ///
  /// In en, this message translates to:
  /// **'disabled'**
  String get demurrageOff;

  /// No description provided for @paramAccrual.
  ///
  /// In en, this message translates to:
  /// **'Accrual window'**
  String get paramAccrual;

  /// No description provided for @paramAccrualValue.
  ///
  /// In en, this message translates to:
  /// **'{periods} periods'**
  String paramAccrualValue(String periods);

  /// No description provided for @paramGuardians.
  ///
  /// In en, this message translates to:
  /// **'Guardians'**
  String get paramGuardians;

  /// No description provided for @paramGuardiansValue.
  ///
  /// In en, this message translates to:
  /// **'personal {pt} of {pc}; public {pubt} of {pubc}'**
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc);

  /// No description provided for @paramMigration.
  ///
  /// In en, this message translates to:
  /// **'Migration'**
  String get paramMigration;

  /// No description provided for @paramMigrationValue.
  ///
  /// In en, this message translates to:
  /// **'delay {days} days'**
  String paramMigrationValue(String days);

  /// No description provided for @paramEpoch.
  ///
  /// In en, this message translates to:
  /// **'Checkpoint epoch'**
  String get paramEpoch;

  /// No description provided for @paramEpochValue.
  ///
  /// In en, this message translates to:
  /// **'{seconds} s'**
  String paramEpochValue(String seconds);

  /// No description provided for @sandboxNote.
  ///
  /// In en, this message translates to:
  /// **'Sandbox (synthetic attestation)'**
  String get sandboxNote;

  /// No description provided for @guardianOfferTitle.
  ///
  /// In en, this message translates to:
  /// **'Declare yourself a public guardian'**
  String get guardianOfferTitle;

  /// No description provided for @guardianAlready.
  ///
  /// In en, this message translates to:
  /// **'You are already a public guardian (wards: {wards})'**
  String guardianAlready(int wards);

  /// No description provided for @guardianReq.
  ///
  /// In en, this message translates to:
  /// **'Requirements: active, age ≥ {days} days'**
  String guardianReq(int days);

  /// No description provided for @guardianConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Confirm becoming a public guardian'**
  String get guardianConfirmTitle;

  /// No description provided for @guardianPublished.
  ///
  /// In en, this message translates to:
  /// **'Declaration published'**
  String get guardianPublished;

  /// No description provided for @nodeRunning.
  ///
  /// In en, this message translates to:
  /// **'Node is running'**
  String get nodeRunning;

  /// No description provided for @nodeStopped.
  ///
  /// In en, this message translates to:
  /// **'Node is stopped'**
  String get nodeStopped;

  /// No description provided for @nodeLevelHint.
  ///
  /// In en, this message translates to:
  /// **'Full level — when charging and on Wi-Fi, otherwise light (12.2)'**
  String get nodeLevelHint;

  /// No description provided for @lockApp.
  ///
  /// In en, this message translates to:
  /// **'Lock the app'**
  String get lockApp;

  /// No description provided for @relockNote.
  ///
  /// In en, this message translates to:
  /// **'Locked. Restart the app.'**
  String get relockNote;

  /// No description provided for @languageTitle.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get languageTitle;

  /// No description provided for @languageSystem.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get languageSystem;

  /// No description provided for @languageRu.
  ///
  /// In en, this message translates to:
  /// **'Русский'**
  String get languageRu;

  /// No description provided for @languageEn.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get languageEn;

  /// No description provided for @languageEs.
  ///
  /// In en, this message translates to:
  /// **'Español'**
  String get languageEs;

  /// No description provided for @languagePt.
  ///
  /// In en, this message translates to:
  /// **'Português'**
  String get languagePt;

  /// No description provided for @languageZh.
  ///
  /// In en, this message translates to:
  /// **'中文'**
  String get languageZh;

  /// No description provided for @recoveryTitle.
  ///
  /// In en, this message translates to:
  /// **'Recovery card'**
  String get recoveryTitle;

  /// No description provided for @recoveryIntro.
  ///
  /// In en, this message translates to:
  /// **'Save this data outside the phone: it is needed to recover the wallet if the device is lost (migration with guardians, section 11).'**
  String get recoveryIntro;

  /// No description provided for @recoveryAddress.
  ///
  /// In en, this message translates to:
  /// **'Wallet address'**
  String get recoveryAddress;

  /// No description provided for @recoveryNetwork.
  ///
  /// In en, this message translates to:
  /// **'Network'**
  String get recoveryNetwork;

  /// No description provided for @recoveryGuardians.
  ///
  /// In en, this message translates to:
  /// **'Your guardians'**
  String get recoveryGuardians;

  /// No description provided for @recoveryNoGuardians.
  ///
  /// In en, this message translates to:
  /// **'No guardians assigned: recovery is impossible, only a restart is available (regular registration with new vouches).'**
  String get recoveryNoGuardians;

  /// No description provided for @recoveryHowToSave.
  ///
  /// In en, this message translates to:
  /// **'How to save: write it on paper or put it into a password manager — outside the phone. The QR code can be printed from this screen.'**
  String get recoveryHowToSave;

  /// No description provided for @recoveryNeverShare.
  ///
  /// In en, this message translates to:
  /// **'Never share migration data: the nonce and guardian signatures. Only the address and this card\'s QR may be shown to others.'**
  String get recoveryNeverShare;

  /// No description provided for @recoverySavedDone.
  ///
  /// In en, this message translates to:
  /// **'Saved — continue'**
  String get recoverySavedDone;

  /// No description provided for @settingsRecovery.
  ///
  /// In en, this message translates to:
  /// **'Recovery card'**
  String get settingsRecovery;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'es', 'pt', 'ru', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'es':
      return AppLocalizationsEs();
    case 'pt':
      return AppLocalizationsPt();
    case 'ru':
      return AppLocalizationsRu();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
