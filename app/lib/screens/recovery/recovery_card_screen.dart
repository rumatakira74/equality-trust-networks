// Карточка восстановления (решение владельца 20.10.2026, спека v9.7
// раздел 11; аудит и доработка — этап 10.1): при создании кошелька
// приложение показывает экран «сохраните эти данные» — адрес кошелька,
// адреса назначенных гардианов, название сети + QR-код карточки —
// с пояснением, зачем они нужны (восстановление при утрате телефона),
// как их сохранить (вне телефона, не в облако), зачем заранее сохранить
// APK той же сети; напоминание: данные миграции (nonce, подписи
// гардианов) передаются только напрямую своему гардиану и удаляются
// из переписок после миграции. Экспорт — системный share изображения
// карточки (п.3 мандата 10.1). Карточка доступна позже из настроек и
// с экрана миграции; просмотр — после разблокировки приложения
// (биометрия или PIN на входе), отдельный PIN-промпт не нужен —
// карточка не деструктивна (п.6).
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';

import '../../l10n/app_localizations.dart';
import '../../state/app_state.dart';

/// Данные карточки (адрес, гардианы, сеть) — строка QR-кода.
String recoveryCardPayload(String address, List<String> guardians,
        String networkName, String networkId) =>
    'tnp-recovery?net=$networkName&id=$networkId&addr=$address'
    '${guardians.isEmpty ? '' : '&guardians=${guardians.join(',')}'}';

class RecoveryCardScreen extends StatefulWidget {
  final VoidCallback? onDone;
  /// Точка внедрения для тестов: системный share заменяется моком
  /// (прод-путь — SharePlus; реальный IO share_plus не живёт в
  /// fake-зоне виджет-тестов).
  final Future<void> Function(Uint8List png)? shareOverride;
  const RecoveryCardScreen({super.key, this.onDone, this.shareOverride});

  @override
  State<RecoveryCardScreen> createState() => _RecoveryCardScreenState();
}

class _RecoveryCardScreenState extends State<RecoveryCardScreen> {
  final _cardKey = GlobalKey<State>();

  /// Экспорт карточки (п.3): системный share изображения — карточка
  /// рендерится в PNG (RepaintBoundary) и передаётся системному
  /// диалогу (печать, сохранение, передача).
  Future<void> _shareCard() async {
    final boundary = _cardKey.currentContext?.findRenderObject()
        as RenderRepaintBoundary?;
    if (boundary == null) return;
    final image = await boundary.toImage(pixelRatio: 3.0);
    final data = await image.toByteData(format: ui.ImageByteFormat.png);
    if (data == null) return;
    final png = data.buffer.asUint8List();
    final share = widget.shareOverride ?? _systemShare;
    await share(png);
  }

  /// Прод-путь: системный share PNG-изображения карточки.
  Future<void> _systemShare(Uint8List png) async {
    final file = XFile.fromData(
      png,
      name: 'recovery-card.png',
      mimeType: 'image/png',
    );
    await SharePlus.instance.share(ShareParams(files: [file]));
  }

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
          RepaintBoundary(
            key: _cardKey,
            child: Card(
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
                        SelectableText(g,
                            style: const TextStyle(fontSize: 11)),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(height: 8),
          // экспорт (п.3): системный share / сохранение изображения
          OutlinedButton.icon(
            onPressed: _shareCard,
            icon: const Icon(Icons.ios_share),
            label: Text(l.recoveryShareBtn),
          ),
          const SizedBox(height: 12),
          Text(l.recoveryHowToSave,
              style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 4),
          // пояснение про APK той же сети (п.2)
          Text(l.recoveryApkNote,
              style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 8),
          Card(
            color: Colors.orange.withValues(alpha: 0.12),
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Text(l.recoveryNeverShare),
            ),
          ),
          if (widget.onDone != null) ...[
            const SizedBox(height: 16),
            FilledButton(
              onPressed: widget.onDone,
              child: Text(l.recoverySavedDone),
            ),
          ],
        ],
      ),
    );
  }
}
