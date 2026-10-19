// Шаблонный тест заменён реальными тестами (см. test/ каталог);
// этот файл проверяет только сборку приложения.
import 'package:flutter_test/flutter_test.dart';
import 'package:tnp_app/services/models.dart';

void main() {
  test('модели парсятся из JSON моста', () {
    final m = ManifestParams.fromJson({
      'vouch_k': 3,
      'transfer_delay_hours': 48,
      'delayed_threshold_default': 3000,
      'delayed_threshold_max': 30000,
      'transfer_mode': 'free',
      'sandbox': false,
    });
    expect(m.vouchK, 3);
    expect(m.transferDelayHours, 48);
    expect(m.transfersDelayed, isTrue);
  });

  test('адрес сокращается для UI', () {
    expect('a'.padRight(64, 'b').shortHash().contains('…'), isTrue);
  });
}
