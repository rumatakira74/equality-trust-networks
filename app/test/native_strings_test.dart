// Контракт генерации нативных строк Android из ARB (тема fix-native-strings).
//
// Сам генератор живёт в сборочном скрипте — gradle-задача
// generateNativeStrings (app/android/app/build.gradle.kts) — и из Dart не
// запускается: в CI его выполняет сборка APK и падает на пропавшем ключе.
// Юнит-тест страхует стык, который сборкой не проверяется: вход генератора
// (шесть ключей во всех девяти ARB) и совпадение имён ресурсов с теми, что
// ищет Kotlin. Рассинхрон имени сборку не роняет — строка молча уходит в
// fallback-литерал, поэтому проверяется отдельно.
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Ожидаемое соответствие: ключ ARB -> имя Android-ресурса.
const expectedKeys = <String, String>{
  'appLabel': 'app_name',
  'nodeChannelName': 'node_channel_name',
  'nodeNotificationTitle': 'node_notification_title',
  'nodeNotificationText': 'node_notification_text',
  'errTxNotFound': 'err_tx_not_found',
  'signDefaultTitle': 'sign_default_title',
};

/// en — шаблон ARB (values/), остальные восемь — `values-<lang>/`.
const locales = ['en', 'ru', 'es', 'pt', 'zh', 'fr', 'de', 'it', 'ja'];

void main() {
  test('нативные строки: ключи есть во всех девяти ARB', () {
    final resourceName = RegExp(r'^[a-z][a-z0-9_]*$');
    for (final lang in locales) {
      final file = File('lib/l10n/app_$lang.arb');
      expect(file.existsSync(), isTrue, reason: 'нет ARB: ${file.path}');
      final data =
          jsonDecode(file.readAsStringSync()) as Map<String, dynamic>;
      for (final entry in expectedKeys.entries) {
        final value = data[entry.key];
        expect(
          value,
          isA<String>(),
          reason: '$lang: в ARB нет строкового ключа ${entry.key}',
        );
        expect(
          (value as String).trim(),
          isNotEmpty,
          reason: '$lang: пустое значение ${entry.key}',
        );
        expect(
          resourceName.hasMatch(entry.value),
          isTrue,
          reason: '${entry.value} — не валидное имя Android-ресурса',
        );
      }
    }
  });

  test('нативные строки: имена ресурсов совпадают со сборкой и Kotlin', () {
    // карта в build.gradle.kts — источник имён для генератора
    final gradle = File('android/app/build.gradle.kts').readAsStringSync();
    final block = RegExp(
      r'val nativeStringKeys = linkedMapOf\(([\s\S]*?)\n\)',
    ).firstMatch(gradle);
    expect(block, isNotNull, reason: 'в build.gradle.kts нет nativeStringKeys');
    final declared = <String, String>{
      for (final m
          in RegExp(r'"(\w+)"\s+to\s+"(\w+)"').allMatches(block!.group(1)!))
        m.group(1)!: m.group(2)!,
    };
    expect(
      declared,
      expectedKeys,
      reason: 'карта ARB -> ресурс в сборке разошлась с ожидаемой',
    );

    // где каждое имя обязано встречаться: рассинхрон имени — это молчаливый
    // fallback в Kotlin, а не ошибка сборки
    final nodeService = File(
      '../android/app/src/main/kotlin/platform/tnp/android/node/NodeService.kt',
    );
    final bridge = File(
      'android/app/src/main/kotlin/platform/tnp/android/flutterbridge/'
      'FlutterBridge.kt',
    );
    for (final file in [nodeService, bridge]) {
      expect(file.existsSync(), isTrue, reason: 'нет файла: ${file.path}');
    }
    final usage = <String, String>{
      'node_channel_name': nodeService.readAsStringSync(),
      'node_notification_title': nodeService.readAsStringSync(),
      'node_notification_text': nodeService.readAsStringSync(),
      'err_tx_not_found': bridge.readAsStringSync(),
      'sign_default_title': bridge.readAsStringSync(),
      // label приложения — манифест через manifestPlaceholders
      'app_name': gradle,
    };
    for (final entry in usage.entries) {
      expect(
        entry.value.contains(entry.key),
        isTrue,
        reason: 'имя ресурса ${entry.key} не найдено там, где его читают',
      );
    }
    expect(
      gradle.contains('@string/app_name'),
      isTrue,
      reason: 'label приложения должен по умолчанию ссылаться на @string/app_name',
    );
  });
}
