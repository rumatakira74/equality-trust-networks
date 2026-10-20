// Перевод (промпт п.4; 10.2, 8.3, 12.4): комиссия видна ДО отправки,
// отправитель платит сверх суммы; выше порога P — отложенный с
// задержкой D и экраном отмены; Bluetooth-платёж офлайн.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import '../pin_gate.dart';

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
    // деструктивная операция (8.2, этап 9): подтверждение — только явный
    // ввод PIN; тревожный PIN открывает приманку (неотличимо), реальной
    // подписи не будет — мост ответит правдоподобно
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    if (!await confirmPin(context, title: l.transferConfirmTitle)) {
      return;
    }
    setState(() => _busy = true);
    try {
      final built = await platform.buildTransfer(
        _recipient.text.trim(),
        int.parse(_amount.text),
      );
      final id = built['id'] as String;
      // подтверждение уже дано PIN-ом: биометрия здесь — только
      // разблокировка ключа Keystore (не подтверждение операции)
      final signed = await platform.signTx(id, l.keyUnlockTitle);
      if (!signed) {
        await platform.discardTx(id);
        setState(() => _result = l.signNotConfirmed);
        return;
      }
      final res = await platform.submitTx(id);
      setState(() => _result = res.accepted
          ? (_willBeDelayed
              ? l.transferDelayed(_delayHours)
              : l.transferSent)
          : l.rejectedWithCode(res.code));
      await app.refreshWallet();
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Bluetooth-платёж (12.4): подписанная транзакция передаётся
  /// напрямую получателю; «получено по Bluetooth» ≠ оплата.
  Future<void> _sendBt() async {
    // деструктивная операция (8.2, этап 9): подтверждение — только PIN
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    if (!await confirmPin(context, title: l.btConfirmTitle)) {
      return;
    }
    setState(() => _busy = true);
    try {
      final built = await platform.buildTransfer(
        _recipient.text.trim(),
        int.parse(_amount.text),
      );
      final id = built['id'] as String;
      // подтверждение уже дано PIN-ом: биометрия — разблокировка ключа
      final signed = await platform.signTx(id, l.keyUnlockTitle);
      if (!signed) {
        await platform.discardTx(id);
        setState(() => _result = l.signNotConfirmed);
        return;
      }
      final device = _btDevices.firstOrNull;
      if (device == null) {
        setState(() => _result = l.btNoDevices);
        return;
      }
      final r = await platform.sendBluetooth(device, id);
      final ok = r['ok'] as bool? ?? false;
      setState(() => _result = ok
          ? l.btSent
          : l.btError('${r['error']}'));
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final manifest = context.watch<AppState>().manifest;
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.transferTitle)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          SegmentedButton<bool>(
            segments: [
              ButtonSegment(value: false, label: Text(l.segNetwork)),
              ButtonSegment(value: true, label: Text(l.segBluetooth)),
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
            decoration: InputDecoration(
              labelText: l.recipientLabel,
              hintText: l.recipientHint,
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _amount,
            decoration: InputDecoration(
              labelText: l.amountLabel,
              helperText: _fee > 0 ? l.feeHelper(_fee) : null,
            ),
            keyboardType: TextInputType.number,
            onChanged: (_) => _quote(),
          ),
          if (_willBeDelayed)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(
                l.delayedWarn(_delayHours),
                style: const TextStyle(color: Colors.orange),
              ),
            ),
          if (_btMode)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(
                l.btModeHint,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy ? null : (_btMode ? _sendBt : _send),
            child: Text(_btMode ? l.btnSendBt : l.btnSend),
          ),
          if (_result != null) ...[
            const SizedBox(height: 16),
            Text(_result!, textAlign: TextAlign.center),
          ],
          if (manifest != null && manifest.transferMoratoriumDays > 0)
            Padding(
              padding: const EdgeInsets.only(top: 16),
              child: Text(
                l.moratoriumNote(manifest.transferMoratoriumDays),
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
        ],
      ),
    );
  }
}
