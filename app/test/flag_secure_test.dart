// Тесты FLAG_SECURE (тема UI-защиты владельца, п.2): окно помечено,
// когда виден экран с балансом/суммами (вкладка кошелька, перевод,
// выпуск, отложенные), и не помечено на публичных экранах
// (поручительства/реестр/настройки). Причины-ключи: возврат с pushed-
// экрана на вкладку кошелька не снимает пометку (IndexedStack).
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:tnp_app/screens/home_shell.dart';
import 'package:tnp_app/screens/wallet/transfer_screen.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/services/secure_mode.dart';
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

  setUp(() {
    SecureMode.reset();
  });

  testWidgets('вкладка кошелька: окно помечено, настройки — нет',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..registered = true;
    await tester.pumpWidget(_wrap(const HomeShell(), fake));
    await tester.pumpAndSettle();

    // старт — вкладка кошелька: FLAG_SECURE включён
    expect(fake.secureModeCalls.isNotEmpty, isTrue);
    expect(fake.secureModeCalls.last, isTrue);

    // вкладка «Настройки» — публичный экран: пометка снята
    await tester.tap(find.text('Настройки'));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isFalse);

    // возврат на кошелёк — снова помечено
    await tester.tap(find.text('Кошелёк'));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isTrue);
  });

  testWidgets('перевод поверх кошелька: пометка держится и после возврата',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..registered = true;
    await tester.pumpWidget(_wrap(const HomeShell(), fake));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isTrue);

    // переход на экран перевода (кнопка ниже первого экрана списка)
    await tester.scrollUntilVisible(
      find.text('Перевод'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Перевод'));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isTrue);

    // возврат на вкладку кошелька — причина вкладки ещё активна
    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isTrue);
  });

  testWidgets('перевод с настроек: пометка только пока экран открыт',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..registered = true;
    await tester.pumpWidget(_wrap(const HomeShell(), fake));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Настройки'));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isFalse);

    // pushed-экран перевода помечает окно сам
    final ctx = tester.element(find.byType(NavigationBar));
    Navigator.of(ctx)
        .push(MaterialPageRoute(builder: (_) => const TransferScreen()));
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isTrue);

    // закрытие перевода на публичной вкладке — пометка снята
    Navigator.of(tester.element(find.byType(TransferScreen))).pop();
    await tester.pumpAndSettle();
    expect(fake.secureModeCalls.last, isFalse);
  });
}
