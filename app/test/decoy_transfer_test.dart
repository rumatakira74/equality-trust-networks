// Тесты приманки на Flutter-пути (A6, тема UI-защиты владельца):
// перевод в режиме приманки отвечает правдоподобным «ожидает
// подтверждения» (спека 8.2), реальной подписи и публикации нет
// (журнал мока: signFake/submitFake); честный перевод вне приманки
// проходит полный путь (sign/submit) — регресс.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:tnp_app/screens/wallet/transfer_screen.dart';
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

Future<void> _fillAndSend(WidgetTester tester) async {
  await tester.enterText(
    find.widgetWithText(TextField, 'Адрес получателя (hex)'),
    'ab' * 32,
  );
  await tester.enterText(
    find.widgetWithText(TextField, 'Сумма (получатель получит ровно столько)'),
    '100',
  );
  await tester.pump();
  await tester.tap(find.text('Отправить'));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets(
      'приманка: перевод отвечает «ожидает подтверждения», реестр не тронут',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', '9876', 100, [])
      ..duress = true; // приманка активна (тревожный вход уже сделан)
    await tester.pumpWidget(_wrap(const TransferScreen(), fake));
    await tester.pumpAndSettle();

    await _fillAndSend(tester);
    // PIN-диалог: тревожный PIN «проходит» неотличимо (владелец под
    // принуждением вводит тревожный — приманка остаётся активной)
    await tester.enterText(find.byType(TextField).last, '9876');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();

    // правдоподобный ответ — «ожидает подтверждения» (не «недоступен»)
    expect(find.text('Перевод отправлен — ожидает подтверждения'),
        findsOneWidget);
    // реальной подписи и публикации не было — только fake-путь моста
    expect(fake.calls, containsAll(['signFake(tx2)', 'submitFake(tx2)']));
    expect(fake.calls.where((c) => c == 'sign(tx2)'), isEmpty);
    expect(fake.calls.where((c) => c == 'submit(tx2)'), isEmpty);
    expect(fake.duress, isTrue); // приманка не вскрыта
  });

  testWidgets('регресс: честный перевод вне приманки проходит полный путь',
      (tester) async {
    final fake = FakePlatformService()
      ..pinSet = true
      ..setupPin('1234', null, 100, []);
    await tester.pumpWidget(_wrap(const TransferScreen(), fake));
    await tester.pumpAndSettle();

    await _fillAndSend(tester);
    await tester.enterText(find.byType(TextField).last, '1234');
    await tester.tap(find.text('Подтвердить'));
    await tester.pumpAndSettle();

    // полный путь: подпись и публикация реальные, ответ обычный
    // (без снапшота кошелька в AppState перевод уходит в ветку
    // «отложен» — порог P не загружен; путь тот же: sign + submit)
    expect(fake.calls, containsAll(['sign(tx2)', 'submit(tx2)']));
    expect(find.textContaining('Перевод отложен'), findsOneWidget);
  });
}
