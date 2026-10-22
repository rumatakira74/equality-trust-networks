// PENDING №7 (прогон 11.2): открытая клавиатура перекрывала кнопки
// экранов ввода. LockScreen (ввод PIN): прокрутка под инсеты
// клавиатуры — кнопка входа остаётся доступной, переполнения нет.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/lock_screen.dart';
import 'package:tnp_app/services/platform_service.dart';

import 'fake_platform.dart';

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('клавиатура не перекрывает кнопку входа (PENDING №7)',
      (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          Provider<PlatformService>.value(value: fake),
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
          home: const LockScreen(child: SizedBox()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(LockScreen)))!;

    // открытая клавиатура (инсет 45% высоты)
    tester.view.viewInsets = FakeViewPadding(
      bottom: tester.view.physicalSize.height * 0.45,
    );
    addTearDown(tester.view.reset);
    await tester.pumpAndSettle();

    // нет переполнения, кнопка в дереве и достижима прокруткой
    expect(tester.takeException(), isNull);
    await tester.scrollUntilVisible(
      find.text(l.btnLogin),
      100,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(find.text(l.btnLogin), findsOneWidget);
    final btn = tester.widget<FilledButton>(
      find.ancestor(
        of: find.text(l.btnLogin),
        matching: find.byType(FilledButton),
      ),
    );
    expect(btn.onPressed, isNotNull);
  });
}
