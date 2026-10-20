// Поручительства (промпт п.3; 7.3–7.4): статус кандидата — сколько
// vouch набрано до K; «поручиться за новичка» при личной встрече
// через QR или Bluetooth (присутствие НЕ считается доказанным —
// предупреждение в UI); экран слотов поручителя: занятые/свободные,
// освобождение (3 выпуска без evidence / 90 дней), статистика
// (за скольких поручился, сколько активны).
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';

class VouchingScreen extends StatefulWidget {
  const VouchingScreen({super.key});

  @override
  State<VouchingScreen> createState() => _VouchingScreenState();
}

class _VouchingScreenState extends State<VouchingScreen> {
  int _tab = 0;

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.vouchingTitle)),
      body: Column(
        children: [
          SegmentedButton<int>(
            segments: [
              ButtonSegment(value: 0, label: Text(l.segMyStatus)),
              ButtonSegment(value: 1, label: Text(l.segVouch)),
              ButtonSegment(value: 2, label: Text(l.segMySlots)),
            ],
            selected: {_tab},
            onSelectionChanged: (s) => setState(() => _tab = s.first),
          ),
          Expanded(
            child: switch (_tab) {
              0 => const _CandidateStatus(),
              1 => const _VouchForNewcomer(),
              _ => const _MySlots(),
            },
          ),
        ],
      ),
    );
  }
}

/// Статус кандидата: сколько vouch набрано до K (7.4).
class _CandidateStatus extends StatefulWidget {
  const _CandidateStatus();

  @override
  State<_CandidateStatus> createState() => _CandidateStatusState();
}

class _CandidateStatusState extends State<_CandidateStatus> {
  Map<String, int> _progress = {};
  WalletInfo? _wallet;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final platform = context.read<PlatformService>();
    final progress = await platform.vouchProgress();
    final wallet = await platform.walletInfo();
    if (mounted) {
      setState(() {
        _progress = progress;
        _wallet = wallet;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    final received = _progress['received'] ?? 0;
    final k = _progress['k'] ?? 3;
    final active = _wallet?.active ?? false;
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text(
          active ? l.walletActive : l.vouchProgress(received, k),
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 12),
        LinearProgressIndicator(value: k == 0 ? 0 : received / k),
        const SizedBox(height: 12),
        if (!active) Text(l.vouchFindHelpers),
        if (_wallet?.flagged == true)
          Text(
            l.vouchFlagged,
            style: const TextStyle(color: Colors.orange),
          ),
      ],
    );
  }
}

/// Поручиться за новичка (7.4): адрес кандидата — QR (показ кандидатом,
/// ввод вручную на стороне поручителя) или Bluetooth при встрече.
/// Присутствие не доказано — предупреждение обязательно.
class _VouchForNewcomer extends StatefulWidget {
  const _VouchForNewcomer();

  @override
  State<_VouchForNewcomer> createState() => _VouchForNewcomerState();
}

class _VouchForNewcomerState extends State<_VouchForNewcomer> {
  final _candidate = TextEditingController();
  bool _showQr = false;
  bool _busy = false;
  String? _result;

  Future<void> _vouch() async {
    setState(() => _busy = true);
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    try {
      final res = await app.signAndSubmit(
        () => platform.buildVouch(_candidate.text.trim()),
        l.vouchConfirmTitle,
      );
      setState(() => _result = res.accepted
          ? l.vouchPublished
          : l.rejectedWithCode(res.code));
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _receiveByBt() async {
    final l = AppLocalizations.of(context)!;
    setState(() => _result = l.btWaiting);
    final platform = context.read<PlatformService>();
    try {
      if (!await platform.btPermissions()) {
        await platform.requestBtPermissions();
      }
      final addr = await platform.btReceiveAddress();
      setState(() {
        _candidate.text = addr;
        _result = l.btAddrReceived;
      });
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final l = AppLocalizations.of(context)!;
    final wallet = app.wallet;
    final canVouch = wallet?.active == true && !(wallet?.flagged ?? false);
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        // предупреждение обязательно (7.4): присутствие не доказано
        Card(
          color: Colors.amber.withValues(alpha: 0.15),
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Text(l.vouchWarning),
          ),
        ),
        const SizedBox(height: 12),
        if (!canVouch)
          Text(l.vouchOnlyActive)
        else ...[
          TextField(
            controller: _candidate,
            decoration: InputDecoration(
              labelText: l.candidateLabel,
              hintText: l.candidateHint,
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: _receiveByBt,
                  icon: const Icon(Icons.bluetooth),
                  label: Text(l.btnBtReceive),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => setState(() => _showQr = !_showQr),
                  icon: const Icon(Icons.qr_code),
                  label: Text(_showQr ? l.btnHideQr : l.btnMyQr),
                ),
              ),
            ],
          ),
          if (_showQr && app.address.isNotEmpty)
            Center(
              child: Column(
                children: [
                  const SizedBox(height: 12),
                  QrImageView(
                    data: app.address,
                    size: 180,
                    backgroundColor: Colors.white,
                  ),
                  Text(l.myQrHint),
                ],
              ),
            ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy || _candidate.text.trim().isEmpty ? null : _vouch,
            child: Text(l.btnVouch),
          ),
        ],
        if (_result != null) ...[
          const SizedBox(height: 12),
          Text(_result!, textAlign: TextAlign.center),
        ],
      ],
    );
  }
}

/// Слоты поручителя (7.4): занятые/свободные, освобождение
/// (3 периода выпуска без evidence / 90 дней без активации),
/// статистика: за скольких поручился, сколько активны.
class _MySlots extends StatefulWidget {
  const _MySlots();

  @override
  State<_MySlots> createState() => _MySlotsState();
}

class _MySlotsState extends State<_MySlots> {
  SlotsInfo? _slots;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final s = await context.read<PlatformService>().slots();
    if (mounted) setState(() => _slots = s);
  }

  @override
  Widget build(BuildContext context) {
    final s = _slots;
    final manifest = context.watch<AppState>().manifest;
    final l = AppLocalizations.of(context)!;
    if (s == null) {
      return const Center(child: CircularProgressIndicator());
    }
    final activeCount = s.occupied.where((x) => x.candidateActive).length;
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text(l.slotsSummary(s.occupied.length, s.capacity)),
        const SizedBox(height: 4),
        Text(
          l.slotsFreeStats(s.free, s.occupied.length, activeCount),
        ),
        const SizedBox(height: 12),
        const Divider(),
        if (s.occupied.isEmpty) Text(l.slotsEmpty),
        for (final slot in s.occupied)
          ListTile(
            leading: Icon(
              slot.candidateActive ? Icons.verified : Icons.hourglass_top,
              color: slot.candidateActive ? Colors.green : Colors.orange,
            ),
            title: Text(slot.candidate.shortHash()),
            subtitle: Text(slot.candidateActive
                ? l.slotActiveSub(manifest?.slotFreeMintPeriods ?? 3)
                : l.slotInactiveSub(manifest?.slotFreeDays ?? 90)),
          ),
      ],
    );
  }
}
