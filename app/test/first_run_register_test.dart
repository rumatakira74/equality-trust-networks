// Регрессия FINDING-02 (тема fix-lang-tails): первая регистрация на
// чистой установке не должна падать NONODE.
//
// Причина бага: узел стартовал ДО buildRegister, а buildRegister — это
// единственное место, где мост узнаёт адрес кошелька (ключ Keystore
// создаётся при сборке регистрации). Light-узел без адреса не стартует,
// поэтому submitTx отвечал NONODE; со второй попытки (адрес уже известен)
// регистрация проходила.
//
// Заглушка ниже моделирует именно это ограничение моста: nodeStart не
// поднимает узел, пока не собран register, а submitTx без узла кидает
// NONODE. На старом порядке тест падает, на исправленном — проходит.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/onboarding/onboarding_flow.dart';
import 'package:tnp_app/services/models.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';

import 'fake_platform.dart';

/// Мост песочницы с ограничением first-run: адрес кошелька появляется
/// только после buildRegister, без адреса узел не стартует.
class _FirstRunFake extends FakePlatformService {
  bool walletReady = false;
  bool nodeUp = false;

  /// Сквозной журнал значимых вызовов (порядок важен).
  final List<String> order = [];

  @override
  Future<String> buildRegister(List<String> guardians) async {
    final id = await super.buildRegister(guardians);
    walletReady = true; // ключ создан, адрес кошелька известен мосту
    order.add('buildRegister');
    return id;
  }

  @override
  Future<void> nodeStart() async {
    order.add('nodeStart');
    await super.nodeStart();
    // NodeController.ensureStarted: без адреса кошелька узел не
    // поднимается (сервис при этом уже запущен)
    nodeUp = walletReady;
  }

  @override
  Future<Map<String, dynamic>> nodeStats() async {
    if (!nodeUp) return <String, dynamic>{};
    return super.nodeStats();
  }

  @override
  Future<SubmitResult> submitTx(String id) async {
    if (!nodeUp) {
      throw TnpPlatformException('NONODE', 'узел не запущен');
    }
    order.add('submit');
    return super.submitTx(id);
  }
}

Widget _wrap(Widget child, FakePlatformService fake) {
  return MultiProvider(
    providers: [
      Provider<PlatformService>.value(value: fake),
      ChangeNotifierProvider(create: (_) => AppState(fake)..load()),
    ],
    child: MaterialApp(
      locale: const Locale('ru'),
      supportedLocales: AppLocalizations.supportedLocales,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: child,
    ),
  );
}

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets(
      'чистая установка: регистрация проходит без NONODE, узел поднят '
      'после buildRegister', (tester) async {
    final fake = _FirstRunFake()..biometrics = true;
    await tester.pumpWidget(_wrap(const OnboardingFlow(), fake));
    await tester.pumpAndSettle();

    // шаг биометрии
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();

    // шаг PIN
    await tester.enterText(
      find.widgetWithText(TextField, 'PIN (обычный)'),
      '1234',
    );
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();

    // шаг гардианов: пропуск разрешён (предложение обязательно)
    await tester.tap(find.text('Пропустить'));
    await tester.pumpAndSettle();

    // регистрация
    await tester.tap(find.text('Зарегистрировать кошелёк'));
    await tester.pump();
    // опрос готовности узла в AppState.startNode — до 15 с (75 x 200 мс,
    // тема fix-native-strings п.в); фейк отдаёт nodeStats сразу, поэтому
    // цикл выходит на первой итерации, а прокрутка нужна, чтобы
    // завершилась цепочка
    await tester.pump(const Duration(seconds: 6));
    await tester.pumpAndSettle();

    // NONODE не возник: ошибки на экране нет, карточка восстановления
    // показана (регистрация принята)
    expect(find.textContaining('NONODE'), findsNothing);
    expect(find.textContaining('Ошибка'), findsNothing);
    expect(find.text('Карточка восстановления'), findsOneWidget);

    // порядок: сборка регистрации -> старт узла -> публикация
    expect(fake.order, contains('buildRegister'));
    expect(
      fake.order.lastIndexOf('nodeStart'),
      greaterThan(fake.order.indexOf('buildRegister')),
      reason: 'узел должен стартовать после buildRegister (адрес кошелька)',
    );
    expect(fake.order.last, 'submit');
    expect(fake.nodeUp, isTrue, reason: 'узел поднят к моменту публикации');
    expect(fake.calls, contains('submit(tx1)'));
  });
}
