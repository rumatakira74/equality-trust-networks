// Мок платформенного слоя для widget/unit-тестов (этап 7, п.8:
// платформенный слой мокается). Программируемые ответы + журнал
// вызовов для проверок поведения экранов.
import 'dart:async';

import 'package:tnp_app/services/models.dart';
import 'package:tnp_app/services/platform_service.dart';

class FakePlatformService implements PlatformService {
  // программируемые ответы
  ManifestParams manifestParams = ManifestParams.fromJson({
    'network_id': '4e465553',
    'display_name': 'TNP Test Genesis',
    'unit': 'currency',
    'transfer_mode': 'free',
    'issuance_amount': 1000,
    'issuance_period_months': 1,
    'accrual_window_periods': 6,
    'fee_mode': 'burn',
    'fee_min': 1,
    'rate_min': 10000,
    'demurrage_rate_micros': 10000,
    'demurrage_floor': 3000,
    'vouch_k': 3,
    'voucher_min_age_days': 90,
    'slot_free_mint_periods': 3,
    'slot_free_days': 90,
    'transfer_moratorium_days': 7,
    'transfer_delay_hours': 48,
    'delayed_threshold_default': 3000,
    'delayed_threshold_max': 30000,
    'guardian_personal_threshold': 2,
    'guardian_personal_count': 3,
    'guardian_public_threshold': 3,
    'guardian_public_count': 5,
    'guardian_min_age_days': 365,
    'migrate_delay_days': 14,
    'mint_challenge_window_hours': 48,
    'epoch_seconds': 3600,
    'sandbox': false,
  });
  bool biometrics = true;
  bool pinSet = false;
  bool registered = false;
  bool duress = false;
  String pinResult = 'normal'; // normal | none
  // хранимые PIN (как PinManager: обычный обязателен, тревожный опционален;
  // по файлу нельзя узнать, задан ли тревожный — тут хранится для проверок)
  String? _normalPin;
  String? _duressPin;
  bool biometricLoginResult = true;
  WalletInfo wallet = WalletInfo.fromJson({
    'exists': true,
    'address': '11' * 32,
    'status': 'REGISTERED',
    'balance': 1500,
    'liquid': 1500,
    'threshold_p': 3000,
    'vouches_received': 1,
    'seq': 1,
  });
  Map<String, int> vouchProgressData = {'received': 1, 'k': 3};
  SlotsInfo slotsData = SlotsInfo.fromJson({
    'exists': true,
    'capacity': 3,
    'occupied': [
      {'candidate': '22' * 32, 'since': 100, 'candidate_active': true},
      {'candidate': '33' * 32, 'since': 200, 'candidate_active': false},
    ],
  });
  List<HistoryEntry> historyEntries = [
    HistoryEntry.fromJson({'seq': 1, 'type': 'REGISTER', 'timestamp': 100}),
    HistoryEntry.fromJson({
      'seq': 2, 'type': 'MINT', 'amount': 2000, 'period': 1, 'timestamp': 200,
    }),
  ];
  List<PendingTransfer> pendingData = [
    PendingTransfer.fromJson({
      'seq': 3,
      'recipient': '44' * 32,
      'amount': 5000,
      'fee': 50,
      'remaining_sec': 100000,
    }),
  ];
  List<PendingMigration> migrations = [];
  List<PendingGuardianUpdate> guardianUpdates = [];
  String changeHash = 'ab' * 32;
  String guardianChangeSig = 'cd' * 32;
  MintWindow mintWin = MintWindow.fromJson({
    'open': true,
    'window_hours': 48,
    'accrual_window_periods': 6,
    'issuance_amount': 1000,
    'current_period': 2,
    'last_claimed_period': 1,
  });
  List<RegistryRow> registryRows = [
    RegistryRow.fromJson({
      'address': '55' * 32, 'status': 'ACTIVE', 'age_days': 400, 'flagged': false,
    }),
  ];
  List<GuardianOffer> offersData = [];
  List<AwaitingTx> awaitingData = [
    AwaitingTx.fromJson({
      'txid': '66' * 32, 'type': 'TRANSFER', 'recipient': '77' * 32,
      'amount': 300, 'status': 'awaiting_publication',
    }),
  ];
  int feeValue = 10;
  bool signOk = true;
  SubmitResult submitResult = SubmitResult('accepted', 'OK');

  // журнал вызовов
  final List<String> calls = [];
  final List<String> builtTxs = [];
  String? setupDuressPin;

  @override
  Future<ManifestParams> manifest() async => manifestParams;

  @override
  Future<bool> biometricsAvailable() async => biometrics;

  @override
  Future<void> openBiometricsEnroll() async => calls.add('enroll');

  @override
  Future<bool> pinIsSet() async => pinSet;

  @override
  Future<void> setupPin(String normal, String? duress, int decoyBalance,
      List<String> decoyHistory) async {
    calls.add('setupPin');
    setupDuressPin = duress;
    // модель PinManager: оба хеша одним записывающим вызовом; проверка
    // одного и того же кода для обоих PIN (неразличимость)
    _normalPin = normal;
    _duressPin = (duress == null || duress.isEmpty) ? null : duress;
  }

  @override
  Future<String> verifyPin(String pin) async {
    calls.add('verifyPin');
    // без заданного PIN — только none (мост без PinManager)
    if (_normalPin == null) return pinResult;
    if (pin == _normalPin) {
      duress = false; // обычный PIN выводит из приманки
      return 'normal';
    }
    if (_duressPin != null && pin == _duressPin) {
      duress = true; // тревожный: приманка активна, ответ неотличим
      return 'normal';
    }
    return 'none';
  }

  @override
  Future<bool> duressActive() async => duress;

  @override
  Future<void> exitDuress() async => duress = false;

  @override
  Future<bool> biometricLoginAllowed() async => !duress;

  @override
  Future<bool> biometricLogin() async {
    calls.add('biometricLogin');
    if (duress) return false; // отпечаток при активной приманке отключён
    return biometricLoginResult;
  }

  @override
  Future<String> walletAddress() async => wallet.address;

  @override
  Future<bool> walletRegistered() async => registered;

  @override
  Future<String> buildRegister(List<String> guardians) async {
    builtTxs.add('register(${guardians.length})');
    return 'tx1';
  }

  @override
  Future<Map<String, dynamic>> buildTransfer(
      String recipientHex, int amount) async {
    builtTxs.add('transfer($amount)');
    return {'id': 'tx2', 'fee': feeValue, 'delayed': amount > wallet.thresholdP};
  }

  @override
  Future<String> buildMint(int period, int amount) async {
    builtTxs.add('mint($period,$amount)');
    return 'tx3';
  }

  @override
  Future<String> buildVouch(String candidateHex) async {
    builtTxs.add('vouch');
    return 'tx4';
  }

  @override
  Future<String> buildCancel(int pendingSeq) async {
    builtTxs.add('cancel($pendingSeq)');
    return 'tx5';
  }

  @override
  Future<String> buildSetThreshold(int newP) async {
    builtTxs.add('threshold($newP)');
    return 'tx6';
  }

  @override
  Future<String> buildGuardianOffer() async {
    builtTxs.add('offer');
    return 'tx7';
  }

  @override
  Future<String> buildMigratePropose(
      String oldHex, List<Map<String, String>> approvals, int nonce) async {
    builtTxs.add('migrate(${approvals.length})');
    return 'tx8';
  }

  @override
  Future<String> buildMigrateCancel(String newHex, int proposeSeq) async {
    builtTxs.add('migrateCancel');
    return 'tx9';
  }

  @override
  Future<String> migrateStatementHash(
          String oldHex, String newHex, int nonce) async =>
      'aa' * 32;

  @override
  Future<Map<String, dynamic>> signMigrateStatement(
      String oldHex, String newHex, int nonce, String title) async {
    calls.add('signStatement');
    return {'sig': 'bb' * 32};
  }

  @override
  Future<String> buildUpdateGuardiansPropose(
      List<String> guardians, List<Map<String, String>> approvals,
      int nonce) async {
    builtTxs.add('updateGuardians(${guardians.length},${approvals.length})');
    return 'tx10';
  }

  @override
  Future<String> buildUpdateGuardiansCancel(int proposeSeq) async {
    builtTxs.add('updateGuardiansCancel($proposeSeq)');
    return 'tx11';
  }

  @override
  Future<String> guardianChangeHash(
      List<String> guardians, int nonce) async {
    calls.add('changeHash(${guardians.length})');
    return changeHash;
  }

  @override
  Future<Map<String, dynamic>> signGuardianChange(
      String hashHex, String title) async {
    if (duress) {
      // приманка: ключ не используется, ответ неотличим (8.2)
      calls.add('signGuardianChangeFake');
      return {'sig': '', 'fake': true};
    }
    calls.add('signGuardianChange');
    return {'sig': guardianChangeSig};
  }

  @override
  Future<List<PendingGuardianUpdate>> pendingGuardianUpdates(
          {String? addrHex}) async =>
      duress ? <PendingGuardianUpdate>[] : guardianUpdates;

  @override
  Future<bool> signTx(String id, String title) async {
    if (duress) {
      // приманка (8.2): подпись НЕ выполняется, ответ правдоподобен
      calls.add('signFake($id)');
      return true;
    }
    calls.add('sign($id)');
    return signOk;
  }

  @override
  Future<SubmitResult> submitTx(String id) async {
    if (duress) {
      // приманка (8.2): публикации нет, ответ как у обычного перевода
      calls.add('submitFake($id)');
      return SubmitResult('accepted', 'OK');
    }
    calls.add('submit($id)');
    return submitResult;
  }

  @override
  Future<void> discardTx(String id) async => calls.add('discard($id)');

  @override
  Future<bool> btAvailable() async => true;

  @override
  Future<bool> btPermissions() async => true;

  @override
  Future<void> requestBtPermissions() async {}

  @override
  Future<List<String>> btBondedDevices() async => ['Pixel 9'];

  @override
  Future<Map<String, dynamic>> sendBluetooth(String device, String txId) async {
    calls.add('btSend');
    return {'ok': true};
  }

  @override
  Future<Map<String, dynamic>> btSendAddress(String device) async {
    calls.add('btAddr');
    return {'ok': true};
  }

  @override
  Future<String> btReceiveAddress() async => '88' * 32;

  @override
  Future<void> nodeStart() async => calls.add('nodeStart');

  @override
  Future<void> nodeStop() async => calls.add('nodeStop');

  @override
  Future<Map<String, dynamic>> nodeStats() async => {'level': 'full'};

  @override
  Future<void> publishAwaiting() async => calls.add('publishAwaiting');

  @override
  Future<String> offlineStatus(String txidHex) async => 'awaiting_publication';

  @override
  Future<List<AwaitingTx>> awaiting() async => awaitingData;

  @override
  Future<int> balanceWithProof() async => wallet.balance;

  @override
  Future<WalletInfo> walletInfo({String? addrHex}) async => wallet;

  @override
  Future<Map<String, int>> vouchProgress({String? addrHex}) async =>
      vouchProgressData;

  @override
  Future<SlotsInfo> slots({String? addrHex}) async => slotsData;

  @override
  Future<List<HistoryEntry>> history({String? addrHex, int limit = 0}) async =>
      historyEntries;

  @override
  Future<List<PendingTransfer>> pendingTransfers({String? addrHex}) async =>
      pendingData;

  @override
  Future<List<PendingMigration>> pendingMigrations({String? addrHex}) async =>
      migrations;

  @override
  Future<MintWindow> mintWindow() async => mintWin;

  @override
  Future<int> feeQuote(int amount) async => feeValue;

  @override
  Future<List<GuardianOffer>> guardianOffers() async => offersData;

  @override
  Future<List<RegistryRow>> registry() async => registryRows;

  @override
  Future<Map<String, int>> clock() async => {'epoch': 10, 'net_time': 1000, 'period': 2};
}
