// Человекочитаемые сообщения об ошибках транзакций по конвенции кодов
// E_* (spec/00 0.4; этап 12: сгоревшая recurring-единица — понятная
// ошибка, не сырой код).
import '../l10n/app_localizations.dart';

/// Локализованное сообщение для кода отказа; null — показать общий
/// «Отклонено: код» (неизвестный код).
String? localizedTxError(AppLocalizations l, String code) {
  switch (code) {
    case 'E_BURNED':
      // v9.10, раздел 4: перевод единицы периода P отвергается после
      // границы P+1; сброса баланса нет — история проверяема
      return l.burnedUnitError;
    default:
      return null;
  }
}
