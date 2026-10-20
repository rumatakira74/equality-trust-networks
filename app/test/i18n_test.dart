// Тесты локализации (этап 10): пять языков (ru, en, es, pt, zh — состав
// подтверждён владельцем 20.10.2026), fallback — английский; выбор
// языка в настройках меняет интерфейс немедленно.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/settings/settings_screen.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';

import 'fake_platform.dart';

Widget _wrap(Widget child, FakePlatformService fake, Locale? locale) {
  // как в TnpApp: MaterialApp перечитывает locale из AppState —
  // переключение языка действует немедленно
  return MultiProvider(
    providers: [
      Provider<PlatformService>.value(value: fake),
      ChangeNotifierProvider(create: (_) => AppState(fake)..load()),
    ],
    child: Consumer<AppState>(
      builder: (context, app, _) => MaterialApp(
        locale: locale ?? app.locale,
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: const [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        home: child,
      ),
    ),
  );
}

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('пять языков: ключевые строки экрана настроек', (tester) async {
    final cases = {
      const Locale('ru'): ['Настройки', 'Язык', 'Демередж'],
      const Locale('en'): ['Settings', 'Language', 'Demurrage'],
      const Locale('es'): ['Ajustes', 'Idioma', 'Demurrage'],
      const Locale('pt'): ['Ajustes', 'Idioma', 'Demurrage'],
      const Locale('zh'): ['设置', '语言', '损耗费'],
    };
    for (final entry in cases.entries) {
      final fake = FakePlatformService();
      await tester.pumpWidget(_wrap(const SettingsScreen(), fake, entry.key));
      await tester.pumpAndSettle();
      for (final text in entry.value) {
        expect(
          find.text(text),
          findsOneWidget,
          reason: 'локаль ${entry.key}: нет строки «$text»',
        );
      }
      await tester.pumpWidget(const SizedBox());
    }
  });

  testWidgets('выбор языка в настройках меняет интерфейс и сохраняется',
      (tester) async {
    final fake = FakePlatformService();
    // locale: null — язык из AppState; без сохранённого выбора —
    // системный (в тестах — английский, fallback)
    await tester.pumpWidget(_wrap(const SettingsScreen(), fake, null));
    await tester.pumpAndSettle();
    expect(find.text('Settings'), findsOneWidget);
    // переключение на русский чипом выбора
    await tester.tap(find.text('Русский'));
    await tester.pumpAndSettle();
    expect(find.text('Настройки'), findsOneWidget);
    expect(find.text('Settings'), findsNothing);
    // выбор сохранён в хранилище
    final prefs = await SharedPreferences.getInstance();
    expect(prefs.getString('ui_locale'), 'ru');
    // обратно на системный язык
    await tester.tap(find.text('Системный'));
    await tester.pumpAndSettle();
    expect(prefs.getString('ui_locale'), '');
    expect(find.text('Settings'), findsOneWidget);
  });

  test('все пять локалей поддержаны делегатом', () async {
    const expected = {'ru', 'en', 'es', 'pt', 'zh'};
    final supported =
        AppLocalizations.supportedLocales.map((l) => l.languageCode).toSet();
    expect(supported, expected);
    // fallback — английский: шаблон ARB — app_en
  });
}
