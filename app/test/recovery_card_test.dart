// Тесты карточки восстановления (решение владельца 20.10.2026, спека
// v9.7 раздел 11; аудит и доработка — этап 10.1): данные на экране
// соответствуют узлу; QR содержит ожидаемые поля; пояснения (хранение
// не в облако, APK, данные миграции — только гардиану); экспорт
// (системный share изображения); вход из настроек и с экрана миграции;
// экран появляется при создании кошелька (после регистрации).
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/migration/migration_screen.dart';
import 'package:tnp_app/screens/onboarding/onboarding_flow.dart';
import 'package:tnp_app/screens/recovery/recovery_card_screen.dart';
import 'package:tnp_app/screens/settings/settings_screen.dart';
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
    // объяснения (п.4): зачем, как хранить — вне телефона, не в облако
    expect(find.textContaining('восстановления кошелька при утрате'),
        findsOneWidget);
    await tester.scrollUntilVisible(
      find.textContaining('не храните в облаке'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.textContaining('не храните в облаке'), findsOneWidget);
    // пояснение про APK той сети (п.2)
    expect(find.textContaining('APK'), findsOneWidget);
    // напоминание (п.5): только напрямую гардиану, удалить после миграции
    expect(find.textContaining('напрямую вашему гардиану'), findsOneWidget);
    expect(find.textContaining('удалите их из переписок'), findsOneWidget);
    // QR-код карточки (п.3)
    expect(find.byType(QrImageView), findsOneWidget);
    // кнопка экспорта (п.3): системный share изображения
    expect(find.text('Поделиться карточкой (изображение)'), findsOneWidget);
  });

  testWidgets('без гардианов карточка честно предупреждает', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const RecoveryCardScreen(), fake));
    await tester.pumpAndSettle();
    expect(find.textContaining('Гардианы не назначены'), findsOneWidget);
    expect(find.byType(QrImageView), findsOneWidget);
  });

  testWidgets('экспорт: share изображения карточки (п.3)', (tester) async {
    // системный share подменён (реальный IO share_plus не живёт в
    // fake-зоне); мок получает PNG карточки
    Uint8List? captured;
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(
      RecoveryCardScreen(shareOverride: (png) async {
        captured = png;
      }),
      fake,
    ));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text('Поделиться карточкой (изображение)'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    // захват изображения (RepaintBoundary.toImage) завершается в
    // реальной асинхронной зоне
    await tester.runAsync(() async {
      await tester.tap(find.text('Поделиться карточкой (изображение)'));
      await Future<void>.delayed(const Duration(milliseconds: 300));
    });
    await tester.pumpAndSettle();
    expect(captured, isNotNull);
    expect(captured!.length, greaterThan(0));
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

  testWidgets('карточка доступна с экрана миграции', (tester) async {
    final fake = FakePlatformService()..registered = true;
    await tester.pumpWidget(_wrap(const MigrationScreen(), fake));
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

  testWidgets('карточка появляется при создании кошелька (онбординг)',
      (tester) async {
    final fake = FakePlatformService()..biometrics = true;
    await tester.pumpWidget(_wrap(const OnboardingFlow(), fake));
    await tester.pumpAndSettle();
    // биометрия -> PIN
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.widgetWithText(TextField, 'PIN (обычный)'),
      '1234',
    );
    await tester.tap(find.text('Далее'));
    await tester.pumpAndSettle();
    // гардианы: пропустить
    await tester.tap(find.text('Пропустить'));
    await tester.pumpAndSettle();
    // регистрация (мок: подпись и публикация проходят)
    await tester.tap(find.text('Зарегистрировать кошелёк'));
    await tester.pumpAndSettle();
    // после регистрации — карточка восстановления (п.1)
    expect(find.byType(RecoveryCardScreen), findsOneWidget);
    expect(find.text('Карточка восстановления'), findsOneWidget);
    // завершение онбординга кнопкой карточки
    await tester.scrollUntilVisible(
      find.text('Сохранено — продолжить'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Сохранено — продолжить'));
    await tester.pumpAndSettle();
    expect(fake.calls, contains('submit(tx1)'));
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
