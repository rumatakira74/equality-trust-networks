// Тесты поля ввода длинных hex-строк чанками (PENDING №6, прогон 11.2:
// IME эмуляторов теряет/искажает символы при вставке длинной строки).
// Вставка полной строки распределяется по чанкам, недопустимые
// символы отфильтровываются, длины/чётность контролируются.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/widgets/hex_chunk_field.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    locale: const Locale('ru'),
    supportedLocales: AppLocalizations.supportedLocales,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    home: Scaffold(body: ListView(children: [child])),
  );
}

void main() {
  testWidgets('вставка полной строки распределяется по чанкам',
      (tester) async {
    String value = '';
    await tester.pumpWidget(_wrap(HexChunkField(
      label: 'адрес',
      chunkCount: 4,
      expectedLength: 64,
      onChanged: (v) => value = v,
    )));
    // вставка 64 символов в первый чанк
    await tester.enterText(find.byType(TextField).first, 'ab' * 32);
    await tester.pump();
    // значение собрано целиком, без потерь
    expect(value, 'ab' * 32);
    // распределение: 20 + 20 + 20 + 4
    final fields = tester.widgetList<TextField>(find.byType(TextField)).toList();
    expect(fields.length, 4);
    expect(HexChunkField.isValid(value, 64), isTrue);
  });

  testWidgets('недопустимые символы отфильтровываются', (tester) async {
    String value = '';
    await tester.pumpWidget(_wrap(HexChunkField(
      label: 'адрес',
      chunkCount: 4,
      expectedLength: 64,
      onChanged: (v) => value = v,
    )));
    await tester.enterText(
      find.byType(TextField).first,
      'ZZ${'ab' * 31}GG',
    );
    await tester.pump();
    expect(value, 'ab' * 31);
    expect(HexChunkField.isValid(value, 64), isFalse);
    // второй контур контроля символов (IME-композиция): sanitize
    expect(HexChunkField.sanitize('ZZ${'ab' * 31}GG'), 'ab' * 31);
    expect(HexChunkField.sanitize('AB cd'), 'abcd');
  });

  testWidgets('фиксированная длина: контроль счётчика и ошибки',
      (tester) async {
    await tester.pumpWidget(_wrap(HexChunkField(
      label: 'хеш',
      chunkCount: 4,
      expectedLength: 64,
    )));
    final l = AppLocalizations.of(
        tester.element(find.byType(HexChunkField)))!;
    await tester.enterText(find.byType(TextField).first, 'ab' * 10);
    await tester.pump();
    // счётчик: введено 20 из 64
    expect(find.text(l.hexChunksCount(20, 64)), findsOneWidget);
    // избыточная вставка: ошибка длины
    await tester.enterText(find.byType(TextField).first, 'cd' * 40);
    await tester.pump();
    expect(find.text(l.hexLenError(64)), findsOneWidget);
  });

  testWidgets('подпись (переменная длина): чётность контролируется',
      (tester) async {
    await tester.pumpWidget(_wrap(HexChunkField(
      label: 'подпись',
      chunkCount: 8,
    )));
    final l = AppLocalizations.of(
        tester.element(find.byType(HexChunkField)))!;
    await tester.enterText(find.byType(TextField).first, 'abc');
    await tester.pump();
    expect(find.text(l.hexSigEvenError), findsOneWidget);
    expect(HexChunkField.isValid('abc', 0), isFalse);
    await tester.enterText(find.byType(TextField).first, 'abcd');
    await tester.pump();
    expect(HexChunkField.isValid('abcd', 0), isTrue);
  });

  testWidgets('авто-переход фокуса на следующий чанк', (tester) async {
    await tester.pumpWidget(_wrap(HexChunkField(
      label: 'адрес',
      chunkCount: 4,
      expectedLength: 64,
    )));
    await tester.enterText(find.byType(TextField).first, 'a' * 20);
    await tester.pump();
    // фокус перешёл во второй чанк: поле подписи — с фокусом
    final fields = tester.widgetList<TextField>(
      find.byType(TextField),
    ).toList();
    expect(fields[1].focusNode!.hasFocus, isTrue);
    expect(fields[0].focusNode!.hasFocus, isFalse);
  });
}
