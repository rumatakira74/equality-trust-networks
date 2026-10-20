// Точка входа приложения (этап 7). Провайдеры состояния, маршрут:
// загрузка -> (онбординг | экран блокировки) -> главная навигация.
// Локализация (этап 10): ru, en, es, pt, zh — состав подтверждён
// владельцем 20.10.2026; fallback — английский; выбор — в настройках.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:provider/provider.dart';

import 'l10n/app_localizations.dart';
import 'screens/home_shell.dart';
import 'screens/lock_screen.dart';
import 'screens/onboarding/onboarding_flow.dart';
import 'services/platform_service.dart';
import 'state/app_state.dart';

void main() {
  runApp(const TnpApp());
}

class TnpApp extends StatelessWidget {
  const TnpApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        Provider<PlatformService>.value(value: MethodChannelPlatformService()),
        ChangeNotifierProvider(
          create: (ctx) => AppState(ctx.read<PlatformService>())..load(),
        ),
      ],
      child: Consumer<AppState>(
        builder: (context, app, _) => MaterialApp(
          onGenerateTitle: (ctx) =>
              AppLocalizations.of(ctx)?.appTitle ?? 'Trust Networks',
          debugShowCheckedModeBanner: false,
          theme: ThemeData(
            colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF2E6E4E)),
            useMaterial3: true,
          ),
          // пять языков этапа 10; null — язык системы, fallback — en
          locale: app.locale,
          supportedLocales: AppLocalizations.supportedLocales,
          localizationsDelegates: const [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          home: const _Gate(),
        ),
      ),
    );
  }
}

/// Шлюз: загрузка -> онбординг (без PIN) -> блокировка -> главная.
class _Gate extends StatelessWidget {
  const _Gate();

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    if (app.loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    if (!app.pinSet) return const OnboardingFlow();
    return const LockScreen(child: HomeShell());
  }
}
