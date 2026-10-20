// Карточка восстановления (решение владельца 20.10.2026, этап 10):
// при создании кошелька приложение показывает экран «сохраните эти
// данные» — адрес кошелька, адреса назначенных гардианов, название
// сети + QR-код карточки (экспорт/печать) — с объяснением, зачем
// они нужны (восстановление при утрате телефона, 11) и как их
// сохранить (вне телефона: бумага, менеджер паролей); напоминание:
// данные миграции (nonce, подписи гардианов) никому не передавать.
// Карточка доступна и позже (настройки, экран миграции).
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../l10n/app_localizations.dart';
import '../../state/app_state.dart';

/// Данные карточки (адрес, гардианы, сеть) — строка QR-кода.
String recoveryCardPayload(String address, List<String> guardians,
        String networkName, String networkId) =>
    'tnp-recovery?net=$networkName&id=$networkId&addr=$address'
    '${guardians.isEmpty ? '' : '&guardians=${guardians.join(',')}'}';

class RecoveryCardScreen extends StatelessWidget {
  final VoidCallback? onDone;
  const RecoveryCardScreen({super.key, this.onDone});

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final wallet = app.wallet;
    final l = AppLocalizations.of(context)!;
    final guardians = wallet?.personalGuardians ?? const <String>[];
    final networkName = app.manifest?.displayName ?? '—';
    final networkId = app.manifest?.networkId ?? '';
    final address = app.address;
    return Scaffold(
      appBar: AppBar(title: Text(l.recoveryTitle)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(l.recoveryIntro, style: Theme.of(context).textTheme.bodyMedium),
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Center(
                    child: QrImageView(
                      data: recoveryCardPayload(
                          address, guardians, networkName, networkId),
                      size: 180,
                      backgroundColor: Colors.white,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(l.recoveryAddress,
                      style: Theme.of(context).textTheme.titleSmall),
                  SelectableText(address,
                      style: const TextStyle(fontSize: 11)),
                  const SizedBox(height: 8),
                  Text(l.recoveryNetwork,
                      style: Theme.of(context).textTheme.titleSmall),
                  Text(networkName),
                  if (networkId.isNotEmpty)
                    Text('network_id: $networkId',
                        style: Theme.of(context).textTheme.bodySmall),
                  const SizedBox(height: 8),
                  Text(l.recoveryGuardians,
                      style: Theme.of(context).textTheme.titleSmall),
                  if (guardians.isEmpty)
                    Text(l.recoveryNoGuardians)
                  else
                    for (final g in guardians)
                      SelectableText(g, style: const TextStyle(fontSize: 11)),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
          Text(l.recoveryHowToSave,
              style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 8),
          Card(
            color: Colors.orange.withValues(alpha: 0.12),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Text(l.recoveryNeverShare),
            ),
          ),
          if (onDone != null) ...[
            const SizedBox(height: 16),
            FilledButton(
              onPressed: onDone,
              child: Text(l.recoverySavedDone),
            ),
          ],
        ],
      ),
    );
  }
}
