// Экран блокировки (8.1, 8.2): вход — отпечаток или PIN, ОДИН экран, один
// ритм для обычного и тревожного PIN; тревожный вход неотличим (мост сам
// активирует приманку и отвечает «normal»). Отпечаток предлагается,
// только если приманка не активна (biometricLoginAllowed): при активном
// тревожном PIN вход по отпечатку отключён (8.2) — иначе приманку
// обходят на входе силой.
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../l10n/app_localizations.dart';
import '../services/platform_service.dart';

class LockScreen extends StatefulWidget {
  final Widget child;
  const LockScreen({super.key, required this.child});

  @override
  State<LockScreen> createState() => _LockScreenState();
}

class _LockScreenState extends State<LockScreen> {
  final _controller = TextEditingController();
  bool _busy = false;
  String? _error;
  bool _unlocked = false;
  bool _biometricAllowed = false;

  @override
  void initState() {
    super.initState();
    _checkBiometric();
  }

  Future<void> _checkBiometric() async {
    final platform = context.read<PlatformService>();
    final allowed = await platform.biometricLoginAllowed();
    if (mounted) setState(() => _biometricAllowed = allowed);
  }

  Future<void> _submit() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    final platform = context.read<PlatformService>();
    // одинаковая задержка для любого исхода (неразличимость, 8.2)
    final res = await platform.verifyPin(_controller.text);
    await Future.delayed(const Duration(milliseconds: 350));
    if (!mounted) return;
    if (res == 'normal') {
      setState(() => _unlocked = true);
    } else {
      // одинаковое сообщение для неверного и тревожного-неверного
      setState(() {
        _error = AppLocalizations.of(context)!.errWrongPin;
        _busy = false;
      });
      _controller.clear();
    }
  }

  // Вход по отпечатку (8.2, этап 9): разблокировка/просмотр — только когда
  // приманка не активна; подтверждение деструктивных операций — PIN.
  Future<void> _biometric() async {
    if (_busy) return;
    setState(() => _busy = true);
    final platform = context.read<PlatformService>();
    final ok = await platform.biometricLogin();
    if (!mounted) return;
    if (ok) {
      setState(() => _unlocked = true);
    } else {
      setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_unlocked) return widget.child;
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.appTitle)),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(l.enterPin, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            TextField(
              controller: _controller,
              autofocus: true,
              obscureText: true,
              keyboardType: TextInputType.number,
              inputFormatters: [FilteringTextInputFormatter.digitsOnly],
              onSubmitted: (_) => _submit(),
              decoration: InputDecoration(errorText: _error),
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: _busy ? null : _submit,
              child: Text(l.btnLogin),
            ),
            if (_biometricAllowed) ...[
              const SizedBox(height: 8),
              OutlinedButton.icon(
                onPressed: _busy ? null : _biometric,
                icon: const Icon(Icons.fingerprint),
                label: Text(l.btnFingerprint),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
