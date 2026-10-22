// Тесты экранов смены гардианов (11, решение владельца 21.10.2026):
// экран «Гардианы» (текущий список, остаток лимита «использовано N из
// 150», отсчёт окна 48 ч, отмена), перевыбор — PIN-only (этап 9),
// тревожный PIN не трогает реальный список (приманка 8.2), экран
// согласия гардиана (подпись хеша смены биометрией).
import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:tnp_app/l10n/app_localizations.dart';
import 'package:tnp_app/screens/settings/guardians_screen.dart';
import 'package:tnp_app/services/models.dart';
import 'package:tnp_app/services/platform_service.dart';
import 'package:tnp_app/state/app_state.dart';
import 'package:tnp_app/widgets/hex_chunk_field.dart';

import 'fake_platform.dart';

/// Ввод длинной hex-строки в поле-чанки: вставка полной строки в
/// первый чанк — распределение по полям (PENDING №6).
Future<void> enterHex(WidgetTester tester, String label, String value) async {
  final field = find.ancestor(
    of: find.text(label),
    matching: find.byType(HexChunkField),
  );
  await tester.enterText(
    find.descendant(of: field, matching: find.byType(TextField)).first,
    value,
  );
  await tester.pump();
}

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

FakePlatformService _withWallet({int changes = 0, List<String> guardians = const []}) {
  final fake = FakePlatformService()
    ..registered = true
    ..wallet = WalletInfo.fromJson({
      'exists': true,
      'address': '11' * 32,
      'status': 'ACTIVE',
      'balance': 1500,
      'liquid': 1500,
      'threshold_p': 3000,
      'seq': 5,
      'guardian_changes': changes,
      'personal_guardians': guardians,
    });
  return fake;
}

void main() {
  setUpAll(() async {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('экран «Гардианы»: текущий список и остаток лимита',
      (tester) async {
    final fake = _withWallet(
      changes: 7,
      guardians: ['22' * 32, '33' * 32],
    );
    await tester.pumpWidget(_wrap(const GuardiansScreen(), fake));
    await tester.pumpAndSettle();

    final l = AppLocalizations.of(
        tester.element(find.byType(GuardiansScreen)))!;
    // текущий список
    expect(find.text(l.guardiansCurrent), findsOneWidget);
    expect(find.text(('22' * 32).shortHash()), findsOneWidget);
    expect(find.text(('33' * 32).shortHash()), findsOneWidget);
    // остаток лимита: использовано 7 из 150
    expect(find.text(l.guardiansLimitUsed(7, 150)), findsOneWidget);
    // без активной смены доступен перевыбор
    expect(find.text(l.guardiansReelect), findsOneWidget);
  });

  testWidgets('активная смена: отсчёт окна 48 ч и отмена', (tester) async {
    final fake = _withWallet(guardians: ['22' * 32]);
    fake.guardianUpdates = [
      PendingGuardianUpdate.fromJson({
        'propose_seq': 6,
        'new_guardians': ['33' * 32],
        'effective_at': 100000,
        'remaining_sec': 40 * 3600,
        'nonce': 1,
      }),
    ];
    fake.setupPin('1234', '9999', 700, []);
    await tester.pumpWidget(_wrap(const GuardiansScreen(), fake));
    await tester.pumpAndSettle();

    final l = AppLocalizations.of(
        tester.element(find.byType(GuardiansScreen)))!;
    // отсчёт окна
    expect(find.text(l.guardiansPendingSub(40)), findsOneWidget);
    // при активной смене новая не публикуется
    final reelect = tester.widget<FilledButton>(
      find.ancestor(
        of: find.text(l.guardiansReelect),
        matching: find.byType(FilledButton),
      ),
    );
    expect(reelect.onPressed, isNull);

    // отмена — PIN-only (этап 9): без PIN подписи нет
    expect(fake.calls, isNot(contains('sign(tx11)')));
    await tester.tap(find.text(l.guardiansCancelBtn));
    await tester.pumpAndSettle();
    expect(find.text(l.cancelChangeConfirmTitle), findsOneWidget);
    // неверный PIN — отмены нет
    await tester.enterText(find.byType(TextField).last, '0000');
    await tester.tap(find.text(l.btnConfirm));
    await tester.pumpAndSettle();
    expect(find.text(l.errWrongPin), findsOneWidget);
    expect(fake.calls, isNot(contains('sign(tx11)')));
    // верный PIN — подпись и публикация отмены
    await tester.enterText(find.byType(TextField).last, '1234');
    await tester.tap(find.text(l.btnConfirm));
    await tester.pumpAndSettle();
    expect(fake.calls, contains('sign(tx11)'));
    expect(fake.calls, contains('submit(tx11)'));
    expect(find.text(l.changeCanceled), findsOneWidget);
  });

  testWidgets('перевыбор: публикация только по PIN', (tester) async {
    final fake = _withWallet(guardians: ['22' * 32]);
    fake.setupPin('1234', null, 0, []);
    await tester.pumpWidget(_wrap(const GuardiansScreen(), fake));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(GuardiansScreen)))!;

    await tester.tap(find.text(l.guardiansReelect));
    await tester.pumpAndSettle();
    expect(find.text(l.reelectTitle), findsOneWidget);

    // новый список — хеш смены появляется сразу (для передачи гардианам)
    await tester.enterText(
      find.widgetWithText(TextField, l.newGuardiansLabel),
      '33' * 32,
    );
    await tester.pump();
    expect(fake.calls, contains('changeHash(1)'));
    expect(find.text(l.changeHashToCopy), findsOneWidget);

    // пара «гардиан + подпись» — чанки по 20 (PENDING №6)
    await enterHex(tester, l.guardianAddrLabel, '33' * 32);
    await enterHex(tester, l.guardianSigLabel, 'cd' * 32);
    await tester.tap(find.text(l.btnAddSig));
    await tester.pump();

    // публикация без PIN невозможна; кнопка активна только с согласиями
    // (кнопка ниже вьюпорта — доскролливаем)
    await tester.scrollUntilVisible(
      find.text(l.btnPublishChange),
      100,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pump();
    final publish = tester.widget<FilledButton>(
      find.ancestor(
        of: find.text(l.btnPublishChange),
        matching: find.byType(FilledButton),
      ),
    );
    expect(publish.onPressed, isNotNull);

    await tester.tap(find.text(l.btnPublishChange));
    await tester.pumpAndSettle();
    expect(find.text(l.changeConfirmTitle), findsOneWidget);
    expect(fake.calls, isNot(contains('sign(tx10)')));
    // неверный PIN — публикации нет
    await tester.enterText(find.byType(TextField).last, '0000');
    await tester.tap(find.text(l.btnConfirm));
    await tester.pumpAndSettle();
    expect(fake.calls, isNot(contains('sign(tx10)')));
    // верный PIN — подпись и публикация
    await tester.enterText(find.byType(TextField).last, '1234');
    await tester.tap(find.text(l.btnConfirm));
    await tester.pumpAndSettle();
    expect(fake.calls, contains('sign(tx10)'));
    expect(fake.calls, contains('submit(tx10)'));
    expect(find.text(l.changePublished), findsOneWidget);
  });

  testWidgets('тревожный PIN: приманка, реальный список не меняется (8.2)',
      (tester) async {
    final fake = _withWallet(changes: 3, guardians: ['22' * 32]);
    fake.setupPin('1234', '9999', 700, []);
    await tester.pumpWidget(_wrap(const GuardiansScreen(), fake));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(GuardiansScreen)))!;

    await tester.tap(find.text(l.guardiansReelect));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.widgetWithText(TextField, l.newGuardiansLabel),
      '33' * 32,
    );
    await tester.pump();
    await enterHex(tester, l.guardianAddrLabel, '33' * 32);
    await enterHex(tester, l.guardianSigLabel, 'cd' * 32);
    await tester.tap(find.text(l.btnAddSig));
    await tester.pump();
    await tester.scrollUntilVisible(
      find.text(l.btnPublishChange),
      100,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pump();
    await tester.tap(find.text(l.btnPublishChange));
    await tester.pumpAndSettle();

    // тревожный PIN: ответ неотличим, приманка активна на платформе
    await tester.enterText(find.byType(TextField).last, '9999');
    await tester.tap(find.text(l.btnConfirm));
    await tester.pumpAndSettle();
    expect(fake.duress, isTrue);
    // правдоподобный успех без реальной подписи и публикации
    expect(find.text(l.changePublished), findsOneWidget);
    expect(fake.calls, contains('signFake(tx10)'));
    expect(fake.calls, contains('submitFake(tx10)'));
    expect(fake.calls, isNot(contains('sign(tx10)')));
    expect(fake.calls, isNot(contains('submit(tx10)')));
    // реальный список гардианов не меняется: данные легенды
    expect(fake.wallet.personalGuardians, ['22' * 32]);
    expect(fake.wallet.guardianChanges, 3);
  });

  testWidgets('экран согласия гардиана: подпись хеша смены', (tester) async {
    final fake = _withWallet();
    await tester.pumpWidget(_wrap(const GuardianConsentScreen(), fake));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(GuardianConsentScreen)))!;

    await enterHex(tester, l.wardAddressLabel, '11' * 32);
    await enterHex(tester, l.changeHashLabel, 'ab' * 32);
    await tester.tap(find.text(l.btnSignBiometric));
    await tester.pumpAndSettle();
    expect(fake.calls, contains('signGuardianChange'));
    // подпись — для передачи подопечному вручную
    expect(find.text(l.sigToCopy), findsOneWidget);
    expect(find.text(fake.guardianChangeSig), findsOneWidget);
  });

  testWidgets('согласие в приманке: ключ не используется (8.2)', (tester) async {
    final fake = _withWallet()..duress = true;
    await tester.pumpWidget(_wrap(const GuardianConsentScreen(), fake));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(GuardianConsentScreen)))!;

    await enterHex(tester, l.wardAddressLabel, '11' * 32);
    await enterHex(tester, l.changeHashLabel, 'ab' * 32);
    await tester.tap(find.text(l.btnSignBiometric));
    await tester.pumpAndSettle();
    // ответ приманки неотличим по структуре, подписи нет
    expect(fake.calls, isNot(contains('signGuardianChange')));
    expect(find.text(l.signNotConfirmed), findsOneWidget);
  });

  testWidgets('согласие: таймаут биометрии не вешает UI (PENDING №3)',
      (tester) async {
    // платформенный слой не отвечает (таймаут BiometricPrompt на
    // эмуляторах): ожидание ограничено, экран возвращается в рабочее
    // состояние
    final fake = _HangingSignFake();
    await tester.pumpWidget(_wrap(
      const GuardianConsentScreen(signTimeout: Duration(milliseconds: 100)),
      fake,
    ));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(GuardianConsentScreen)))!;

    await enterHex(tester, l.wardAddressLabel, '11' * 32);
    await enterHex(tester, l.changeHashLabel, 'ab' * 32);
    await tester.tap(find.text(l.btnSignBiometric));
    await tester.pump();
    // идёт ожидание: кнопка занята
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    // таймаут: ошибка, экран снова рабочий
    await tester.pump(const Duration(milliseconds: 150));
    expect(find.text(l.biometricTimeout), findsOneWidget);
    final btn = tester.widget<FilledButton>(
      find.ancestor(
        of: find.text(l.btnSignBiometric),
        matching: find.byType(FilledButton),
      ),
    );
    expect(btn.onPressed, isNotNull);
    // повторная попытка возможна (платформа ответила позже — не теряет UI)
    fake.completeSig();
    await tester.tap(find.text(l.btnSignBiometric));
    await tester.pumpAndSettle();
    expect(find.text(l.sigToCopy), findsOneWidget);
  });

  testWidgets('согласие: отмена биометрии возвращает экран в рабочее состояние',
      (tester) async {
    final fake = _withWallet();
    fake.guardianChangeSig = ''; // отмена диалога: подписи нет
    await tester.pumpWidget(_wrap(
      const GuardianConsentScreen(signTimeout: Duration(seconds: 5)),
      fake,
    ));
    await tester.pumpAndSettle();
    final l = AppLocalizations.of(
        tester.element(find.byType(GuardianConsentScreen)))!;

    await enterHex(tester, l.wardAddressLabel, '11' * 32);
    await enterHex(tester, l.changeHashLabel, 'ab' * 32);
    await tester.tap(find.text(l.btnSignBiometric));
    await tester.pumpAndSettle();
    expect(find.text(l.signNotConfirmed), findsOneWidget);
    final btn = tester.widget<FilledButton>(
      find.ancestor(
        of: find.text(l.btnSignBiometric),
        matching: find.byType(FilledButton),
      ),
    );
    expect(btn.onPressed, isNotNull);
  });
}

/// Мок платформы: signGuardianChange не отвечает до completeSig
/// (таймаут BiometricPrompt, PENDING №3).
class _HangingSignFake extends FakePlatformService {
  final _completer = Completer<Map<String, dynamic>>();
  void completeSig() =>
      _completer.complete({'sig': 'cd' * 32});
  @override
  Future<Map<String, dynamic>> signGuardianChange(
          String hashHex, String title) =>
      _completer.future;
}
