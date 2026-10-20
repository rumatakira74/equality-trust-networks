// Сервис платформенного канала (этап 7). Все вызовы идут в Kotlin
// (FlutterBridge, канал platform.tnp/app), Kotlin — в платформенный
// слой этапа 6 и ядро (AAR). Интерфейс абстрактный — в widget/unit-
// тестах платформенный слой мокается.
import 'dart:convert';

import 'package:flutter/services.dart';

import 'models.dart';

/// Ошибка платформенного слоя.
class TnpPlatformException implements Exception {
  final String code;
  final String message;
  TnpPlatformException(this.code, this.message);
  @override
  String toString() => '$code: $message';
}

/// Интерфейс платформенного слоя (мокается в тестах).
abstract class PlatformService {
  // -- манифест и устройство --
  Future<ManifestParams> manifest();
  Future<bool> biometricsAvailable();
  Future<void> openBiometricsEnroll();

  // -- PIN и приманка (8.1, 8.2) --
  Future<bool> pinIsSet();
  Future<void> setupPin(String normal, String? duress, int decoyBalance,
      List<String> decoyHistory);
  Future<String> verifyPin(String pin); // normal | none (неотличимо)
  Future<bool> duressActive();
  Future<void> exitDuress();
  Future<bool> biometricLoginAllowed();
  Future<bool> biometricLogin(); // вход по отпечатку (разблокировка)

  // -- кошелёк --
  Future<String> walletAddress();
  Future<bool> walletRegistered();

  // -- сборка транзакций (PoW на платформенном слое) --
  Future<String> buildRegister(List<String> guardians);
  Future<Map<String, dynamic>> buildTransfer(String recipientHex, int amount);
  Future<String> buildMint(int period, int amount);
  Future<String> buildVouch(String candidateHex);
  Future<String> buildCancel(int pendingSeq);
  Future<String> buildSetThreshold(int newP);
  Future<String> buildGuardianOffer();
  Future<String> buildMigratePropose(
      String oldHex, List<Map<String, String>> approvals, int nonce);
  Future<String> buildMigrateCancel(String newHex, int proposeSeq);
  Future<String> migrateStatementHash(String oldHex, String newHex, int nonce);
  Future<Map<String, dynamic>> signMigrateStatement(
      String oldHex, String newHex, int nonce, String title);

  // -- подпись (биометрия) и публикация --
  Future<bool> signTx(String id, String title);
  Future<SubmitResult> submitTx(String id);
  Future<void> discardTx(String id);

  // -- Bluetooth (7.4, 12.4) --
  Future<bool> btAvailable();
  Future<bool> btPermissions();
  Future<void> requestBtPermissions();
  Future<List<String>> btBondedDevices();
  Future<Map<String, dynamic>> sendBluetooth(String device, String txId);
  Future<Map<String, dynamic>> btSendAddress(String device);
  Future<String> btReceiveAddress();

  // -- узел (12.2) --
  Future<void> nodeStart();
  Future<void> nodeStop();
  Future<Map<String, dynamic>> nodeStats();
  Future<void> publishAwaiting();
  Future<String> offlineStatus(String txidHex);
  Future<List<AwaitingTx>> awaiting();

  // -- запросы состояния (net/query.go) --
  Future<int> balanceWithProof();
  Future<WalletInfo> walletInfo({String? addrHex});
  Future<Map<String, int>> vouchProgress({String? addrHex});
  Future<SlotsInfo> slots({String? addrHex});
  Future<List<HistoryEntry>> history({String? addrHex, int limit = 0});
  Future<List<PendingTransfer>> pendingTransfers({String? addrHex});
  Future<List<PendingMigration>> pendingMigrations({String? addrHex});
  Future<MintWindow> mintWindow();
  Future<int> feeQuote(int amount);
  Future<List<GuardianOffer>> guardianOffers();
  Future<List<RegistryRow>> registry();
  Future<Map<String, int>> clock();
}

/// Реализация через MethodChannel.
class MethodChannelPlatformService implements PlatformService {
  static const _channel = MethodChannel('platform.tnp/app');

  Future<dynamic> _call(String method, [Map<String, dynamic>? args]) async {
    try {
      return await _channel.invokeMethod(method, args ?? {});
    } on PlatformException catch (e) {
      throw TnpPlatformException(e.code, e.message ?? '');
    }
  }

  Map<String, dynamic> _json(dynamic v) {
    if (v == null) return {};
    if (v is Map) return v.cast<String, dynamic>();
    return jsonDecode(v as String) as Map<String, dynamic>;
  }

  List<Map<String, dynamic>> _jsonList(dynamic v) {
    if (v == null) return [];
    final list = v is List ? v : jsonDecode(v as String) as List;
    return list.map((e) {
      if (e is Map) return e.cast<String, dynamic>();
      return jsonDecode(e as String) as Map<String, dynamic>;
    }).toList();
  }

  @override
  Future<ManifestParams> manifest() async =>
      ManifestParams.fromJson(_json(await _call('manifest')));

  @override
  Future<bool> biometricsAvailable() async =>
      await _call('biometricsAvailable') as bool;

  @override
  Future<void> openBiometricsEnroll() => _call('openBiometricsEnroll');

  @override
  Future<bool> pinIsSet() async => await _call('pinIsSet') as bool;

  @override
  Future<void> setupPin(String normal, String? duress, int decoyBalance,
      List<String> decoyHistory) async {
    await _call('setupPin', {
      'normal': normal,
      'duress': duress ?? '',
      'decoyBalance': decoyBalance,
      'decoyHistory': decoyHistory,
    });
  }

  @override
  Future<String> verifyPin(String pin) async =>
      await _call('verifyPin', {'pin': pin}) as String;

  @override
  Future<bool> duressActive() async => await _call('duressActive') as bool;

  @override
  Future<void> exitDuress() => _call('exitDuress');

  @override
  Future<bool> biometricLoginAllowed() async =>
      await _call('biometricLoginAllowed') as bool;

  @override
  Future<bool> biometricLogin() async =>
      await _call('biometricLogin') as bool;

  @override
  Future<String> walletAddress() async =>
      (await _call('walletAddress') as String?) ?? '';

  @override
  Future<bool> walletRegistered() async =>
      await _call('walletRegistered') as bool;

  @override
  Future<String> buildRegister(List<String> guardians) async =>
      (await _call('buildRegister', {'guardians': guardians})
          as Map)['id'] as String;

  @override
  Future<Map<String, dynamic>> buildTransfer(
      String recipientHex, int amount) async {
    final r = await _call('buildTransfer', {
      'recipient': recipientHex,
      'amount': amount,
    });
    return (r as Map).cast<String, dynamic>();
  }

  @override
  Future<String> buildMint(int period, int amount) async =>
      (await _call('buildMint', {'period': period, 'amount': amount})
          as Map)['id'] as String;

  @override
  Future<String> buildVouch(String candidateHex) async =>
      (await _call('buildVouch', {'candidate': candidateHex})
          as Map)['id'] as String;

  @override
  Future<String> buildCancel(int pendingSeq) async =>
      (await _call('buildCancel', {'pendingSeq': pendingSeq})
          as Map)['id'] as String;

  @override
  Future<String> buildSetThreshold(int newP) async =>
      (await _call('buildSetThreshold', {'newP': newP})
          as Map)['id'] as String;

  @override
  Future<String> buildGuardianOffer() async =>
      (await _call('buildGuardianOffer') as Map)['id'] as String;

  @override
  Future<String> buildMigratePropose(
      String oldHex, List<Map<String, String>> approvals, int nonce) async {
    final r = await _call('buildMigratePropose', {
      'old': oldHex,
      'approvals': approvals,
      'nonce': nonce,
    });
    return (r as Map)['id'] as String;
  }

  @override
  Future<String> buildMigrateCancel(String newHex, int proposeSeq) async =>
      (await _call('buildMigrateCancel', {
        'new': newHex,
        'proposeSeq': proposeSeq,
      }) as Map)['id'] as String;

  @override
  Future<String> migrateStatementHash(
          String oldHex, String newHex, int nonce) async =>
      await _call('migrateStatementHash', {
        'old': oldHex,
        'new': newHex,
        'nonce': nonce,
      }) as String;

  @override
  Future<Map<String, dynamic>> signMigrateStatement(
      String oldHex, String newHex, int nonce, String title) async {
    final r = await _call('signMigrateStatement', {
      'old': oldHex,
      'new': newHex,
      'nonce': nonce,
      'title': title,
    });
    return (r as Map).cast<String, dynamic>();
  }

  @override
  Future<bool> signTx(String id, String title) async {
    final r = await _call('signTx', {'id': id, 'title': title});
    return (r as Map)['ok'] as bool;
  }

  @override
  Future<SubmitResult> submitTx(String id) async {
    final r = (await _call('submitTx', {'id': id}) as Map).cast<String, dynamic>();
    return SubmitResult(
      r['outcome'] as String? ?? '',
      r['code'] as String? ?? '',
      fake: r['fake'] as bool? ?? false,
    );
  }

  @override
  Future<void> discardTx(String id) => _call('discardTx', {'id': id});

  @override
  Future<bool> btAvailable() async => await _call('btAvailable') as bool;

  @override
  Future<bool> btPermissions() async => await _call('btPermissions') as bool;

  @override
  Future<void> requestBtPermissions() => _call('requestBtPermissions');

  @override
  Future<List<String>> btBondedDevices() async =>
      ((await _call('btBondedDevices') as List?) ?? [])
          .map((e) => '$e')
          .toList();

  @override
  Future<Map<String, dynamic>> sendBluetooth(String device, String txId) async {
    final r = await _call('sendBluetooth', {'device': device, 'id': txId});
    return (r as Map).cast<String, dynamic>();
  }

  @override
  Future<Map<String, dynamic>> btSendAddress(String device) async {
    final r = await _call('btSendAddress', {'device': device});
    return (r as Map).cast<String, dynamic>();
  }

  @override
  Future<String> btReceiveAddress() async =>
      (await _call('btReceiveAddress') as Map)['address'] as String;

  @override
  Future<void> nodeStart() => _call('nodeStart');

  @override
  Future<void> nodeStop() => _call('nodeStop');

  @override
  Future<Map<String, dynamic>> nodeStats() async {
    final s = await _call('nodeStats');
    if (s == null) return {};
    return jsonDecode(s as String) as Map<String, dynamic>;
  }

  @override
  Future<void> publishAwaiting() => _call('publishAwaiting');

  @override
  Future<String> offlineStatus(String txidHex) async =>
      await _call('offlineStatus', {'txid': txidHex}) as String? ?? 'unknown';

  @override
  Future<List<AwaitingTx>> awaiting() async => _jsonList(await _call('awaiting'))
      .map((e) => AwaitingTx.fromJson(e))
      .toList();

  @override
  Future<int> balanceWithProof() async =>
      (await _call('balanceWithProof') as num?)?.toInt() ?? -1;

  @override
  Future<WalletInfo> walletInfo({String? addrHex}) async =>
      WalletInfo.fromJson(_json(await _call('walletInfo', {'addr': addrHex})));

  @override
  Future<Map<String, int>> vouchProgress({String? addrHex}) async {
    final j = _json(await _call('vouchProgress', {'addr': addrHex}));
    return {
      'received': (j['received'] as num?)?.toInt() ?? 0,
      'k': (j['k'] as num?)?.toInt() ?? 0,
    };
  }

  @override
  Future<SlotsInfo> slots({String? addrHex}) async =>
      SlotsInfo.fromJson(_json(await _call('slots', {'addr': addrHex})));

  @override
  Future<List<HistoryEntry>> history({String? addrHex, int limit = 0}) async =>
      _jsonList(await _call('history', {'addr': addrHex, 'limit': limit}))
          .map((e) => HistoryEntry.fromJson(e))
          .toList();

  @override
  Future<List<PendingTransfer>> pendingTransfers({String? addrHex}) async =>
      _jsonList(await _call('pendingTransfers', {'addr': addrHex}))
          .map((e) => PendingTransfer.fromJson(e))
          .toList();

  @override
  Future<List<PendingMigration>> pendingMigrations({String? addrHex}) async =>
      _jsonList(await _call('pendingMigrations', {'addr': addrHex}))
          .map((e) => PendingMigration.fromJson(e))
          .toList();

  @override
  Future<MintWindow> mintWindow() async =>
      MintWindow.fromJson(_json(await _call('mintWindow')));

  @override
  Future<int> feeQuote(int amount) async =>
      (await _call('feeQuote', {'amount': amount}) as num?)?.toInt() ?? 0;

  @override
  Future<List<GuardianOffer>> guardianOffers() async =>
      _jsonList(await _call('guardianOffers'))
          .map((e) => GuardianOffer.fromJson(e))
          .toList();

  @override
  Future<List<RegistryRow>> registry() async =>
      _jsonList(await _call('registry'))
          .map((e) => RegistryRow.fromJson(e))
          .toList();

  @override
  Future<Map<String, int>> clock() async {
    final j = _json(await _call('clock'));
    return {
      'epoch': (j['epoch'] as num?)?.toInt() ?? 0,
      'net_time': (j['net_time'] as num?)?.toInt() ?? 0,
      'period': (j['period'] as num?)?.toInt() ?? 0,
    };
  }
}
