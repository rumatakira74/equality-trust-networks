// Кошелёк (промпт этапа 7, п.4): баланс — эффективный остаток с
// демереджем, показанный честно (10.3: списание материализуется на
// границе периода; баланс уже честен), история цепочки (9.1),
// отложенные переводы с отменой и сменой P (8.3), Bluetooth-платежи
// со статусом «получено по Bluetooth — ожидает публикации» (12.4).
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import 'mint_screen.dart';
import 'pending_screen.dart';
import 'transfer_screen.dart';

class WalletScreen extends StatefulWidget {
  const WalletScreen({super.key});

  @override
  State<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends State<WalletScreen> {
  List<HistoryEntry> _history = [];
  List<PendingTransfer> _pending = [];
  List<AwaitingTx> _awaiting = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final platform = context.read<PlatformService>();
    try {
      final history = await platform.history(limit: 50);
      final pending = await platform.pendingTransfers();
      final awaiting = await platform.awaiting();
      if (!mounted) return;
      setState(() {
        _history = history;
        _pending = pending.where((p) => !p.canceled && !p.executed).toList();
        _awaiting = awaiting;
        _loading = false;
      });
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final wallet = app.wallet;
    final manifest = app.manifest;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Кошелёк'),
        actions: [
          IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh)),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                if (wallet != null) _balanceCard(context, wallet, manifest),
                const SizedBox(height: 8),
                if (_pending.isNotEmpty)
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.schedule),
                      title: Text('Отложенных переводов: ${_pending.length}'),
                      subtitle: const Text(
                        'Крупные переводы исполняются через задержку D; '
                        'отменить можно в течение D',
                      ),
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const PendingScreen()),
                      ),
                    ),
                  ),
                if (_awaiting.isNotEmpty) _awaitingCard(context),
                const SizedBox(height: 8),
                FilledButton.icon(
                  onPressed: wallet?.active == true
                      ? () => Navigator.of(context).push(MaterialPageRoute(
                            builder: (_) => const MintScreen(),
                          ))
                      : null,
                  icon: const Icon(Icons.download),
                  label: const Text('Выпуск (челлендж)'),
                ),
                const SizedBox(height: 8),
                FilledButton.icon(
                  onPressed: wallet != null
                      ? () => Navigator.of(context).push(MaterialPageRoute(
                            builder: (_) => const TransferScreen(),
                          ))
                      : null,
                  icon: const Icon(Icons.send),
                  label: const Text('Перевод'),
                ),
                const SizedBox(height: 16),
                Text('История', style: Theme.of(context).textTheme.titleMedium),
                const Divider(),
                if (_history.isEmpty)
                  const Text('Пока пусто'),
                for (final h in _history)
                  ListTile(
                    dense: true,
                    leading: _iconFor(h.type),
                    title: Text(h.describe()),
                    subtitle: h.timestamp > 0
                        ? Text(DateTime.fromMillisecondsSinceEpoch(
                              h.timestamp * 1000,
                            ).toLocal().toString().substring(0, 16))
                        : null,
                  ),
              ],
            ),
    );
  }

  Widget _balanceCard(
      BuildContext context, WalletInfo wallet, ManifestParams? manifest) {
    final demurrage = manifest?.demurrageEnabled == true;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    '${wallet.balance}',
                    style: Theme.of(context)
                        .textTheme
                        .headlineMedium
                        ?.copyWith(fontWeight: FontWeight.bold),
                  ),
                ),
                _statusChip(wallet),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              'эффективный остаток${demurrage ? ' (демередж применён)' : ''}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            if (wallet.locked > 0)
              Text('в резерве отложенных: ${wallet.locked}'),
            if (wallet.flagged)
              const Padding(
                padding: EdgeInsets.only(top: 8),
                child: Text(
                  'Кошелёк помечен (flagged) — метка публична, вердиктов нет',
                  style: TextStyle(color: Colors.orange),
                ),
              ),
            if (wallet.registered && !wallet.active)
              const Padding(
                padding: EdgeInsets.only(top: 8),
                child: Text(
                  'Статус registered: для активации нужно K поручительств '
                  '(вкладка «Поручительства»)',
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _statusChip(WalletInfo wallet) {
    final color = wallet.active
        ? Colors.green
        : (wallet.status == 'EXPIRED' ? Colors.red : Colors.blueGrey);
    return Chip(
      label: Text(wallet.status),
      backgroundColor: color.withValues(alpha: 0.15),
      labelStyle: TextStyle(color: color, fontWeight: FontWeight.bold),
      visualDensity: VisualDensity.compact,
    );
  }

  Widget _awaitingCard(BuildContext context) {
    return Card(
      color: Colors.amber.withValues(alpha: 0.15),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Получено по Bluetooth — ожидает публикации',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            // 12.4: «получено по Bluetooth» ≠ оплата, до подтверждения
            // сети платёж не считается
            for (final a in _awaiting)
              ListTile(
                dense: true,
                leading: const Icon(Icons.bluetooth),
                title: Text('${a.amount} от ${a.recipient.shortHash()}'),
                subtitle: Text('статус: ${a.status}'),
              ),
            TextButton(
              onPressed: () async {
                await context.read<PlatformService>().publishAwaiting();
                _refresh();
              },
              child: const Text('Опубликовать (сеть доступна)'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _iconFor(String type) {
    switch (type) {
      case 'REGISTER':
        return const Icon(Icons.app_registration);
      case 'VOUCH':
        return const Icon(Icons.handshake);
      case 'MINT':
        return const Icon(Icons.download);
      case 'TRANSFER':
        return const Icon(Icons.swap_horiz);
      case 'MIGRATE':
        return const Icon(Icons.switch_account);
      case 'GUARDIAN_OFFER':
        return const Icon(Icons.volunteer_activism);
      default:
        return const Icon(Icons.circle_outlined);
    }
  }
}
