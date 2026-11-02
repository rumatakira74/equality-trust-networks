// Тесты F5 (аудит контура 1): в режиме-приманки ВСЕ точки выдачи адреса
// возвращают подменной адрес легенды — валидного формата, стабильный между
// сессиями/рестартами, не равный реальному и не связанный с ним; выход
// обычным PIN возвращает реальный адрес; реальный кошелёк (реестр) при
// этом не тронут. Контракт моста (Kotlin): walletAddress -> displayAddress.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/recovery/recovery_card_screen.dart';
import 'package:tnp_app/screens/vouching/vouching_screen.dart';
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

Widget _wrapWith(Widget child, FakePlatformService fake, AppState app) {
  return MultiProvider(
    providers: [
      Provider<PlatformService>.value(value: fake),
      ChangeNotifierProvider<AppState>.value(value: app),
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

  test('F5: приманка — адрес валиден, не равен реальному, стабилен между '
      'рестартами; выход обычным PIN — реальный адрес', () async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..registered = true
      ..setupPin('1234', '9876', 100, ['2026-01-01 — 100 — ab']);
    final real = fake.wallet.address;

    // тревожный вход — приманка активна, ответ неотличим
    expect(await fake.verifyPin('9876'), 'normal');
    expect(fake.duress, isTrue);

    // точка выдachi адреса (мост walletAddress) — подменной
    final a1 = await fake.walletAddress();
    expect(a1, isNot(real));
    expect(RegExp(r'^[0-9a-f]{64}$').hasMatch(a1), isTrue,
        reason: 'подменной адрес должен быть валидного формата (64 hex)');

    // AppState видит подменной адрес
    final app1 = AppState(fake);
    await app1.load();
    expect(app1.address, a1);

    // «рестарт» приложения: новый AppState, повторная загрузка — адрес тот же
    final app2 = AppState(fake);
    await app2.load();
    expect(app2.address, a1, reason: 'адрес приманки обязан быть стабилен');

    // decoy-кошелёк (walletInfo) согласован: тот же подменной адрес
    final w = await fake.walletInfo();
    expect(w.address, a1);

    // реальный кошелёк при этом не тронут (внутреннее состояние — реальный)
    expect(fake.wallet.address, real);
    expect((await fake.registry()).single.address, isNot(a1));

    // выход обычным PIN — реальный адрес
    expect(await fake.verifyPin('1234'), 'normal');
    expect(fake.duress, isFalse);
    expect(await fake.walletAddress(), real);
    final app3 = AppState(fake);
    await app3.load();
    expect(app3.address, real);
  });

  testWidgets('F5: карточка восстановления в приманке показывает подменной '
      'адрес (текст и QR), реальный отсутствует', (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..registered = true
      ..setupPin('1234', '9876', 100, []);
    final real = fake.wallet.address;
    await fake.verifyPin('9876'); // приманка активна
    final decoy = await fake.walletAddress();
    expect(decoy, isNot(real));

    await tester.pumpWidget(_wrap(const RecoveryCardScreen(), fake));
    await tester.pumpAndSettle();

    // на экране — подменной адрес, реального нет
    expect(find.text(decoy), findsOneWidget);
    expect(find.text(real), findsNothing);

    // QR-пейлоад карточки — с подменным адресом
    final app = AppState(fake);
    await app.load();
    final payload = recoveryCardPayload(
        app.address, const [], 'TNP Test Genesis', '4e465553');
    expect(payload, contains('addr=$decoy'));
    expect(payload, isNot(contains(real)));
  });

  testWidgets('F5: реальная последовательность — load() под замком кэширует '
      'реальный адрес, вход тревожным PIN + refreshAfterUnlock — приманка '
      'во всех точках (карточка, QR), выход обычным PIN — реальный',
      (tester) async {
    // поверхность как у устройства: кнопка QR ниже дефолтных 600 px
    tester.view.physicalSize = const Size(1080, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final fake = FakePlatformService()
      ..pinSet = true
      ..registered = true
      ..setupPin('1234', '9876', 100, []);
    final real = fake.wallet.address;

    // старт приложения под замком: снапшот кэширует реальный адрес
    final app = AppState(fake);
    await app.load();
    expect(fake.duress, isFalse);
    expect(app.address, real);

    // вход тревожным PIN: мост активирует приманку, ответ неотличим
    expect(await fake.verifyPin('9876'), 'normal');
    expect(fake.duress, isTrue);

    // разблокировка перечитывает адрес и снапшот кошелька (фикс F5)
    await app.refreshAfterUnlock();
    final decoy = app.address;
    expect(decoy, isNot(real));
    expect(RegExp(r'^[0-9a-f]{64}$').hasMatch(decoy), isTrue);
    expect(app.wallet?.address, decoy,
        reason: 'снапшот кошелька должен быть перечитан в режиме приманки');

    // карточка восстановления: текст и QR — подменной, реального нет
    await tester.pumpWidget(_wrapWith(const RecoveryCardScreen(), fake, app));
    await tester.pumpAndSettle();
    expect(find.text(decoy), findsOneWidget);
    expect(find.text(real), findsNothing);
    final payload = recoveryCardPayload(
        app.address, const [], 'TNP Test Genesis', '4e465553');
    expect(payload, contains('addr=$decoy'));
    expect(payload, isNot(contains(real)));

    // QR ваучинга: рендерится из app.address (vouching_screen.dart:220) —
    // тот же подменной адрес, реального нет
    await tester.pumpWidget(_wrapWith(const VouchingScreen(), fake, app));
    await tester.pumpAndSettle();
    // вкладка «Поручиться»: QR-кнопка живёт там
    await tester.tap(find.text('Поручиться'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.byIcon(Icons.qr_code));
    await tester.tap(find.byIcon(Icons.qr_code));
    await tester.pumpAndSettle();
    expect(find.byType(QrImageView), findsOneWidget);
    expect(app.address, decoy);
    expect(app.address, isNot(real));

    // выход обычным PIN: снова реальный адрес
    expect(await fake.verifyPin('1234'), 'normal');
    expect(fake.duress, isFalse);
    await app.refreshAfterUnlock();
    expect(app.address, real);
    expect(app.wallet?.address, real);
  });
}
