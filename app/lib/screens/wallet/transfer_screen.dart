// Перевод (промпт п.4; 10.2, 8.3, 12.4): комиссия видна ДО отправки,
// отправитель платит сверх суммы; выше порога P — отложенный с
// задержкой D и экраном отмены; Bluetooth-платёж офлайн.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../services/platform_service.dart';
import '../../state/app_state.dart';

class TransferScreen extends StatefulWidget {
  const TransferScreen({super.key});

  @override
  State<TransferScreen> createState() => _TransferScreenState();
}

class _TransferScreenState extends State<TransferScreen> {
  final _recipient = TextEditingController();
  final _amount = TextEditingController();
  int _fee = 0;
  bool _willBeDelayed = false;
  int _delayHours = 48;
  String? _result;
  bool _busy = false;
  bool _btMode = false;
  List<String> _btDevices = [];

  @override
  void dispose() {
    _recipient.dispose();
    _amount.dispose();
    super.dispose();
  }

  Future<void> _quote() async {
    final amount = int.tryParse(_amount.text) ?? 0;
    if (amount <= 0) {
      setState(() => _fee = 0);
      return;
    }
    final platform = context.read<PlatformService>();
    final fee = await platform.feeQuote(amount);
    if (!mounted) return;
    final wallet = context.read<AppState>().wallet;
    final manifest = context.read<AppState>().manifest;
    setState(() {
      _fee = fee;
      // 8.3: выше P — отложенный; правило действует только в сетях
      // с передаваемыми единицами (мост учитывает transfer_mode/sandbox)
      _willBeDelayed = (wallet?.thresholdP ?? 0) < amount &&
          (manifest?.transfersDelayed ?? false);
      _delayHours = manifest?.transferDelayHours ?? 48;
    });
  }

  Future<void> _send() async {
    setState(() => _busy = true);
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    try {
      final built = await platform.buildTransfer(
        _recipient.text.trim(),
        int.parse(_amount.text),
      );
      final id = built['id'] as String;
      final signed = await platform.signTx(id, 'Подтвердите перевод');
      if (!signed) {
        await platform.discardTx(id);
        setState(() => _result = 'Подпись не подтверждена');
        return;
      }
      final res = await platform.submitTx(id);
      setState(() => _result = res.accepted
          ? (_willBeDelayed
              ? 'Перевод отложен: исполнение через $_delayHours ч, '
                'отменить можно в течение задержки'
              : 'Перевод отправлен')
          : 'Отклонено: ${res.code}');
      await app.refreshWallet();
    } catch (e) {
      setState(() => _result = 'Ошибка: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Bluetooth-платёж (12.4): подписанная транзакция передаётся
  /// напрямую получателю; «получено по Bluetooth» ≠ оплата.
  Future<void> _sendBt() async {
    setState(() => _busy = true);
    final platform = context.read<PlatformService>();
    try {
      final built = await platform.buildTransfer(
        _recipient.text.trim(),
        int.parse(_amount.text),
      );
      final id = built['id'] as String;
      final signed = await platform.signTx(id, 'Подтвердите Bluetooth-платёж');
      if (!signed) {
        await platform.discardTx(id);
        setState(() => _result = 'Подпись не подтверждена');
        return;
      }
      final device = _btDevices.firstOrNull;
      if (device == null) {
        setState(() => _result = 'Нет сопряжённых устройств');
        return;
      }
      final r = await platform.sendBluetooth(device, id);
      final ok = r['ok'] as bool? ?? false;
      setState(() => _result = ok
          ? 'Передано по Bluetooth. Получатель обязан дождаться '
            'подтверждения сети — до публикации это не оплата'
          : 'Bluetooth: ${r['error']}');
    } catch (e) {
      setState(() => _result = 'Ошибка: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final manifest = context.watch<AppState>().manifest;
    return Scaffold(
      appBar: AppBar(title: const Text('Перевод')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          SegmentedButton<bool>(
            segments: const [
              ButtonSegment(value: false, label: Text('Через сеть')),
              ButtonSegment(value: true, label: Text('Bluetooth')),
            ],
            selected: {_btMode},
            onSelectionChanged: (s) async {
              setState(() => _btMode = s.first);
              if (s.first) {
                final platform = context.read<PlatformService>();
                if (!await platform.btPermissions()) {
                  await platform.requestBtPermissions();
                }
                final devices = await platform.btBondedDevices();
                if (mounted) setState(() => _btDevices = devices);
              }
            },
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _recipient,
            decoration: const InputDecoration(
              labelText: 'Адрес получателя (hex)',
              hintText: '64 hex-символа',
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _amount,
            decoration: InputDecoration(
              labelText: 'Сумма (получатель получит ровно столько)',
              helperText: _fee > 0
                  ? 'Комиссия: $_fee (сверх суммы, сжигается) — платит отправитель'
                  : null,
            ),
            keyboardType: TextInputType.number,
            onChanged: (_) => _quote(),
          ),
          if (_willBeDelayed)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(
                'Сумма выше порога P: перевод будет отложен. Исполнение '
                'через $_delayHours ч; отменить можно в течение задержки '
                '(экран «Отложенные»).',
                style: const TextStyle(color: Colors.orange),
              ),
            ),
          if (_btMode)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(
                'Bluetooth офлайн: получатель увидит «получено по Bluetooth — '
                'ожидает публикации». До подтверждения сети это не оплата '
                '(12.4).',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy ? null : (_btMode ? _sendBt : _send),
            child: Text(_btMode ? 'Передать по Bluetooth' : 'Отправить'),
          ),
          if (_result != null) ...[
            const SizedBox(height: 16),
            Text(_result!, textAlign: TextAlign.center),
          ],
          if (manifest != null && manifest.transferMoratoriumDays > 0)
            Padding(
              padding: const EdgeInsets.only(top: 16),
              child: Text(
                'Мораторий: переводы доступны через '
                '${manifest.transferMoratoriumDays} дн. после регистрации.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
        ],
      ),
    );
  }
}
