// PENDING №4 (прогон 11.2): экран History после рестарта приложения/
// узла показывал «Nothing yet» — история запрашивалась до завершения
// сверки и не обновлялась. Экран перезапрашивает историю, когда узел
// поднялся или пришёл новый снапшот кошелька (рестарт узла, сверка).
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/wallet/wallet_screen.dart';
import 'package:tnp_app/services/models.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';

import 'fake_platform.dart';

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('History обновляется после рестарта узла (PENDING №4)',
      (tester) async {
    // высокий экран: история видна без прокрутки (ListView ленивый)
    tester.view.physicalSize = const Size(800, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    // до сверки журнал пуст: приложение поднялось, узел ещё нет
    final fake = FakePlatformService()..registered = true;
    fake.historyEntries = [];
    final app = AppState(fake);
    await tester.pumpWidget(
      MultiProvider(
        providers: [
          Provider<PlatformService>.value(value: fake),
          ChangeNotifierProvider.value(value: app),
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
          home: const WalletScreen(),
        ),
      ),
    );
    await app.load();
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(WalletScreen)))!;
    expect(find.text(l.historyEmpty), findsOneWidget);

    // рестарт узла: сверка завершилась, история появилась —
    // AppState обновился, экран перезапросил историю без перехода
    fake.historyEntries = [
      HistoryEntry.fromJson({'seq': 1, 'type': 'REGISTER', 'timestamp': 100}),
      HistoryEntry.fromJson({
        'seq': 2, 'type': 'MINT', 'amount': 2000, 'period': 1,
        'timestamp': 200,
      }),
    ];
    await app.refreshWallet();
    await tester.pumpAndSettle();

    expect(find.text(l.historyEmpty), findsNothing);
    expect(find.text(l.txRegister), findsOneWidget);
    expect(find.text(l.txMint(1)), findsOneWidget);
  });
}
