// Выпуск с челленджем (промпт п.4; 7.6, 10.1): усиленный PoW в
// рандомном слоте внутри месяца, окно ±48 ч; точный момент слота в
// UI не показывается (PENDING, отчёт этапа 7). Пропуск — перенос в
// пределах accrual_window, средства не блокируются.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';

class MintScreen extends StatefulWidget {
  const MintScreen({super.key});

  @override
  State<MintScreen> createState() => _MintScreenState();
}

class _MintScreenState extends State<MintScreen> {
  MintWindow? _window;
  bool _busy = false;
  String? _result;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final w = await context.read<PlatformService>().mintWindow();
      if (mounted) setState(() => _window = w);
    } catch (e) {
      if (mounted) setState(() => _result = '$e');
    }
  }

  Future<void> _mint() async {
    setState(() {
      _busy = true;
      _result = 'Усиленный PoW (Argon2id)…';
    });
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    try {
      // amount = эмиссия × (period − last_claimed); узел пересчитает
      final w = _window!;
      final amount = w.issuanceAmount * (w.currentPeriod - w.lastClaimedPeriod);
      final res = await app.signAndSubmit(
        () => platform.buildMint(w.currentPeriod, amount),
        'Подтвердите выпуск',
      );
      setState(() => _result =
          res.accepted ? 'Выпуск опубликован' : 'Отклонено: ${res.code}');
      await app.refreshWallet();
    } catch (e) {
      setState(() => _result = 'Ошибка: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final w = _window;
    return Scaffold(
      appBar: AppBar(title: const Text('Выпуск')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (w != null) ...[
              Text(
                'Период: ${w.currentPeriod}',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 8),
              Text(
                w.open
                    ? 'Окно челленджа открыто (±${w.windowHours} ч вокруг '
                      'слота). Можно выпускать.'
                    : 'Окно челленджа закрыто. Слот внутри месяца '
                      'непредсказуем заранее; пропуск переносится в '
                      'пределах окна накопления (${w.accrualWindowPeriods} '
                      'периодов), средства не блокируются.',
              ),
              const SizedBox(height: 16),
              Text('Последний заявленный период: ${w.lastClaimedPeriod}'),
              Text('Эмиссия за период: ${w.issuanceAmount}'),
              const SizedBox(height: 24),
              FilledButton(
                onPressed: w.open && !_busy ? _mint : null,
                child: const Text('Выпустить (усиленный PoW)'),
              ),
            ],
            if (_result != null) ...[
              const SizedBox(height: 16),
              Text(_result!, textAlign: TextAlign.center),
            ],
          ],
        ),
      ),
    );
  }
}
