// Реестр (промпт п.5; 9.1, 7.4, 7.5): публичный список участников;
// карточка участника — публичная история, возраст, статус, глубина
// графа поручительств, статистика поручителя; метка flagged
// показывается, если есть; автоматических вердиктов нет.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../wallet/wallet_screen.dart' show describeHistory;

class RegistryScreen extends StatefulWidget {
  const RegistryScreen({super.key});

  @override
  State<RegistryScreen> createState() => _RegistryScreenState();
}

class _RegistryScreenState extends State<RegistryScreen> {
  List<RegistryRow> _rows = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    try {
      final rows = await context.read<PlatformService>().registry();
      if (mounted) {
        setState(() {
          _rows = rows;
          _loading = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(
        title: Text(l.registryTitle),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView.builder(
              itemCount: _rows.length,
              itemBuilder: (_, i) {
                final row = _rows[i];
                return ListTile(
                  leading: Icon(
                    row.flagged ? Icons.flag : Icons.circle,
                    color: row.flagged ? Colors.orange : Colors.green,
                    size: row.flagged ? 24 : 12,
                  ),
                  title: Text(row.address.shortHash()),
                  subtitle: Text(l.registryRowSub(row.status, row.ageDays)),
                  onTap: () => Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => MemberScreen(address: row.address),
                    ),
                  ),
                );
              },
            ),
    );
  }
}

/// Карточка участника: публичная история, возраст, статус, глубина
/// графа, статистика поручителя (7.4). Решает принимающий — вердиктов нет.
class MemberScreen extends StatefulWidget {
  final String address;
  const MemberScreen({super.key, required this.address});

  @override
  State<MemberScreen> createState() => _MemberScreenState();
}

class _MemberScreenState extends State<MemberScreen> {
  WalletInfo? _info;
  List<HistoryEntry> _history = [];
  SlotsInfo? _slots;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final platform = context.read<PlatformService>();
    final info = await platform.walletInfo(addrHex: widget.address);
    final history = await platform.history(addrHex: widget.address, limit: 30);
    final slots = await platform.slots(addrHex: widget.address);
    if (mounted) {
      setState(() {
        _info = info;
        _history = history;
        _slots = slots;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final info = _info;
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(widget.address.shortHash())),
      body: info == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Text(widget.address,
                    style: Theme.of(context).textTheme.bodySmall),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  children: [
                    Chip(label: Text(info.status)),
                    Chip(label: Text(l.chipAge(info.ageDays))),
                    if (info.seed) Chip(label: Text(l.chipSeed)),
                    if (info.flagged)
                      const Chip(
                        label: Text('flagged'),
                        backgroundColor: Colors.orange,
                      ),
                    if (info.publicGuardian)
                      Chip(label: Text(l.chipPublicGuardian)),
                  ],
                ),
                const SizedBox(height: 8),
                // глубина графа поручительств (7.4): -1 — не связан
                Text(info.vouchDepth < 0
                    ? l.vouchDepthUnlinked
                    : l.vouchDepthLabel(info.vouchDepth)),
                Text(l.rewardLine(info.completedMintPeriods)),
                if (_slots != null && _slots!.exists)
                  Text(l.asVoucher(_slots!.occupied.length, _slots!.free)),
                const Divider(),
                Text(l.publicHistory),
                for (final h in _history)
                  ListTile(
                    dense: true,
                    leading: const Icon(Icons.circle, size: 8),
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
}
