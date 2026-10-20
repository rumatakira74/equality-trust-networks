// Кошелёк (промпт этапа 7, п.4): баланс — эффективный остаток с
// демереджем, показанный честно (10.3: списание материализуется на
// границе периода; баланс уже честен), история цепочки (9.1),
// отложенные переводы с отменой и сменой P (8.3), Bluetooth-платежи
// со статусом «получено по Bluetooth — ожидает публикации» (12.4).
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
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
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(
        title: Text(l.navWallet),
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
                      title: Text(l.pendingCount(_pending.length)),
                      subtitle: Text(l.pendingHint),
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
                  label: Text(l.btnMint),
                ),
                const SizedBox(height: 8),
                FilledButton.icon(
                  onPressed: wallet != null
                      ? () => Navigator.of(context).push(MaterialPageRoute(
                            builder: (_) => const TransferScreen(),
                          ))
                      : null,
                  icon: const Icon(Icons.send),
                  label: Text(l.btnTransfer),
                ),
                const SizedBox(height: 16),
                Text(l.historyTitle, style: Theme.of(context).textTheme.titleMedium),
                const Divider(),
                if (_history.isEmpty) Text(l.historyEmpty),
                for (final h in _history)
                  ListTile(
                    dense: true,
                    leading: _iconFor(h.type),
                    title: Text(describeHistory(l, h)),
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
    final l = AppLocalizations.of(context)!;
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
              demurrage ? l.effBalanceDem : l.effBalance,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            if (wallet.locked > 0) Text(l.lockedReserve(wallet.locked)),
            if (wallet.flagged)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  l.flaggedNotice,
                  style: const TextStyle(color: Colors.orange),
                ),
              ),
            if (wallet.registered && !wallet.active)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(l.registeredNeedK),
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
    final l = AppLocalizations.of(context)!;
    return Card(
      color: Colors.amber.withValues(alpha: 0.15),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l.btAwaitingTitle,
              style: const TextStyle(fontWeight: FontWeight.bold),
            ),
            // 12.4: «получено по Bluetooth» ≠ оплата, до подтверждения
            // сети платёж не считается
            for (final a in _awaiting)
              ListTile(
                dense: true,
                leading: const Icon(Icons.bluetooth),
                title: Text(l.btAwaitingRow(a.amount, a.recipient.shortHash())),
                subtitle: Text(l.btStatus(a.status)),
              ),
            TextButton(
              onPressed: () async {
                await context.read<PlatformService>().publishAwaiting();
                _refresh();
              },
              child: Text(l.btnPublishAwaiting),
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

/// Описание записи истории (л10н, этап 10): строки вынесены из моделей
/// в UI-слой — перевод зависит от локали.
String describeHistory(AppLocalizations l, HistoryEntry h) {
  switch (h.type) {
    case 'REGISTER':
      return l.txRegister;
    case 'VOUCH':
      return l.txVouch;
    case 'MINT':
      return l.txMint(h.period);
    case 'TRANSFER':
      if (h.note.isNotEmpty) return h.note;
      return l.txTransfer(h.amount, h.recipient.shortHash());
    case 'MIGRATE':
      return h.note;
    case 'GUARDIAN_OFFER':
      return h.note;
    default:
      return h.type;
  }
}
