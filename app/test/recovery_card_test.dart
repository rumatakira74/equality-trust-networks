// Тесты карточки восстановления (решение владельца 20.10.2026,
// этап 10): карточка показывает корректные данные после регистрации
// (адрес, гардианы, сеть, QR) и доступна из настроек.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/recovery/recovery_card_screen.dart';
import 'package:tnp_app/screens/settings/settings_screen.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:tnp_app/services/models.dart';
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

  testWidgets('карточка показывает корректные данные после регистрации',
      (tester) async {
    final addr = 'ab' * 32;
    final guardian1 = 'cd' * 32;
    final guardian2 = 'ef' * 32;
    final fake = FakePlatformService()
      ..registered = true
      ..wallet = WalletInfo.fromJson({
        'exists': true,
        'address': addr,
        'status': 'ACTIVE',
        'balance': 0,
        'liquid': 0,
        'threshold_p': 3000,
        'personal_guardians': [guardian1, guardian2],
      });
    await tester.pumpWidget(_wrap(const RecoveryCardScreen(), fake));
    await tester.pumpAndSettle();
    // адрес кошелька
    expect(find.text(addr), findsOneWidget);
    // назначенные гардианы
    expect(find.text(guardian1), findsOneWidget);
    expect(find.text(guardian2), findsOneWidget);
    // объяснение: зачем и как сохранить; напоминание о миграции
    expect(find.textContaining('восстановления кошелька при утрате'),
        findsOneWidget);
    await tester.scrollUntilVisible(
      find.textContaining('менеджер паролей'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.textContaining('менеджер паролей'), findsOneWidget);
    expect(find.textContaining('nonce и подписи гардианов'), findsOneWidget);
    // QR-код карточки присутствует
    expect(find.byType(QrImageView), findsOneWidget);
  });

  testWidgets('без гардианов карточка честно предупреждает', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const RecoveryCardScreen(), fake));
    await tester.pumpAndSettle();
    expect(find.textContaining('Гардианы не назначены'), findsOneWidget);
    expect(find.byType(QrImageView), findsOneWidget);
  });

  testWidgets('карточка доступна из настроек', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const SettingsScreen(), fake));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text('Карточка восстановления'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Карточка восстановления'));
    await tester.pumpAndSettle();
    expect(find.byType(RecoveryCardScreen), findsOneWidget);
  });

  test('QR-код карточки содержит адрес, сеть и гардианов', () {
    final payload = recoveryCardPayload(
      'ab' * 32,
      ['cd' * 32, 'ef' * 32],
      'TNP Test',
      'netid-123',
    );
    expect(payload, contains('addr=${'ab' * 32}'));
    expect(payload, contains('net=TNP Test'));
    expect(payload, contains('id=netid-123'));
    expect(payload, contains('guardians=${'cd' * 32},${'ef' * 32}'));
  });
}
