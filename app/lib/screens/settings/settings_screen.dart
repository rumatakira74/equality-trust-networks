// Настройки (этап 7): манифест сети (параметры — из genesis, не
// хардкодятся), guardian_offer (объявить себя публичным гардианом,
// 11), управление узлом (12.2), блокировка приложения.
// Этап 10: выбор языка интерфейса (ru, en, es, pt, zh — состав
// подтверждён владельцем 20.10.2026; fallback — английский).
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../state/app_state.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final manifest = app.manifest;
    final wallet = app.wallet;
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.settingsTitle)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _languageCard(context, app, l),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(l.networkLine(manifest?.displayName ?? '—'),
                      style: Theme.of(context).textTheme.titleMedium),
                  Text('network_id: ${manifest?.networkId ?? '—'}',
                      style: Theme.of(context).textTheme.bodySmall),
                  const Divider(),
                  _param(l.paramUnit, manifest?.unit ?? '—'),
                  _param(l.paramTransfer, manifest?.transferMode ?? '—'),
                  _param(l.paramIssuance,
                      l.paramIssuanceValue('${manifest?.issuanceAmount ?? '—'}',
                          '${manifest?.issuancePeriodMonths ?? '—'}')),
                  _param(l.paramVouchK, '${manifest?.vouchK ?? '—'}'),
                  _param(l.paramThresholdP,
                      l.paramThresholdPValue(
                          '${manifest?.delayedThresholdDefault ?? '—'}',
                          '${manifest?.delayedThresholdMax ?? '—'}')),
                  _param(l.paramDelayD,
                      l.paramDelayDValue('${manifest?.transferDelayHours ?? '—'}')),
                  _param(l.paramMoratorium,
                      l.paramMoratoriumValue(
                          '${manifest?.transferMoratoriumDays ?? '—'}')),
                  _param(l.paramDemurrage,
                      manifest?.demurrageEnabled == true
                          ? l.demurrageOn
                          : l.demurrageOff),
                  _param(l.paramAccrual,
                      l.paramAccrualValue('${manifest?.accrualWindowPeriods ?? '—'}')),
                  _param(l.paramGuardians,
                      l.paramGuardiansValue(
                          '${manifest?.guardianPersonalThreshold ?? '—'}',
                          '${manifest?.guardianPersonalCount ?? '—'}',
                          '${manifest?.guardianPublicThreshold ?? '—'}',
                          '${manifest?.guardianPublicCount ?? '—'}')),
                  _param(l.paramMigration,
                      l.paramMigrationValue('${manifest?.migrateDelayDays ?? '—'}')),
                  _param(l.paramEpoch,
                      l.paramEpochValue('${manifest?.epochSeconds ?? '—'}')),
                  if (manifest?.sandbox == true)
                    Text(l.sandboxNote,
                        style: const TextStyle(color: Colors.blue)),
                ],
              ),
            ),
          ),
          const SizedBox(height: 8),
          ListTile(
            leading: const Icon(Icons.volunteer_activism),
            title: Text(l.guardianOfferTitle),
            subtitle: Text(wallet?.publicGuardian == true
                ? l.guardianAlready(wallet?.wards ?? 0)
                : l.guardianReq(manifest?.guardianMinAgeDays ?? 12 * 30)),
            trailing: wallet?.publicGuardian == true
                ? const Icon(Icons.check)
                : const Icon(Icons.chevron_right),
            onTap: wallet?.publicGuardian == true ? null : () => _offer(context),
          ),
          ListTile(
            leading: Icon(
              app.nodeRunning ? Icons.router : Icons.router_outlined,
            ),
            title: Text(app.nodeRunning ? l.nodeRunning : l.nodeStopped),
            subtitle: Text(l.nodeLevelHint),
            trailing: Switch(
              value: app.nodeRunning,
              onChanged: (v) => v ? app.startNode() : app.nodeStop(),
            ),
          ),
          ListTile(
            leading: const Icon(Icons.lock),
            title: Text(l.lockApp),
            onTap: () => _lock(context),
          ),
        ],
      ),
    );
  }

  /// Выбор языка (этап 10): системный или один из пяти; сохраняется
  /// в SharedPreferences (AppState.setLocale).
  Widget _languageCard(BuildContext context, AppState app, AppLocalizations l) {
    const languages = ['ru', 'en', 'es', 'pt', 'zh'];
    String label(String code) {
      switch (code) {
        case 'ru':
          return l.languageRu;
        case 'en':
          return l.languageEn;
        case 'es':
          return l.languageEs;
        case 'pt':
          return l.languagePt;
        case 'zh':
          return l.languageZh;
        default:
          return l.languageSystem;
      }
    }

    final current = app.locale?.languageCode ?? '';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l.languageTitle, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              children: [
                for (final code in [''] + languages)
                  ChoiceChip(
                    label: Text(label(code)),
                    selected: current == code,
                    onSelected: (_) =>
                        app.setLocale(code.isEmpty ? null : Locale(code)),
                  ),
              ],
            ),
          ],
        ),
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
    final l = AppLocalizations.of(context)!;
    try {
      final res = await app.signAndSubmit(
        platform.buildGuardianOffer,
        l.guardianConfirmTitle,
      );
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(
          res.accepted ? l.guardianPublished : l.rejectedWithCode(res.code),
        ),
      ));
      await app.refreshWallet();
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(l.errorWith('$e'))));
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
    return Scaffold(
      body: Center(
          child: Text(AppLocalizations.of(context)!.relockNote)),
    );
  }
}
