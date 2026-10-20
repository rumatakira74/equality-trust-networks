// Тесты PIN-only на деструктивных экранах (8.2, этап 9 — решение
// владельца 20.10.2026): подтверждение перевода/выпуска/миграции —
// только явный ввод PIN, не биометрия; тревожный PIN на экране перевода
// открывает приманку (неотличимо, без реальной подписи); тревожный PIN
// жив после переводов и рестарта; регресс этапа 8 — блокировка при
// заданном тревожном PIN не вернулась (оба PIN распознаются).
// Платформенный слой мокается (FakePlatformService — модель PinManager:
// оба PIN проверяются одним кодом, ответ на тревожный неотличим).
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:tnp_app/screens/lock_screen.dart';
import 'package:tnp_app/screens/wallet/transfer_screen.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';

import 'fake_platform.dart';

Widget _wrap(Widget child, FakePlatformService fake) {
  return MultiProvider(
    providers: [
      Provider<PlatformService>.value(value: fake),
      ChangeNotifierProvider(create: (_) => AppState(fake)..load()),
    ],
    child: MaterialApp(home: child),
  );
}

Future<void> _fillTransfer(WidgetTester tester) async {
  await tester.enterText(
    find.widgetWithText(TextField, 'Адрес получателя (hex)'),
    'ab' * 32,
  );
  await tester.enterText(
    find.widgetWithText(TextField, 'Сумма (получатель получит ровно столько)'),
    '100',
  );
  await tester.pump();
}

void main() {
  testWidgets('перевод требует PIN: без PIN подписи нет (8.2, этап 9)',
      (tester) async {
    final fake = FakePlatformService()..registered = true;
    fake.setupPin('1234', '9999', 700, []);
    await tester.pumpWidget(_wrap(const TransferScreen(), fake));
    await tester.pumpAndSettle();
    await _fillTransfer(tester);

    await tester.tap(find.text('Отправить'));
    await tester.pumpAndSettle();
    // PIN-промпт — подтверждение операции; биометрического подтверждения
    // на экране перевода больше нет
    expect(find.text('Подтвердите перевод'), findsOneWidget);
    expect(fake.calls, isNot(contains('sign(tx2)')));

    // неверный PIN: подписи нет, ошибка неотличима
    await tester.enterText(find.byType(TextField).last, '0000');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();
    expect(find.text('Неверный PIN'), findsOneWidget);
    expect(fake.calls, isNot(contains('sign(tx2)')));

    // верный PIN: подпись и публикация
    await tester.enterText(find.byType(TextField).last, '1234');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();
    expect(find.text('Перевод отправлен'), findsOneWidget);
    expect(fake.calls, contains('sign(tx2)'));
    expect(fake.calls, contains('submit(tx2)'));
  });

  testWidgets('тревожный PIN на экране перевода открывает приманку (8.2)',
      (tester) async {
    final fake = FakePlatformService()..registered = true;
    fake.setupPin('1234', '9999', 700, []);
    await tester.pumpWidget(_wrap(const TransferScreen(), fake));
    await tester.pumpAndSettle();
    await _fillTransfer(tester);

    await tester.tap(find.text('Отправить'));
    await tester.pumpAndSettle();
    // тревожный PIN: ответ «normal» неотличим, приманка активируется
    // на стороне платформы
    await tester.enterText(find.byType(TextField).last, '9999');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();
    // правдоподобный успех без реальной подписи и публикации
    expect(find.text('Перевод отправлен'), findsOneWidget);
    expect(fake.duress, isTrue);
    expect(fake.calls, contains('signFake(tx2)'));
    expect(fake.calls, contains('submitFake(tx2)'));
    expect(fake.calls, isNot(contains('sign(tx2)')));
    expect(fake.calls, isNot(contains('submit(tx2)')));
  });

  testWidgets('тревожный PIN жив после переводов и рестарта (регресс 8.1)',
      (tester) async {
    final fake = FakePlatformService()..registered = true;
    fake.setupPin('1234', '9999', 700, []);

    // перевод по тревожному PIN (приманка активна)
    await tester.pumpWidget(_wrap(const TransferScreen(), fake));
    await tester.pumpAndSettle();
    await _fillTransfer(tester);
    await tester.tap(find.text('Отправить'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField).last, '9999');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();
    expect(fake.duress, isTrue);

    // рестарт: новое состояние приложения, PIN переживает рестарт
    // (модель PinManager: хранилище переживает процесс)
    await tester.pumpWidget(_wrap(
      LockScreen(key: UniqueKey(), child: const Text('HOME')),
      fake,
    ));
    await tester.pumpAndSettle();
    // отпечаток при активной приманке отключён (8.2)
    expect(find.text('Отпечаток'), findsNothing);
    // тревожный PIN по-прежнему распознаётся — вход неотличим
    await tester.enterText(find.byType(TextField), '9999');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.text('HOME'), findsOneWidget);
    expect(fake.duress, isTrue);

    // обычный PIN выводит из приманки (блокировки нет — регресс этапа 8)
    await tester.pumpWidget(_wrap(
      LockScreen(key: UniqueKey(), child: const Text('HOME')),
      fake,
    ));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '1234');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.text('HOME'), findsOneWidget);
    expect(fake.duress, isFalse);
  });

  testWidgets('с тревожным PIN приложение не блокируется (регресс этапа 8)',
      (tester) async {
    // регресс бага 697205b: с заданным тревожным PIN запись не
    // декодировалась — verify NONE для любого PIN, вечная блокировка
    final fake = FakePlatformService()..pinSet = true;
    fake.setupPin('1234', '9999', 700, []);
    await tester.pumpWidget(_wrap(const LockScreen(child: Text('HOME')), fake));
    await tester.pumpAndSettle();
    // неверный — ошибка
    await tester.enterText(find.byType(TextField), '0000');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.text('Неверный PIN'), findsOneWidget);
    expect(find.text('HOME'), findsNothing);
    // обычный — вход
    await tester.enterText(find.byType(TextField), '1234');
    await tester.tap(find.text('Войти'));
    await tester.pumpAndSettle();
    expect(find.text('HOME'), findsOneWidget);
  });

  testWidgets('вход: отпечаток или PIN; при приманке отпечаток отключён (8.2)',
      (tester) async {
    final fake = FakePlatformService()..pinSet = true;
    fake.setupPin('1234', null, 0, []);
    await tester.pumpWidget(_wrap(const LockScreen(child: Text('HOME')), fake));
    await tester.pumpAndSettle();
    // приманка не активна — вход по отпечатку предлагается
    expect(find.text('Отпечаток'), findsOneWidget);
    await tester.tap(find.text('Отпечаток'));
    await tester.pumpAndSettle();
    expect(fake.calls, contains('biometricLogin'));
    expect(find.text('HOME'), findsOneWidget);

    // при активной приманке отпечаток не предлагается
    final fake2 = FakePlatformService()
      ..pinSet = true
      ..duress = true;
    await tester.pumpWidget(_wrap(const LockScreen(child: Text('HOME')), fake2));
    await tester.pumpAndSettle();
    expect(find.text('Отпечаток'), findsNothing);
  });
}
