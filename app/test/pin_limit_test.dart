// Тесты лимита попыток PIN (тема UI-защиты владельца, п.1): 5 неудач —
// блокировка ввода с сообщением об остатке секунд; успешный вход
// (обычный ИЛИ тревожный — неотличимо, 8.2) сбрасывает счётчик;
// во время блокировки проверка не выполняется. Платформенный слой
// мокается (FakePlatformService — модель PinAttemptPolicy/PinManager).
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:tnp_app/screens/lock_screen.dart';
import 'package:tnp_app/screens/pin_gate.dart';
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
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('экран блокировки: 5 неудач -> блокировка с остатком секунд',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', '9876', 100, []);
    await tester.pumpWidget(
        _wrap(const LockScreen(child: SizedBox()), fake));
    await tester.pumpAndSettle();

    for (var i = 0; i < 4; i++) {
      await tester.enterText(find.byType(TextField), '0000');
      await tester.tap(find.text('Войти'));
      await tester.pumpAndSettle();
      expect(find.text('Неверный PIN'), findsOneWidget);
    }
    // 5-я неудача — порог: сообщение блокировки с остатком секунд
    await tester.enterText(find.byType(TextField), '0000');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Слишком много неудачных попыток'),
        findsOneWidget);
    expect(find.textContaining('30'), findsOneWidget);
    expect(fake.pinFailures, 5);
  });

  testWidgets('во время блокировки даже верный PIN не проверяется',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', null, 100, [])
      ..pinFailures = 5; // уже заблокировано (30 с)
    await tester.pumpWidget(
        _wrap(const LockScreen(child: SizedBox()), fake));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '1234');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    // вход не произошёл, счётчик не сброшен, сообщение блокировки
    expect(find.textContaining('Слишком много неудачных попыток'),
        findsOneWidget);
    expect(fake.pinFailures, 5);
  });

  testWidgets('успешный вход сбрасывает счётчик (обычный PIN)',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', null, 100, [])
      ..pinFailures = 4; // одна неудача до порога
    await tester.pumpWidget(
        _wrap(const LockScreen(child: SizedBox()), fake));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '1234');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(fake.pinFailures, 0);
  });

  testWidgets('успешный тревожный вход тоже сбрасывает (неотличимо, 8.2)',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', '9876', 100, [])
      ..pinFailures = 4;
    await tester.pumpWidget(
        _wrap(const LockScreen(child: SizedBox()), fake));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '9876');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(fake.pinFailures, 0);
    expect(fake.duress, true); // приманка активна — вход был «успешным»
  });

  testWidgets('PIN-диалог деструктивной операции: блокировка видна и там',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', null, 100, [])
      ..pinFailures = 5; // заблокировано
    await tester.pumpWidget(_wrap(
      Builder(builder: (context) => Scaffold(
        body: FilledButton(
          onPressed: () => confirmPin(context),
          child: const Text('op'),
        ),
      )),
      fake,
    ));
    await tester.pumpAndSettle();

    await tester.tap(find.text('op'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '0000');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Слишком много неудачных попыток'),
        findsOneWidget);
  });
}
