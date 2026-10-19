// Экран блокировки (8.1, 8.2): вход по PIN — ОДИН экран, один ритм
// для обычного и тревожного PIN; тревожный вход неотличим (мост сам
// активирует приманку и отвечает «normal»). Отпечаток предлагается,
// только если приманка не активна (biometricLoginAllowed).
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

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
        _error = 'Неверный PIN';
        _busy = false;
      });
      _controller.clear();
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_unlocked) return widget.child;
    return Scaffold(
      appBar: AppBar(title: const Text('Сети доверия')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text('Введите PIN', textAlign: TextAlign.center),
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
              child: const Text('Войти'),
            ),
          ],
        ),
      ),
    );
  }
}
