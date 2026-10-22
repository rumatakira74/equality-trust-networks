// Ввод длинных hex-строк чанками по 20 символов (PENDING №6, прогон
// 11.2): IME эмуляторов теряет/искажает символы при вставке длинной
// строки в одно поле. Чанки — отдельные поля с авто-переходом фокуса;
// вставка полной строки распределяется по чанкам; после вставки
// контролируются символы (только 0–9, a–f) и длина.
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../l10n/app_localizations.dart';

/// Размер чанка (символов в одном поле).
const int kHexChunkSize = 20;

/// Поле ввода длинной hex-строки чанками. Значение — конкатенация
/// чанков (нижний регистр); [expectedLength] > 0 — фиксированная длина
/// (адрес, хеш), 0 — переменная (подписи DER).
class HexChunkField extends StatefulWidget {
  const HexChunkField({
    super.key,
    required this.label,
    required this.chunkCount,
    this.expectedLength = 0,
    this.onChanged,
  });

  final String label;
  final int chunkCount;
  final int expectedLength;
  final ValueChanged<String>? onChanged;

  /// Допустимые символы после вставки (hex, нижний регистр).
  static String sanitize(String raw) =>
      raw.toLowerCase().replaceAll(RegExp(r'[^0-9a-f]'), '');

  /// Валидность значения: фиксированная длина — точное совпадение,
  /// переменная — непустая и чётная (байты hex).
  static bool isValid(String value, int expectedLength) {
    if (value.isEmpty) return false;
    if (expectedLength > 0) return value.length == expectedLength;
    return value.length % 2 == 0;
  }

  @override
  State<HexChunkField> createState() => _HexChunkFieldState();
}

class _HexChunkFieldState extends State<HexChunkField> {
  late final List<TextEditingController> _controllers;
  late final List<FocusNode> _nodes;
  String _value = '';
  bool _dropped = false;

  @override
  void initState() {
    super.initState();
    _controllers = List.generate(
      widget.chunkCount,
      (_) => TextEditingController(),
    );
    _nodes = List.generate(widget.chunkCount, (_) => FocusNode());
  }

  @override
  void dispose() {
    for (final c in _controllers) {
      c.dispose();
    }
    for (final n in _nodes) {
      n.dispose();
    }
    super.dispose();
  }

  /// Пересборка значения: sanitize всех чанков, конкатенация,
  /// равномерное распределение по 20 символов (вставка полной строки
  /// в первый чанк раскладывается по полям), авто-переход фокуса.
  void _rebuild({int? edited}) {
    final raw = _controllers.map((c) => c.text).join();
    final clean = HexChunkField.sanitize(raw);
    // контроль символов после вставки (второй контур поверх inputFormatters)
    _dropped = clean.length != raw.length;
    final capacity = widget.chunkCount * kHexChunkSize;
    final value = clean.length > capacity ? clean.substring(0, capacity) : clean;
    for (var i = 0; i < widget.chunkCount; i++) {
      final chunk = i * kHexChunkSize;
      final part = chunk + kHexChunkSize <= value.length
          ? value.substring(chunk, chunk + kHexChunkSize)
          : (chunk < value.length ? value.substring(chunk) : '');
      _controllers[i].text = part;
    }
    if (_value != value) {
      _value = value;
      widget.onChanged?.call(value);
    }
    // счётчик/ошибка и распределение чанков — перестраиваем поле
    setState(() {});
    // авто-переход: чанк заполнен — фокус в следующий
    if (edited != null && edited + 1 < widget.chunkCount) {
      final part = _controllers[edited].text;
      if (part.length == kHexChunkSize) {
        _nodes[edited + 1].requestFocus();
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    final fixed = widget.expectedLength > 0;
    String? error;
    if (_dropped) {
      error = l.hexCharsDropped;
    } else if (fixed && _value.length > widget.expectedLength) {
      error = l.hexLenError(widget.expectedLength);
    } else if (!fixed && _value.isNotEmpty && _value.length % 2 != 0) {
      error = l.hexSigEvenError;
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(widget.label),
        const SizedBox(height: 4),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (var i = 0; i < widget.chunkCount; i++)
              SizedBox(
                width: 152,
                child: TextField(
                  controller: _controllers[i],
                  focusNode: _nodes[i],
                  style: const TextStyle(fontSize: 12),
                  inputFormatters: [
                    FilteringTextInputFormatter.allow(RegExp('[0-9a-fA-F]')),
                  ],
                  onChanged: (_) => _rebuild(edited: i),
                  decoration: InputDecoration(
                    isDense: true,
                    counterText: '',
                    hintText: '${(i) * kHexChunkSize + 1}–'
                        '${(i + 1) * kHexChunkSize}',
                  ),
                ),
              ),
          ],
        ),
        Text(
          fixed
              ? l.hexChunksCount(_value.length, widget.expectedLength)
              : l.hexChunksSigCount(_value.length),
          style: Theme.of(context).textTheme.bodySmall,
        ),
        if (error != null)
          Text(error, style: const TextStyle(color: Colors.red)),
      ],
    );
  }
}
