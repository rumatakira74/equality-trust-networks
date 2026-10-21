// Тесты экрана запуска миграции (находка №5 прогона stage11-prep):
// nonce генерируется при открытии и сразу виден — гардиан не мог
// подписать MigrateStatement без nonce, публикация миграции из
// приложения была невозможна. Nonce отображается до ввода подписей
// гардианов; копирование в буфер.
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/migration/migration_screen.dart';
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
  String? clipboardText;

  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
    TestWidgetsFlutterBinding.ensureInitialized();
    // буфер обмена — платформенный канал: без мока invokeMethod не
    // завершается в widget-тесте
    TestWidgetsFlutterBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(SystemChannels.platform, (call) async {
      switch (call.method) {
        case 'Clipboard.setData':
          clipboardText = call.arguments['text'] as String?;
          return null;
        case 'Clipboard.getData':
          return clipboardText == null ? null : {'text': clipboardText};
      }
      return null;
    });
  });

  testWidgets('nonce виден до ввода подписей гардианов', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const StartMigrationScreen(), fake));
    await tester.pumpAndSettle();

    final l = AppLocalizations.of(
        tester.element(find.byType(StartMigrationScreen)))!;
    // подпись-метка nonce (передайте гардианам)
    expect(find.text(l.nonceToCopy), findsOneWidget);
    // сам nonce — число 1..2^30, до любого ввода подписей
    final nonceText = tester.widget<SelectableText>(
      find.byType(SelectableText),
    );
    final nonce = int.parse(nonceText.data!);
    expect(nonce, greaterThan(0));
    expect(nonce, lessThanOrEqualTo(1 << 30));
    // подписей ещё нет — кнопка публикации отключена
    expect(find.text(l.btnPublishProposal), findsOneWidget);
    final button = tester.widget<FilledButton>(
      find.ancestor(
        of: find.text(l.btnPublishProposal),
        matching: find.byType(FilledButton),
      ),
    );
    expect(button.onPressed, isNull);
  });

  testWidgets('копирование nonce в буфер', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const StartMigrationScreen(), fake));
    await tester.pumpAndSettle();

    clipboardText = null;
    await tester.tap(find.byIcon(Icons.copy));
    await tester.pump();
    expect(clipboardText, isNotNull);
    expect(int.parse(clipboardText!), greaterThan(0));
  });
}
