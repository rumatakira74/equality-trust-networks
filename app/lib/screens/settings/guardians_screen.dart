// Смена личных гардианов (11, решение владельца 21.10.2026): текущий
// список, перевыбор (пары «гардиан + подпись» над хешем смены, согласие
// передаётся вручную — как approve миграции), отсчёт окна 48 ч, отмена
// в окне, индикация остатка лимита («использовано N из 150»).
// Публикация и отмена — только явный ввод PIN (правило этапа 9);
// тревожный PIN открывает режим-приманку: операции «смены» затрагивают
// только данные приманки (платформенный слой не подписывает и не
// публикует), реальный список гардианов не меняется.
import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import '../../widgets/hex_chunk_field.dart';
import '../pin_gate.dart';

/// Лимит смен — константа протокола (не параметр сети), дублируется
/// только для индикации; источник истины — ядро (E_GUARDIAN_CHANGE_LIMIT).
const int kGuardianChangeLimit = 150;

class GuardiansScreen extends StatefulWidget {
  const GuardiansScreen({super.key});

  @override
  State<GuardiansScreen> createState() => _GuardiansScreenState();
}

class _GuardiansScreenState extends State<GuardiansScreen> {
  List<PendingGuardianUpdate> _pending = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final platform = context.read<PlatformService>();
    final app = context.read<AppState>();
    try {
      final pending = await platform.pendingGuardianUpdates();
      if (mounted) setState(() => _pending = pending);
    } catch (_) {}
    await app.refreshWallet();
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final wallet = app.wallet;
    final l = AppLocalizations.of(context)!;
    final active =
        _pending.where((u) => u.active).toList(growable: false);
    final history = _pending.where((u) => !u.active).toList(growable: false);
    return Scaffold(
      appBar: AppBar(
        title: Text(l.guardiansTitle),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(l.guardiansIntro),
          const SizedBox(height: 12),
          // остаток лимита: использовано N из 150
          Text(
            l.guardiansLimitUsed(
              wallet?.guardianChanges ?? 0,
              kGuardianChangeLimit,
            ),
            style: Theme.of(context).textTheme.titleSmall,
          ),
          const SizedBox(height:12),
          Text(l.guardiansCurrent,
              style: Theme.of(context).textTheme.titleMedium),
          if (wallet == null || wallet.personalGuardians.isEmpty)
            Text(l.guardiansNone)
          else
            for (final g in wallet.personalGuardians)
              ListTile(
                dense: true,
                leading: const Icon(Icons.how_to_reg),
                title: Text(g.shortHash()),
              ),
          const SizedBox(height: 12),
          if (active.isNotEmpty) ...[
            for (final u in active)
              Card(
                child: ListTile(
                  leading: const Icon(Icons.schedule),
                  title: Text(l.reelectTitle),
                  subtitle: Text(l.guardiansPendingSub(
                    (u.remainingSec / 3600).ceil(),
                  )),
                  // отмена — только явный ввод PIN (этап 9)
                  trailing: OutlinedButton(
                    onPressed: () => _cancel(u),
                    child: Text(l.guardiansCancelBtn),
                  ),
                ),
              ),
            const SizedBox(height: 12),
          ],
          for (final u in history)
            ListTile(
              dense: true,
              leading: Icon(
                u.canceled ? Icons.block : (u.expired ? Icons.timer_off : Icons.check),
              ),
              title: Text(u.canceled
                  ? l.guardiansStatusCanceled
                  : (u.expired
                      ? l.guardiansStatusExpired
                      : l.guardiansStatusEffective)),
            ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: active.isEmpty
                ? () => Navigator.of(context).push(MaterialPageRoute(
                      builder: (_) => const ReelectGuardiansScreen(),
                    ))
                : null,
            icon: const Icon(Icons.published_with_changes),
            label: Text(l.guardiansReelect),
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: () => Navigator.of(context).push(MaterialPageRoute(
                  builder: (_) => const GuardianConsentScreen(),
                )),
            icon: const Icon(Icons.how_to_reg),
            label: Text(l.consentTitle),
          ),
        ],
      ),
    );
  }

  Future<void> _cancel(PendingGuardianUpdate u) async {
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    // отмена — деструктивная операция: только явный ввод PIN (этап 9);
    // тревожный PIN неотличимо «проходит», публикации нет (8.2)
    if (!await confirmPin(context, title: l.cancelChangeConfirmTitle)) {
      return;
    }
    try {
      final res = await app.signAndSubmit(
        () => platform.buildUpdateGuardiansCancel(u.proposeSeq),
        l.keyUnlockTitle,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(
            res.accepted ? l.changeCanceled : l.rejectedWithCode(res.code)),
      ));
      _refresh();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(l.errorWith('$e'))));
      }
    }
  }
}

/// Перевыбор: ввод нового списка (≤ 3), хеш смены для передачи гардианам,
/// ввод пар «гардиан + подпись», публикация (PIN).
class ReelectGuardiansScreen extends StatefulWidget {
  const ReelectGuardiansScreen({super.key});

  @override
  State<ReelectGuardiansScreen> createState() => _ReelectGuardiansScreenState();
}

class _ReelectGuardiansScreenState extends State<ReelectGuardiansScreen> {
  final _newGuardians = TextEditingController();
  String _guardianInput = '';
  String _sigInput = '';
  final List<Map<String, String>> _approvals = [];
  int _nonce = 1;
  String? _hash;
  String? _result;

  @override
  void initState() {
    super.initState();
    _nonce = Random().nextInt(1 << 30) + 1;
  }

  List<String> get _guardiansList => _newGuardians.text
      .split(RegExp(r'[\s,;]+'))
      .map((s) => s.trim())
      .where((s) => s.isNotEmpty)
      .toList();

  Future<void> _updateHash() async {
    final platform = context.read<PlatformService>();
    final gs = _guardiansList;
    if (gs.isEmpty || gs.length > 3) return;
    try {
      final h = await platform.guardianChangeHash(gs, _nonce);
      if (mounted) setState(() => _hash = h);
    } catch (e) {
      if (mounted) {
        setState(() => _result = AppLocalizations.of(context)!.errorWith('$e'));
      }
    }
  }

  Future<void> _addApproval() async {
    // подпись собирается вне приложения (гардиан подписывает у себя);
    // здесь — ввод пары «адрес гардиана + hex подписи» (чанки по 20,
    // PENDING №6); контроль длины/символов — до добавления пары
    if (!HexChunkField.isValid(_guardianInput, 64) ||
        !HexChunkField.isValid(_sigInput, 0)) {
      return;
    }
    setState(() {
      _approvals.add({
        'guardian': _guardianInput,
        'sig': _sigInput,
      });
      _guardianInput = '';
      _sigInput = '';
    });
  }

  Future<void> _publish() async {
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    // публикация — деструктивная операция: только явный ввод PIN;
    // тревожный PIN открывает приманку, реальный список не меняется
    if (!await confirmPin(context, title: l.changeConfirmTitle)) {
      return;
    }
    try {
      final res = await app.signAndSubmit(
        () => platform.buildUpdateGuardiansPropose(
          _guardiansList,
          _approvals,
          _nonce,
        ),
        l.keyUnlockTitle,
      );
      setState(() => _result = res.accepted
          ? l.changePublished
          : l.rejectedWithCode(res.code));
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    final gs = _guardiansList;
    final enough = gs.isNotEmpty &&
        gs.length <= 3 &&
        _approvals.length >= gs.length;
    return Scaffold(
      appBar: AppBar(title: Text(l.reelectTitle)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(l.reelectIntro),
          const SizedBox(height: 12),
          TextField(
            controller: _newGuardians,
            decoration: InputDecoration(labelText: l.newGuardiansLabel),
            maxLines: 3,
            onChanged: (_) => _updateHash(),
          ),
          const SizedBox(height: 8),
          // хеш смены виден сразу: гардиан подписывает именно его
          if (_hash != null) ...[
            Text(l.changeHashToCopy),
            Row(
              children: [
                Expanded(
                  child: SelectableText(
                    _hash!,
                    style: const TextStyle(fontSize: 11),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.copy, size: 18),
                  tooltip: l.changeHashToCopy,
                  onPressed: () {
                    Clipboard.setData(ClipboardData(text: _hash!));
                    ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text(l.changeHashToCopy)));
                  },
                ),
              ],
            ),
            const SizedBox(height: 8),
          ],
          Text(l.guardiansApprovals(_approvals.length, gs.length)),
          for (final a in _approvals)
            ListTile(
              dense: true,
              leading: const Icon(Icons.how_to_reg),
              title: Text((a['guardian'] ?? '').shortHash()),
            ),
          HexChunkField(
            label: l.guardianAddrLabel,
            chunkCount: 4,
            expectedLength: 64,
            onChanged: (v) => setState(() => _guardianInput = v),
          ),
          const SizedBox(height: 8),
          HexChunkField(
            label: l.guardianSigLabel,
            chunkCount: 8,
            onChanged: (v) => setState(() => _sigInput = v),
          ),
          const SizedBox(height: 8),
          OutlinedButton(
            onPressed: HexChunkField.isValid(_guardianInput, 64) &&
                    HexChunkField.isValid(_sigInput, 0)
                ? _addApproval
                : null,
            child: Text(l.btnAddSig),
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: enough ? _publish : null,
            child: Text(l.btnPublishChange),
          ),
          if (_result != null) ...[
            const SizedBox(height: 12),
            Text(_result!, textAlign: TextAlign.center),
          ],
        ],
      ),
    );
  }
}

/// Экран согласия гардиана: адрес подопечного, хеш смены, биометрия
/// (как ApproveMigrationScreen). Подпись передаётся вручную.
/// Длинные hex-строки — чанками по 20 символов (PENDING №6): IME
/// эмуляторов теряет символы при вставке в одно поле.
/// Таймаут биометрии не вешает UI (PENDING №3): платформенный слой
/// может не ответить (BiometricPrompt на эмуляторах) — ожидание
/// ограничено, экран возвращается в рабочее состояние.
class GuardianConsentScreen extends StatefulWidget {
  const GuardianConsentScreen({
    super.key,
    this.signTimeout = const Duration(seconds: 60),
  });

  /// Таймаут ожидания ответа биометрии; в тестах — короче.
  final Duration signTimeout;

  @override
  State<GuardianConsentScreen> createState() => _GuardianConsentScreenState();
}

class _GuardianConsentScreenState extends State<GuardianConsentScreen> {
  String _wardAddress = '';
  String _changeHash = '';
  String? _sig;
  String? _error;
  bool _busy = false;

  Future<void> _sign() async {
    if (_busy) return;
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final r = await platform
          .signGuardianChange(_changeHash, l.consentSignTitle)
          .timeout(widget.signTimeout);
      final sig = r['sig'] as String? ?? '';
      if (!mounted) return;
      setState(() {
        _sig = sig.isEmpty ? null : sig;
        _error = sig.isEmpty ? l.signNotConfirmed : null;
      });
    } catch (e) {
      // таймаут платформенного слоя: UI не висит, экран рабочий
      if (!mounted) return;
      setState(() {
        _error = e is TimeoutException ? l.biometricTimeout : l.errorWith('$e');
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    final valid = HexChunkField.isValid(_wardAddress, 64) &&
        HexChunkField.isValid(_changeHash, 64);
    return Scaffold(
      appBar: AppBar(title: Text(l.consentTitle)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(l.consentIntro),
          const SizedBox(height: 12),
          HexChunkField(
            label: l.wardAddressLabel,
            chunkCount: 4,
            expectedLength: 64,
            onChanged: (v) => setState(() => _wardAddress = v),
          ),
          const SizedBox(height: 8),
          HexChunkField(
            label: l.changeHashLabel,
            chunkCount: 4,
            expectedLength: 64,
            onChanged: (v) => setState(() => _changeHash = v),
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _busy || !valid ? null : _sign,
            child: _busy
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : Text(l.btnSignBiometric),
          ),
          if (_sig != null) ...[
            const SizedBox(height: 12),
            Text(l.sigToCopy),
            SelectableText(_sig!, style: const TextStyle(fontSize: 11)),
          ],
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(_error!, style: const TextStyle(color: Colors.red)),
            ),
        ],
      ),
    );
  }
}
