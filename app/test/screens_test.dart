// Widget-тесты экранов приложения (этап 7, п.8): платформенный слой
// мокается (FakePlatformService). Проверяются ключевые правила
// KNOWLEDGE: отказ без биометрии (8.1), обязательность шага гардианов
// (11), PIN-вход (8.2), статус кандидата до K (7.4), предупреждение
// о недоказанном присутствии (7.4), отложенные переводы (8.3),
// «получено по Bluetooth — ожидает публикации» (12.4), приманка (8.2).
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:tnp_app/screens/lock_screen.dart';
import 'package:tnp_app/screens/onboarding/onboarding_flow.dart';
import 'package:tnp_app/screens/vouching/vouching_screen.dart';
import 'package:tnp_app/screens/wallet/pending_screen.dart';
import 'package:tnp_app/screens/wallet/wallet_screen.dart';
import 'package:tnp_app/services/models.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';

import 'fake_platform.dart';

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
  // мок хранилища настроек (этап 10): выбор языка сохраняется в
  // SharedPreferences; в тестах плагин подменяется
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('онбординг: без биометрии — отказ и системный экран (8.1)',
      (tester) async {
    final fake = FakePlatformService()..biometrics = false;
    await tester.pumpWidget(_wrap(const OnboardingFlow(), fake));
    await tester.pumpAndSettle();
    expect(find.textContaining('Отпечаток не настроен'), findsOneWidget);
    expect(find.text('Настроить отпечаток'), findsOneWidget);
    // продолжения нет — только настройка биометрии
    expect(find.text('Далее'), findsNothing);
    await tester.tap(find.text('Настроить отпечаток'));
    await tester.pumpAndSettle();
    expect(fake.calls, contains('enroll'));
  });

  testWidgets('онбординг: биометрия есть -> шаг PIN -> шаг гардианов (8.1, 11)',
      (tester) async {
    final fake = FakePlatformService()..biometrics = true;
    await tester.pumpWidget(_wrap(const OnboardingFlow(), fake));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();
    // шаг PIN: обычный обязателен
    expect(find.textContaining('Задать тревожный PIN'), findsOneWidget);
    await tester.enterText(
      find.widgetWithText(TextField, 'PIN (обычный)'),
      '1234',
    );
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();
    // шаг гардианов: предложение обязательно, пропустить можно
    expect(find.textContaining('личных гардианов'), findsOneWidget);
    expect(find.text('Пропустить'), findsOneWidget);
    // выбор гардиана из реестра (R6: известные адреса)
    await tester.tap(find.byType(CheckboxListTile));
    await tester.pumpAndSettle();
    await tester.tap(find.textContaining('Выбрано: 1'));
    await tester.pumpAndSettle();
    // шаг register: подтверждение -> сборка с гардианами
    await tester.tap(find.text('Зарегистрировать кошелёк'));
    await tester.pumpAndSettle();
    expect(fake.builtTxs, contains('register(1)'));
    expect(fake.calls, contains('sign(tx1)'));
    expect(fake.calls, contains('submit(tx1)'));
  });

  testWidgets('блокировка: неверный PIN — ошибка, верный — вход (8.2)',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..pinResult = 'none';
    await tester.pumpWidget(
      _wrap(const LockScreen(child: Text('HOME')), fake),
    );
    await tester.enterText(find.byType(TextField), '0000');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.text('Неверный PIN'), findsOneWidget);
    expect(find.text('HOME'), findsNothing);
    // верный (и тревожный — ответ один и тот же «normal»)
    fake.pinResult = 'normal';
    await tester.enterText(find.byType(TextField), '1234');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.text('HOME'), findsOneWidget);
  });

  testWidgets('кошелёк: баланс с демереджем честно, awaiting-карточка (10.3, 12.4)',
      (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const WalletScreen(), fake));
    await tester.pumpAndSettle();
    expect(find.text('1500'), findsOneWidget);
    expect(find.textContaining('демередж применён'), findsOneWidget);
    expect(
      find.text('Получено по Bluetooth — ожидает публикации'),
      findsOneWidget,
    );
    expect(find.textContaining('300 от'), findsOneWidget);
  });

  testWidgets('поручительства: прогресс до K и предупреждение (7.4)',
      (tester) async {
    final fake = FakePlatformService();
    await tester.pumpWidget(_wrap(const VouchingScreen(), fake));
    await tester.pumpAndSettle();
    expect(find.text('Поручительств: 1 из 3'), findsOneWidget);
    // вкладка «Поручиться» — предупреждение обязательно
    await tester.tap(find.text('Поручиться'));
    await tester.pumpAndSettle();
    expect(
      find.textContaining('НЕ доказывает присутствие'),
      findsOneWidget,
    );
    // вкладка слотов: занятые/свободные, статистика
    await tester.tap(find.text('Мои слоты'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Свободно: 1'), findsOneWidget);
    expect(find.textContaining('активны 1'), findsOneWidget);
  });

  testWidgets('отложенные: отсчёт, отмена, смена P (8.3)', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const PendingScreen(), fake));
    await tester.pumpAndSettle();
    expect(find.textContaining('5000 →'), findsOneWidget);
    expect(find.text('Отменить'), findsOneWidget);
    expect(find.textContaining('Порог P: 3000'), findsOneWidget);
    expect(find.textContaining('P_max = 30000'), findsOneWidget);
  });

  testWidgets('приманка: легенда неотличима от обычного баланса (8.2)',
      (tester) async {
    // при тревожном входе мост возвращает легенду тем же методом
    // walletInfo — экран показывает её как обычный баланс
    final fake = FakePlatformService()
      ..registered = true
      ..wallet = WalletInfo.fromJson({
        'exists': true,
        'address': '',
        'status': 'ACTIVE',
        'balance': 700,
        'liquid': 700,
        'threshold_p': 3000,
        'decoy': true,
      });
    await tester.pumpWidget(_wrap(const WalletScreen(), fake));
    await tester.pumpAndSettle();
    expect(find.text('700'), findsOneWidget);
    // экран не содержит признаков приманки
    expect(find.byType(Card), findsWidgets);
    expect(find.textContaining('приманка'), findsNothing);
  });

  test('модели: манифест и параметры сети (п.1: без хардкода)', () {
    final m = ManifestParams.fromJson({
      'vouch_k': 5,
      'transfer_delay_hours': 24,
      'delayed_threshold_default': 100,
      'delayed_threshold_max': 1000,
      'transfer_mode': 'soulbound',
    });
    expect(m.vouchK, 5);
    expect(m.transferDelayHours, 24);
    // soulbound: отложенные не применяются (8.3)
    expect(m.transfersDelayed, isFalse);
  });
}
