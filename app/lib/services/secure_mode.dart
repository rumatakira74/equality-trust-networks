// FLAG_SECURE (тема UI-защиты владельца, п.2): окно активности помечается,
// когда виден экран с балансом/суммами — системные скриншоты и превью
// «недавних приложений» выходят пустыми. Причины-ключи: экран/вкладка
// включает свою причину; окно помечено, пока активна хотя бы одна —
// вкладка кошелька в IndexedStack живёт одновременно с pushed-экранами
// (перевод/выпуск/отложенные), и снятие флага при возврате на вкладку
// не должно открывать баланс для фотографирования.
import '../services/platform_service.dart';

class SecureMode {
  static final Map<String, bool> _reasons = {};
  static bool _applied = false;

  /// Включить/выключить причину пометки (ключ — экран или вкладка).
  static Future<void> set(
      PlatformService platform, String key, bool on) async {
    final wasActive = _reasons.values.any((v) => v);
    if (on) {
      _reasons[key] = true;
    } else {
      _reasons.remove(key);
    }
    final want = _reasons.values.any((v) => v);
    if (want == wasActive && want == _applied) return;
    try {
      await platform.setSecureMode(want);
      _applied = want;
    } catch (_) {
      // мост недоступен (тесты/платформа) — повтор на следующей смене
    }
  }

  /// Сброс состояния (тесты).
  static void reset() {
    _reasons.clear();
    _applied = false;
  }
}
