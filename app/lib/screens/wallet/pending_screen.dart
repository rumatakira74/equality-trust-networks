// Отложенные переводы (8.3): список с обратным отсчётом до
// исполнения, отмена в течение D (подпись с биометрией), смена
// порога P: повышение вступает через 48 ч, понижение — сразу,
// в пределах P_max.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';

class PendingScreen extends StatefulWidget {
  const PendingScreen({super.key});

  @override
  State<PendingScreen> createState() => _PendingScreenState();
}

class _PendingScreenState extends State<PendingScreen> {
  List<PendingTransfer> _items = [];
  final _newP = TextEditingController();
  String? _result;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final items = await context.read<PlatformService>().pendingTransfers();
    if (mounted) setState(() => _items = items);
  }

  Future<void> _cancel(PendingTransfer p) async {
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    try {
      final res = await app.signAndSubmit(
        () => platform.buildCancel(p.seq),
        l.cancelConfirmTitle,
      );
      setState(() => _result =
          res.accepted ? l.canceledMsg : l.rejectedWithCode(res.code));
      _refresh();
      app.refreshWallet();
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    }
  }

  Future<void> _setThreshold() async {
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    final newP = int.tryParse(_newP.text);
    if (newP == null) {
      setState(() => _result = l.invalidThreshold);
      return;
    }
    try {
      final res = await app.signAndSubmit(
        () => platform.buildSetThreshold(newP),
        l.btnChangeThreshold,
      );
      final wallet = app.wallet;
      final raising = newP > (wallet?.thresholdP ?? 0);
      setState(() => _result = res.accepted
          ? (raising ? l.thresholdRaised : l.thresholdLowered)
          : l.rejectedWithCode(res.code));
      await app.refreshWallet();
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final wallet = app.wallet;
    final manifest = app.manifest;
    final pMax = manifest?.delayedThresholdMax ?? 0;
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.pendingTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(l.thresholdP(wallet?.thresholdP ?? 0)),
                  if (wallet?.pendingThreshold != null &&
                      (wallet?.pendingThreshold ?? 0) > 0)
                    Text(
                      l.thresholdPendingRaise(
                        wallet?.pendingThreshold ?? 0,
                        _fmtTime(wallet?.pendingThresholdNotBefore ?? 0),
                      ),
                    ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _newP,
                    decoration: InputDecoration(
                      labelText: l.newThresholdLabel,
                      helperText: l.newThresholdHelper(pMax),
                    ),
                    keyboardType: TextInputType.number,
                  ),
                  const SizedBox(height: 8),
                  OutlinedButton(
                    onPressed: _setThreshold,
                    child: Text(l.btnChangeThreshold),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 8),
          if (_items.isEmpty) Text(l.pendingEmpty),
          for (final p in _items.where((x) => !x.canceled && !x.executed))
            Card(
              child: ListTile(
                leading: const Icon(Icons.schedule),
                title: Text(l.pendingRow(p.amount, p.recipient.shortHash())),
                subtitle: Text(
                  l.pendingRowSub(_fmt(p.remainingSec), p.fee),
                ),
                trailing: OutlinedButton(
                  onPressed: () => _cancel(p),
                  child: Text(l.btnCancelTransfer),
                ),
              ),
            ),
          if (_result != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(_result!, textAlign: TextAlign.center),
            ),
        ],
      ),
    );
  }

  String _fmt(int sec) {
    if (sec <= 0) return '—';
    final h = sec ~/ 3600;
    final m = (sec % 3600) ~/ 60;
    return AppLocalizations.of(context)!.timeHm(h, m);
  }

  String _fmtTime(int unix) =>
      DateTime.fromMillisecondsSinceEpoch(unix * 1000).toLocal().toString();
}
