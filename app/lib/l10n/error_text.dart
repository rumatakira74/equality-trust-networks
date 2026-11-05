// Тексты ошибок платформенного слоя (FINDING-03, тема fix-lang-tails).
//
// Мост отдаёт код и техническое сообщение (например, NONODE: «узел не
// запущен»). Пользователю показывается перевод из ARB — единого источника
// переводов; нативное сообщение остаётся диагностикой. Неизвестный код
// показывается как есть, чтобы не терять информацию при разборе.
import '../services/platform_service.dart';
import 'app_localizations.dart';

extension LocalizedError on AppLocalizations {
  /// Текст ошибки для показа пользователю.
  String errorText(Object e) {
    if (e is TnpPlatformException && e.code == 'NONODE') return errNoNode;
    return errorWith('$e');
  }
}
