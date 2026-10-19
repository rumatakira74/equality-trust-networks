// Поручительства (промпт п.3; 7.3–7.4): статус кандидата — сколько
// vouch набрано до K; «поручиться за новичка» при личной встрече
// через QR или Bluetooth (присутствие НЕ считается доказанным —
// предупреждение в UI); экран слотов поручителя: занятые/свободные,
// освобождение (3 выпуска без evidence / 90 дней), статистика
// (за скольких поручился, сколько активны).
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';

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
    return Scaffold(
      appBar: AppBar(title: const Text('Поручительства')),
      body: Column(
        children: [
          SegmentedButton<int>(
            segments: const [
              ButtonSegment(value: 0, label: Text('Мой статус')),
              ButtonSegment(value: 1, label: Text('Поручиться')),
              ButtonSegment(value: 2, label: Text('Мои слоты')),
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
    final received = _progress['received'] ?? 0;
    final k = _progress['k'] ?? 3;
    final active = _wallet?.active ?? false;
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text(
          active ? 'Кошелёк активен' : 'Поручительств: $received из $k',
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 12),
        LinearProgressIndicator(value: k == 0 ? 0 : received / k),
        const SizedBox(height: 12),
        if (!active)
          const Text(
            'Найдите K поручителей при личной встрече: покажите им QR-код '
            'своего адреса на вкладке «Поручиться» (кнопка «Мой QR»).',
          ),
        if (_wallet?.flagged == true)
          const Text(
            'Метка flagged: поручиться за вас нельзя (vouch от '
            'flagged-кошелька запрещён, решение №14)',
            style: TextStyle(color: Colors.orange),
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
    try {
      final res = await app.signAndSubmit(
        () => platform.buildVouch(_candidate.text.trim()),
        'Подтвердите поручительство',
      );
      setState(() => _result = res.accepted
          ? 'Поручительство опубликовано'
          : 'Отклонено: ${res.code}');
    } catch (e) {
      setState(() => _result = 'Ошибка: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _receiveByBt() async {
    setState(() => _result = 'Ожидание адреса по Bluetooth (60 с)…');
    final platform = context.read<PlatformService>();
    try {
      if (!await platform.btPermissions()) {
        await platform.requestBtPermissions();
      }
      final addr = await platform.btReceiveAddress();
      setState(() {
        _candidate.text = addr;
        _result = 'Адрес получен по Bluetooth';
      });
    } catch (e) {
      setState(() => _result = 'Ошибка: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final wallet = app.wallet;
    final canVouch = wallet?.active == true && !(wallet?.flagged ?? false);
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        // предупреждение обязательно (7.4): присутствие не доказано
        Card(
          color: Colors.amber.withValues(alpha: 0.15),
          child: const Padding(
            padding: EdgeInsets.all(12),
            child: Text(
              'Поручительство при личной встрече (QR или Bluetooth) НЕ '
              'доказывает присутствие человека. Вы отвечаете за выбор: '
              'слот ограничен, при evidence против поручаемого вы '
              'теряете вознаграждение (не деньги).',
            ),
          ),
        ),
        const SizedBox(height: 12),
        if (!canVouch)
          const Text(
            'Поручителем может быть только active-кошелёк с возрастом '
            '≥ 3 мес. и ≥ 2 выполненными периодами выпуска (7.4).',
          )
        else ...[
          TextField(
            controller: _candidate,
            decoration: const InputDecoration(
              labelText: 'Адрес кандидата (hex)',
              hintText: '64 hex-символа — из QR или Bluetooth',
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: _receiveByBt,
                  icon: const Icon(Icons.bluetooth),
                  label: const Text('Получить по BT'),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => setState(() => _showQr = !_showQr),
                  icon: const Icon(Icons.qr_code),
                  label: Text(_showQr ? 'Скрыть мой QR' : 'Мой QR'),
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
                  const Text('Мой адрес — покажите поручителю'),
                ],
              ),
            ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy || _candidate.text.trim().isEmpty ? null : _vouch,
            child: const Text('Поручиться'),
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
    if (s == null) {
      return const Center(child: CircularProgressIndicator());
    }
    final activeCount = s.occupied.where((x) => x.candidateActive).length;
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text('Слоты: ${s.occupied.length} занято из ${s.capacity}'),
        const SizedBox(height: 4),
        Text(
          'Свободно: ${s.free}. Статистика поручителя: поручился за '
          '${s.occupied.length}, активны $activeCount (публичная репутация, 7.4)',
        ),
        const SizedBox(height: 12),
        const Divider(),
        if (s.occupied.isEmpty) const Text('Слоты свободны'),
        for (final slot in s.occupied)
          ListTile(
            leading: Icon(
              slot.candidateActive ? Icons.verified : Icons.hourglass_top,
              color: slot.candidateActive ? Colors.green : Colors.orange,
            ),
            title: Text(slot.candidate.shortHash()),
            subtitle: Text(slot.candidateActive
                ? 'кандидат активен: слот освободится после '
                  '${manifest?.slotFreeMintPeriods ?? 3} периодов выпуска '
                  'без evidence'
                : 'кандидат не активен: слот освободится через '
                  '${manifest?.slotFreeDays ?? 90} дн.'),
          ),
      ],
    );
  }
}
