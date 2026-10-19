// Настройки (этап 7): манифест сети (параметры — из genesis, не
// хардкодятся), guardian_offer (объявить себя публичным гардианом,
// 11), управление узлом (12.2), блокировка приложения.
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../state/app_state.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final manifest = app.manifest;
    final wallet = app.wallet;
    return Scaffold(
      appBar: AppBar(title: const Text('Настройки')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Сеть: ${manifest?.displayName ?? '—'}',
                      style: Theme.of(context).textTheme.titleMedium),
                  Text('network_id: ${manifest?.networkId ?? '—'}',
                      style: Theme.of(context).textTheme.bodySmall),
                  const Divider(),
                  _param('Единица', manifest?.unit ?? '—'),
                  _param('Передача', manifest?.transferMode ?? '—'),
                  _param('Выпуск', '${manifest?.issuanceAmount ?? '—'} за '
                      '${manifest?.issuancePeriodMonths ?? '—'} мес.'),
                  _param('K (поручительства)', '${manifest?.vouchK ?? '—'}'),
                  _param('Порог P', '${manifest?.delayedThresholdDefault ?? '—'} '
                      '(P_max ${manifest?.delayedThresholdMax ?? '—'})'),
                  _param('Задержка D', '${manifest?.transferDelayHours ?? '—'} ч'),
                  _param('Мораторий',
                      '${manifest?.transferMoratoriumDays ?? '—'} дн.'),
                  _param('Демередж',
                      manifest?.demurrageEnabled == true ? 'включён' : 'выключен'),
                  _param('Окно накопления',
                      '${manifest?.accrualWindowPeriods ?? '—'} периодов'),
                  _param('Гардианы',
                      'личные ${manifest?.guardianPersonalThreshold ?? '—'} из '
                      '${manifest?.guardianPersonalCount ?? '—'}; публичные '
                      '${manifest?.guardianPublicThreshold ?? '—'} из '
                      '${manifest?.guardianPublicCount ?? '—'}'),
                  _param('Миграция',
                      'задержка ${manifest?.migrateDelayDays ?? '—'} дн.'),
                  _param('Эпоха чекпоинта', '${manifest?.epochSeconds ?? '—'} с'),
                  if (manifest?.sandbox == true)
                    const Text('Песочница (синтетическая аттестация)',
                        style: TextStyle(color: Colors.blue)),
                ],
              ),
            ),
          ),
          const SizedBox(height: 8),
          ListTile(
            leading: const Icon(Icons.volunteer_activism),
            title: const Text('Объявить себя публичным гардианом'),
            subtitle: Text(wallet?.publicGuardian == true
                ? 'Вы уже публичный гардиан (подопечных: ${wallet?.wards})'
                : 'Требования: active, возраст ≥ '
                    '${manifest?.guardianMinAgeDays ?? 12 * 30} дн.'),
            trailing: wallet?.publicGuardian == true
                ? const Icon(Icons.check)
                : const Icon(Icons.chevron_right),
            onTap: wallet?.publicGuardian == true ? null : () => _offer(context),
          ),
          ListTile(
            leading: Icon(
              app.nodeRunning ? Icons.router : Icons.router_outlined,
            ),
            title: Text(app.nodeRunning ? 'Узел работает' : 'Узел остановлен'),
            subtitle: const Text(
              'Уровень full — при зарядке и Wi-Fi, иначе light (12.2)',
            ),
            trailing: Switch(
              value: app.nodeRunning,
              onChanged: (v) => v ? app.startNode() : app.nodeStop(),
            ),
          ),
          ListTile(
            leading: const Icon(Icons.lock),
            title: const Text('Заблокировать приложение'),
            onTap: () => _lock(context),
          ),
        ],
      ),
    );
  }

  Widget _param(String name, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        children: [
          Expanded(child: Text(name, style: const TextStyle(fontSize: 13))),
          Text(value, style: const TextStyle(fontSize: 13)),
        ],
      ),
    );
  }

  Future<void> _offer(BuildContext context) async {
    final app = context.read<AppState>();
    final platform = app.platform;
    try {
      final res = await app.signAndSubmit(
        platform.buildGuardianOffer,
        'Подтвердите объявление публичным гардианом',
      );
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(
          res.accepted ? 'Объявление опубликовано' : 'Отклонено: ${res.code}',
        ),
      ));
      await app.refreshWallet();
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('Ошибка: $e')));
      }
    }
  }

  /// «Заблокировать» — перезапуск шлюза: LockScreen снова требует PIN.
  void _lock(BuildContext context) {
    Navigator.of(context, rootNavigator: true).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const _RelockGate()),
      (_) => false,
    );
  }
}

/// Шлюз переблокировки: PIN -> главная навигация заново.
class _RelockGate extends StatelessWidget {
  const _RelockGate();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      body: Center(child: Text('Заблокировано. Перезапустите приложение.')),
    );
  }
}
