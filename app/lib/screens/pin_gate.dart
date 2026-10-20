// PIN-промпт деструктивной операции (8.2, этап 9 — решение владельца
// 20.10.2026): подтверждение перевода, выпуска, миграции propose — только
// явный ввод PIN, не биометрия. Принцип: «PIN-промпт — точка входа в
// режим-приманку, биометрия принуждаема физически» — палец прикладывают
// силой, PIN-ввод сохраняет жертве выбор: тревожный PIN активирует
// приманку на стороне платформы, ответ «normal» неотличим, операция
// «проходит» правдоподобно без реальной подписи. Один экран, один ритм
// для обоих PIN (неразличимость, 8.2).
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../l10n/app_localizations.dart';
import '../services/platform_service.dart';

/// Подтверждение деструктивной операции вводом PIN.
/// true — PIN принят (обычный или тревожный — неотличимо); false —
/// отменено пользователем.
Future<bool> confirmPin(
  BuildContext context, {
  String? title,
}) async {
  title ??= AppLocalizations.of(context)!.confirmOperation;
  final resolvedTitle = title;
  final ok = await showDialog<bool>(
    context: context,
    barrierDismissible: false,
    builder: (_) => _PinDialog(title: resolvedTitle),
  );
  return ok ?? false;
}

class _PinDialog extends StatefulWidget {
  final String title;
  const _PinDialog({required this.title});

  @override
  State<_PinDialog> createState() => _PinDialogState();
}

class _PinDialogState extends State<_PinDialog> {
  final _controller = TextEditingController();
  String? _error;
  bool _busy = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    final platform = context.read<PlatformService>();
    // один и тот же вызов и ритм для любого PIN (неразличимость)
    final res = await platform.verifyPin(_controller.text);
    await Future.delayed(const Duration(milliseconds: 350));
    if (!mounted) return;
    if (res == 'normal') {
      Navigator.of(context).pop(true);
    } else {
      // неверный и тревожный-неверный — одинаковое сообщение
      setState(() {
        _error = AppLocalizations.of(context)!.errWrongPin;
        _busy = false;
      });
      _controller.clear();
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(widget.title),
      content: TextField(
        controller: _controller,
        autofocus: true,
        obscureText: true,
        keyboardType: TextInputType.number,
        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        onSubmitted: (_) => _submit(),
        decoration: InputDecoration(errorText: _error),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(false),
          child: Text(AppLocalizations.of(context)!.btnCancel),
        ),
        FilledButton(
          onPressed: _busy ? null : _submit,
          child: Text(AppLocalizations.of(context)!.btnConfirm),
        ),
      ],
    );
  }
}
