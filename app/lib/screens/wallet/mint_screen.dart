// Выпуск с челленджем (промпт п.4; 7.6, 10.1): усиленный PoW в
// рандомном слоте внутри месяца, окно ±48 ч; точный момент слота в
// UI не показывается (PENDING, отчёт этапа 7). Пропуск — перенос в
// пределах accrual_window, средства не блокируются.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import '../pin_gate.dart';

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
    // деструктивная операция (8.2, этап 9): подтверждение — только явный
    // ввод PIN; тревожный PIN открывает приманку (неотличимо)
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    if (!await confirmPin(context, title: l.mintConfirmTitle)) {
      return;
    }
    setState(() {
      _busy = true;
      _result = l.mintPowStatus;
    });
    try {
      // период и сумма к заявлению — из узла (M4–M6: только завершённые
      // периоды, amount = эмиссия × diff от base; формула — в ядре)
      final w = _window!;
      if (w.claimPeriod <= 0) {
        setState(() => _result = l.mintNoPeriods);
        return;
      }
      // подтверждение уже дано PIN-ом: биометрия — разблокировка ключа
      final res = await app.signAndSubmit(
        () => platform.buildMint(w.claimPeriod, w.claimAmount),
        l.keyUnlockTitle,
      );
      setState(() => _result =
          res.accepted ? l.mintPublished : l.rejectedWithCode(res.code));
      await app.refreshWallet();
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final w = _window;
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.mintTitle)),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (w != null) ...[
              Text(
                l.mintPeriod(w.currentPeriod),
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 8),
              Text(
                w.open
                    ? (w.windowHours > 0
                          ? l.mintWindowOpen(w.windowHours)
                          : l.mintWindowOff)
                    : l.mintWindowClosed(w.accrualWindowPeriods),
              ),
              const SizedBox(height: 16),
              Text(l.mintLastClaimed(w.lastClaimedPeriod)),
              Text(l.mintClaimPeriod(
                  w.claimPeriod > 0 ? '${w.claimPeriod}' : '—')),
              Text(l.mintClaimAmount(w.claimAmount)),
              Text(l.mintIssuance(w.issuanceAmount)),
              const SizedBox(height: 24),
              FilledButton(
                onPressed:
                    w.open && w.claimPeriod > 0 && !_busy ? _mint : null,
                child: Text(l.btnMintDo),
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
