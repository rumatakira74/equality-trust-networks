// Тест кодировок (тема UI-защиты владельца, п.7): строка с
// кириллицей и спецсимволами, пришедшая из платформенного слоя
// (модель пути Go -> gomobile -> Kotlin -> Flutter: ManifestJSON
// с display_name), отображается на экране без искажений.
// Go-часть цепочки — mobile/encoding_test.go (TestCyrillicStringChainRoundTrip);
// контрольная сверка «лог против экрана» — эмуляторный прогон темы.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:tnp_app/screens/onboarding/onboarding_flow.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/services/models.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';

import 'fake_platform.dart';

void main() {
  testWidgets('кириллица и спецсимволы из платформенного слоя — на экране',
      (tester) async {
    SharedPreferences.setMockInitialValues({});
    const name = 'Сеть доверия №1 — «тест», ünïcode ✓ 日本語';
    final fake = FakePlatformService()
      ..manifestParams = ManifestParams.fromJson({'display_name': name})
      ..biometrics = true;
    await tester.pumpWidget(
      MultiProvider(
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
          home: const OnboardingFlow(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // displayPair = '$displayName · $eqLabel' — строка целиком на экране
    expect(find.textContaining(name), findsOneWidget);
  });
}
