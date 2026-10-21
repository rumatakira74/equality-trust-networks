// Миграция (промпт п.6; 11): запуск (личные 2 из 3, публичные 3 из 5,
// пороги комбинируются), отмена живым старым ключом, задержка 14 дней
// с обратным отсчётом; перезапуск через обычную register (старый
// кошелёк остаётся в реестре — предупреждение о потере средств);
// экран выбора публичных гардианов (guardian_offer).
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import '../pin_gate.dart';
import '../recovery/recovery_card_screen.dart';
import '../settings/guardians_screen.dart';

class MigrationScreen extends StatefulWidget {
  const MigrationScreen({super.key});

  @override
  State<MigrationScreen> createState() => _MigrationScreenState();
}

class _MigrationScreenState extends State<MigrationScreen> {
  List<PendingMigration> _pending = [];
  List<GuardianOffer> _offers = [];

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    final platform = context.read<PlatformService>();
    try {
      final pending = await platform.pendingMigrations();
      if (mounted) setState(() => _pending = pending);
    } catch (_) {}
    try {
      final offers = await platform.guardianOffers();
      if (mounted) setState(() => _offers = offers);
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final manifest = app.manifest;
    final l = AppLocalizations.of(context)!;
    final active = _pending.where((m) => !m.canceled && !m.executed).toList();
    return Scaffold(
      appBar: AppBar(
        title: Text(l.migrationTitle),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(l.migrationIntro),
          const SizedBox(height: 12),
          if (active.isNotEmpty) ...[
            for (final m in active)
              Card(
                child: ListTile(
                  leading: const Icon(Icons.schedule),
                  title: Text(l.migrationTo(m.newAddress.shortHash())),
                  subtitle: Text(
                    l.migrationRowSub(
                      (m.remainingSec / 86400).round(),
                      manifest?.migrateDelayDays ?? 14,
                      m.approvals.length,
                    ),
                  ),
                  // отмена живым старым ключом (11)
                  trailing: m.oldAddress == app.address
                      ? OutlinedButton(
                          onPressed: () => _cancel(m),
                          child: Text(l.btnCancelTransfer),
                        )
                      : null,
                ),
              ),
            const SizedBox(height: 12),
          ],
          FilledButton.icon(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const StartMigrationScreen()),
            ),
            icon: const Icon(Icons.add),
            label: Text(l.btnStartMigration),
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const GuardiansScreen()),
            ),
            icon: const Icon(Icons.published_with_changes),
            label: Text(l.guardiansTitle),
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const ApproveMigrationScreen()),
            ),
            icon: const Icon(Icons.how_to_reg),
            label: Text(l.btnApproveMigration),
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: () => Navigator.of(context).push(MaterialPageRoute(
                  builder: (_) => const RecoveryCardScreen(),
                )),
            icon: const Icon(Icons.badge_outlined),
            label: Text(l.settingsRecovery),
          ),
          const SizedBox(height: 24),
          Text(l.publicGuardiansTitle,
              style: Theme.of(context).textTheme.titleMedium),
          if (_offers.isEmpty) Text(l.noGuardians),
          for (final o in _offers)
            ListTile(
              leading: const Icon(Icons.volunteer_activism),
              title: Text(o.address.shortHash()),
              subtitle: Text(l.guardianOfferRow(o.ageDays, o.wards)),
              trailing: o.flagged ? const Text('flagged') : null,
            ),
          const SizedBox(height: 24),
          // перезапуск: обычная register, старый кошелёк остаётся
          Card(
            color: Colors.red.withValues(alpha: 0.08),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l.restartIntro,
                    style: const TextStyle(color: Colors.red),
                  ),
                  const SizedBox(height: 8),
                  OutlinedButton(
                    onPressed: () => _restartWarning(context),
                    child: Text(l.btnRestart),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _cancel(PendingMigration m) async {
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    try {
      final res = await app.signAndSubmit(
        () => platform.buildMigrateCancel(m.newAddress, m.proposeSeq),
        l.cancelMigrationTitle,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(
            res.accepted ? l.migrationCanceled : l.rejectedWithCode(res.code)),
      ));
      _refresh();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(l.errorWith('$e'))));
      }
    }
  }

  Future<void> _restartWarning(BuildContext context) async {
    final l = AppLocalizations.of(context)!;
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l.restartDialogTitle),
        content: Text(l.restartDialogBody),
        actions: [
          TextButton(
              onPressed: () => Navigator.of(ctx).pop(), child: Text(l.btnCancel)),
          FilledButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              // перезапуск — обычная register: онбординг заново
              // (ключ будет перегенерирован на этом устройстве)
              Navigator.of(context).pushReplacement(
                MaterialPageRoute(builder: (_) => const _RestartFlow()),
              );
            },
            child: Text(l.btnUnderstandRestart),
          ),
        ],
      ),
    );
  }
}

/// Перезапуск: повторный онбординг (register с новым ключом).
class _RestartFlow extends StatelessWidget {
  const _RestartFlow();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Text(AppLocalizations.of(context)!.restartFlowNote),
      ),
    );
  }
}

/// Запуск миграции на НОВОМ устройстве: ввод старого адреса, сбор
/// подписей гардианов (личные 2 из 3; публичные 3 из 5; комбинируются),
/// публикация MIGRATE/propose в цепочке нового кошелька.
class StartMigrationScreen extends StatefulWidget {
  const StartMigrationScreen({super.key});

  @override
  State<StartMigrationScreen> createState() => _StartMigrationScreenState();
}

class _StartMigrationScreenState extends State<StartMigrationScreen> {
  final _oldAddress = TextEditingController();
  final List<Map<String, String>> _approvals = [];
  final _guardianInput = TextEditingController();
  final _sigInput = TextEditingController();
  int _nonce = 1;
  String? _result;

  @override
  void initState() {
    super.initState();
    _nonce = Random().nextInt(1 << 30) + 1;
  }

  Future<void> _addApproval() async {
    // подпись собирается вне приложения (гардиан подписывает у себя);
    // здесь — ввод пары «адрес гардиана + hex подписи»
    setState(() {
      _approvals.add({
        'guardian': _guardianInput.text.trim(),
        'sig': _sigInput.text.trim(),
      });
      _guardianInput.clear();
      _sigInput.clear();
    });
  }

  Future<void> _propose() async {
    // деструктивная операция (8.2, этап 9): подтверждение — только явный
    // ввод PIN; тревожный PIN открывает приманку (неотличимо)
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    if (!await confirmPin(context, title: l.migrationConfirmTitle)) {
      return;
    }
    try {
      // подтверждение уже дано PIN-ом: биометрия — разблокировка ключа
      final res = await app.signAndSubmit(
        () => platform.buildMigratePropose(
          _oldAddress.text.trim(),
          _approvals,
          _nonce,
        ),
        l.keyUnlockTitle,
      );
      setState(() => _result = res.accepted
          ? l.migrationStarted(app.manifest?.migrateDelayDays ?? 14)
          : l.rejectedWithCode(res.code));
    } catch (e) {
      setState(() => _result = l.errorWith('$e'));
    }
  }

  @override
  Widget build(BuildContext context) {
    final manifest = context.watch<AppState>().manifest;
    final l = AppLocalizations.of(context)!;
    final needPersonal = manifest?.guardianPersonalThreshold ?? 2;
    final needPublic = manifest?.guardianPublicThreshold ?? 3;
    final enough = _approvals.length >= needPersonal;
    return Scaffold(
      appBar: AppBar(title: Text(l.startMigrationTitle)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(l.startMigrationIntro),
          const SizedBox(height: 12),
          TextField(
            controller: _oldAddress,
            decoration: InputDecoration(labelText: l.oldAddressLabel),
          ),
          const SizedBox(height: 8),
          // nonce виден сразу: гардиан подписывает MigrateStatement с
          // этим nonce (находка №5 — раньше он был скрыт, публикация
          // миграции из приложения была невозможна)
          Text(l.nonceToCopy),
          Row(
            children: [
              SelectableText('$_nonce',
                  style: const TextStyle(
                      fontSize: 16, fontWeight: FontWeight.bold)),
              IconButton(
                icon: const Icon(Icons.copy, size: 18),
                tooltip: l.nonceToCopy,
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: '$_nonce'));
                  ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text(l.nonceToCopy)));
                },
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(l.sigsCollected(
            _approvals.length,
            needPersonal,
            manifest?.guardianPersonalCount ?? 3,
            needPublic,
            manifest?.guardianPublicCount ?? 5,
          )),
          for (final a in _approvals)
            ListTile(
              dense: true,
              leading: const Icon(Icons.how_to_reg),
              title: Text(a['guardian']?.shortHash() ?? ''),
            ),
          TextField(
            controller: _guardianInput,
            decoration: InputDecoration(labelText: l.guardianAddrLabel),
          ),
          TextField(
            controller: _sigInput,
            decoration: InputDecoration(labelText: l.guardianSigLabel),
          ),
          const SizedBox(height: 8),
          OutlinedButton(onPressed: _addApproval, child: Text(l.btnAddSig)),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: enough ? _propose : null,
            child: Text(l.btnPublishProposal),
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

/// Экран гардиана: подпись утверждения миграции своим ключом
/// (биометрия). Утверждение связывает сеть, старый и новый адрес,
/// nonce и эпоху (11).
class ApproveMigrationScreen extends StatefulWidget {
  const ApproveMigrationScreen({super.key});

  @override
  State<ApproveMigrationScreen> createState() => _ApproveMigrationScreenState();
}

class _ApproveMigrationScreenState extends State<ApproveMigrationScreen> {
  final _oldAddress = TextEditingController();
  final _newAddress = TextEditingController();
  final _nonce = TextEditingController();
  String? _sig;
  String? _error;

  Future<void> _sign() async {
    final platform = context.read<PlatformService>();
    final l = AppLocalizations.of(context)!;
    try {
      final r = await platform.signMigrateStatement(
        _oldAddress.text.trim(),
        _newAddress.text.trim(),
        int.tryParse(_nonce.text) ?? 0,
        l.wardSignTitle,
      );
      final sig = r['sig'] as String? ?? '';
      setState(() {
        _sig = sig.isEmpty ? null : sig;
        _error = sig.isEmpty ? l.signNotConfirmed : null;
      });
    } catch (e) {
      setState(() => _error = l.errorWith('$e'));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.approveTitle)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(l.approveIntro),
          const SizedBox(height: 12),
          TextField(
            controller: _oldAddress,
            decoration: InputDecoration(labelText: l.oldAddressLabel),
          ),
          TextField(
            controller: _newAddress,
            decoration: InputDecoration(labelText: l.newAddressLabel),
          ),
          TextField(
            controller: _nonce,
            decoration: InputDecoration(labelText: l.nonceLabel),
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 16),
          FilledButton(onPressed: _sign, child: Text(l.btnSignBiometric)),
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
