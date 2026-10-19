// Состояние приложения (этап 7): Provider + ChangeNotifier —
// обоснование выбора в отчёте этапа 7. Хранит манифест, свой кошелёк,
// статус узла; экраны запрашивают остальное по требованию.
import 'dart:async';

import 'package:flutter/foundation.dart';

import '../services/models.dart';
import '../services/platform_service.dart';

/// Общее состояние: манифест, кошелёк, PIN, узел.
class AppState extends ChangeNotifier {
  final PlatformService platform;
  AppState(this.platform);

  ManifestParams? _manifest;
  ManifestParams? get manifest => _manifest;

  bool _pinSet = false;
  bool get pinSet => _pinSet;

  bool _walletRegistered = false;
  bool get walletRegistered => _walletRegistered;

  String _address = '';
  String get address => _address;

  WalletInfo? _wallet;
  WalletInfo? get wallet => _wallet;

  bool _nodeRunning = false;
  bool get nodeRunning => _nodeRunning;

  bool _biometricsOk = false;
  bool get biometricsOk => _biometricsOk;

  bool _loading = true;
  bool get loading => _loading;

  /// Ошибка последней операции (для SnackBar).
  String? lastError;
  void setError(String? e) {
    lastError = e;
    notifyListeners();
  }

  /// Первичная загрузка: манифест, PIN, кошелёк, биометрия.
  Future<void> load() async {
    _loading = true;
    notifyListeners();
    try {
      _manifest = await platform.manifest();
      _pinSet = await platform.pinIsSet();
      _walletRegistered = await platform.walletRegistered();
      _address = await platform.walletAddress();
      _biometricsOk = await platform.biometricsAvailable();
      await refreshWallet();
    } catch (e) {
      lastError = '$e';
    }
    _loading = false;
    notifyListeners();
  }

  /// Обновить снапшот своего кошелька (полный узел — весь реестр).
  Future<void> refreshWallet() async {
    if (!_walletRegistered) return;
    try {
      // сначала статистика узла: walletInfo до готовности узла кидает
      // NONODE и не должен маскировать остальные обновления
      _nodeRunning = (await platform.nodeStats()).isNotEmpty;
      _wallet = await platform.walletInfo();
    } on TnpPlatformException {
      _nodeRunning = false;
    }
    notifyListeners();
  }

  /// Запуск узла (12.2) и обновление состояния.
  Future<void> startNode() async {
    await platform.nodeStart();
    // узел поднимается в foreground-сервисе асинхронно: дождаться
    // готовности, иначе снапшоты увидят пустой реестр
    for (var i = 0; i < 25; i++) {
      try {
        if ((await platform.nodeStats()).isNotEmpty) break;
      } on TnpPlatformException {
        // узел ещё не готов
      }
      await Future.delayed(const Duration(milliseconds: 200));
    }
    _nodeRunning = true;
    notifyListeners();
  }

  /// Остановка узла (12.2).
  Future<void> nodeStop() async {
    await platform.nodeStop();
    _nodeRunning = false;
    notifyListeners();
  }

  /// Онбординг завершён: PIN задан, кошелёк регистрируется.
  Future<void> onboardingDone() async {
    _pinSet = true;
    _walletRegistered = true;
    _address = await platform.walletAddress();
    notifyListeners();
  }

  /// Полный цикл: сборка -> биометрическая подпись -> публикация.
  /// Возвращает результат публикации (в приманке — неотличимый ответ).
  Future<SubmitResult> signAndSubmit(
    Future<String> Function() build,
    String title,
  ) async {
    final id = await build();
    final ok = await platform.signTx(id, title);
    if (!ok) {
      await platform.discardTx(id);
      throw TnpPlatformException('SIGN', 'подпись не подтверждена');
    }
    return platform.submitTx(id);
  }
}
