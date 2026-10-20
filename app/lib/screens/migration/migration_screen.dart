// Миграция (промпт п.6; 11): запуск (личные 2 из 3, публичные 3 из 5,
// пороги комбинируются), отмена живым старым ключом, задержка 14 дней
// с обратным отсчётом; перезапуск через обычную register (старый
// кошелёк остаётся в реестре — предупреждение о потере средств);
// экран выбора публичных гардианов (guardian_offer).
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import '../pin_gate.dart';

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
    final active = _pending.where((m) => !m.canceled && !m.executed).toList();
    return Scaffold(
      appBar: AppBar(
        title: const Text('Миграция'),
        actions: [IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh))],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text(
            'Восстановление кошелька: гардианы назначаются заранее '
            '(при создании). Личные — 2 из 3; с публичными — 3 из 5 '
            '(пороги комбинируются).',
          ),
          const SizedBox(height: 12),
          if (active.isNotEmpty) ...[
            for (final m in active)
              Card(
                child: ListTile(
                  leading: const Icon(Icons.schedule),
                  title: Text('Миграция на ${m.newAddress.shortHash()}'),
                  subtitle: Text(
                    'исполнение через ${(m.remainingSec / 86400).round()} дн. '
                    '(задержка ${manifest?.migrateDelayDays ?? 14} дн.); '
                    'подписей гардианов: ${m.approvals.length}',
                  ),
                  // отмена живым старым ключом (11)
                  trailing: m.oldAddress == app.address
                      ? OutlinedButton(
                          onPressed: () => _cancel(m),
                          child: const Text('Отменить'),
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
            label: const Text('Запустить миграцию (новое устройство)'),
          ),
          const SizedBox(height: 8),
          OutlinedButton.icon(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const ApproveMigrationScreen()),
            ),
            icon: const Icon(Icons.how_to_reg),
            label: const Text('Подтвердить чужую миграцию (я — гардиан)'),
          ),
          const SizedBox(height: 24),
          Text('Публичные гардианы сети (guardian_offer, 11)',
              style: Theme.of(context).textTheme.titleMedium),
          if (_offers.isEmpty)
            const Text('Пока никто не объявил себя публичным гардианом'),
          for (final o in _offers)
            ListTile(
              leading: const Icon(Icons.volunteer_activism),
              title: Text(o.address.shortHash()),
              subtitle: Text('возраст ${o.ageDays} дн., подопечных: ${o.wards}'),
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
                  const Text(
                    'Если гардианов нет: перезапуск — обычная регистрация '
                    'с новым ключом и новыми K поручительствами. Старый '
                    'кошелёк остаётся в реестре нетронутым и недоступным.',
                    style: TextStyle(color: Colors.red),
                  ),
                  const SizedBox(height: 8),
                  OutlinedButton(
                    onPressed: () => _restartWarning(context),
                    child: const Text('Перезапустить кошелёк'),
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
    try {
      final res = await app.signAndSubmit(
        () => platform.buildMigrateCancel(m.newAddress, m.proposeSeq),
        'Подтвердите отмену миграции (старый ключ)',
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(res.accepted ? 'Миграция отменена' : 'Отклонено: ${res.code}'),
      ));
      _refresh();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Ошибка: $e')));
      }
    }
  }

  Future<void> _restartWarning(BuildContext context) async {
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Потеря средств'),
        content: const Text(
          'Перезапуск не даёт ничего сверх регистрации нового кошелька: '
          'средства на старом адресе теряются, история не переносится. '
          'Миграция с гардианами переносит возраст, историю выпусков и '
          'поручительства.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(ctx).pop(), child: const Text('Отмена')),
          FilledButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              // перезапуск — обычная register: онбординг заново
              // (ключ будет перегенерирован на этом устройстве)
              Navigator.of(context).pushReplacement(
                MaterialPageRoute(builder: (_) => const _RestartFlow()),
              );
            },
            child: const Text('Понимаю, перезапустить'),
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
    return const Scaffold(
      body: Center(
        child: Text('Перезапуск: удалите данные приложения и пройдите '
            'создание кошелька заново (новый ключ, новые K поручительств).'),
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
    if (!await confirmPin(context, title: 'Подтвердите миграцию')) {
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
        'Разблокировка ключа',
      );
      setState(() => _result = res.accepted
          ? 'Миграция запущена: исполнение через '
              '${app.manifest?.migrateDelayDays ?? 14} дн.; старый ключ '
              'может отменить в течение задержки'
          : 'Отклонено: ${res.code}');
    } catch (e) {
      setState(() => _result = 'Ошибка: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    final manifest = context.watch<AppState>().manifest;
    final needPersonal = manifest?.guardianPersonalThreshold ?? 2;
    final needPublic = manifest?.guardianPublicThreshold ?? 3;
    final enough = _approvals.length >= needPersonal;
    return Scaffold(
      appBar: AppBar(title: const Text('Запуск миграции')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Text(
            'Вы на НОВОМ устройстве. Введите адрес старого кошелька и '
            'подписи гардианов (каждый гардиан подписывает утверждение в '
            'своём приложении — вкладка «Подтвердить чужую миграцию»).',
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _oldAddress,
            decoration: const InputDecoration(labelText: 'Старый адрес (hex)'),
          ),
          const SizedBox(height: 8),
          Text('Подписей собрано: ${_approvals.length} '
              '(личный порог $needPersonal из ${manifest?.guardianPersonalCount ?? 3}; '
              'с публичными — $needPublic из ${manifest?.guardianPublicCount ?? 5})'),
          for (final a in _approvals)
            ListTile(
              dense: true,
              leading: const Icon(Icons.how_to_reg),
              title: Text(a['guardian']?.shortHash() ?? ''),
            ),
          TextField(
            controller: _guardianInput,
            decoration: const InputDecoration(labelText: 'Адрес гардиана (hex)'),
          ),
          TextField(
            controller: _sigInput,
            decoration: const InputDecoration(labelText: 'Подпись гардиана (hex)'),
          ),
          const SizedBox(height: 8),
          OutlinedButton(onPressed: _addApproval, child: const Text('Добавить подпись')),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: enough ? _propose : null,
            child: const Text('Опубликовать предложение'),
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
    try {
      final r = await platform.signMigrateStatement(
        _oldAddress.text.trim(),
        _newAddress.text.trim(),
        int.tryParse(_nonce.text) ?? 0,
        'Подтвердите миграцию подопечного (вы — гардиан)',
      );
      final sig = r['sig'] as String? ?? '';
      setState(() {
        _sig = sig.isEmpty ? null : sig;
        _error = sig.isEmpty ? 'Подпись не подтверждена' : null;
      });
    } catch (e) {
      setState(() => _error = 'Ошибка: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Подтверждение миграции (гардиан)')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          const Text(
            'Проверьте личность подопечного по своим каналам. Подпись '
            'связывает пару адресов и nonce; передайте её подопечному '
            'лично.',
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _oldAddress,
            decoration: const InputDecoration(labelText: 'Старый адрес (hex)'),
          ),
          TextField(
            controller: _newAddress,
            decoration: const InputDecoration(labelText: 'Новый адрес (hex)'),
          ),
          TextField(
            controller: _nonce,
            decoration: const InputDecoration(labelText: 'Nonce (от подопечного)'),
            keyboardType: TextInputType.number,
          ),
          const SizedBox(height: 16),
          FilledButton(onPressed: _sign, child: const Text('Подписать (биометрия)')),
          if (_sig != null) ...[
            const SizedBox(height: 12),
            const Text('Подпись (передайте подопечному):'),
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
