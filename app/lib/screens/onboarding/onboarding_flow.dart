// Онбординг (этап 7, промпт п.2): биометрия (отказ без настроенной,
// 8.1) -> PIN обычный и тревожный с легендой (8.2) -> обязательное
// предложение гардианов (11; отказаться можно, шаг — нельзя) ->
// register (аттестация + усиленный PoW).
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';

class OnboardingFlow extends StatefulWidget {
  const OnboardingFlow({super.key});

  @override
  State<OnboardingFlow> createState() => _OnboardingFlowState();
}

class _OnboardingFlowState extends State<OnboardingFlow> {
  int _step = 0; // 0 биометрия, 1 PIN, 2 гардианы, 3 регистрация

  @override
  void initState() {
    super.initState();
    // узел нужен ДО выбора гардианов (реестр синхронизируется с пиром)
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().startNode();
    });
  }

  /// гардианы, выбранные на шаге 2 (передаются в register)
  final List<String> guardians = [];

  void _toRegister(List<String> selected) {
    guardians
      ..clear()
      ..addAll(selected);
    setState(() => _step = 3);
  }

  void _next() => setState(() => _step++);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Создание кошелька')),
      body: switch (_step) {
        0 => _BiometricsStep(onDone: _next),
        1 => _PinStep(onDone: _next),
        2 => _GuardiansStep(onDone: _toRegister),
        _ => const _RegisterStep(),
      },
    );
  }
}

/// Шаг 1: биометрия обязательна (8.1: устройство без настроенной
/// биометрии — регистрация не разрешается; системный экран добавления
/// отпечатка, затем повторная проверка).
class _BiometricsStep extends StatefulWidget {
  final VoidCallback onDone;
  const _BiometricsStep({required this.onDone});

  @override
  State<_BiometricsStep> createState() => _BiometricsStepState();
}

class _BiometricsStepState extends State<_BiometricsStep> {
  bool? _available;

  @override
  void initState() {
    super.initState();
    _check();
  }

  Future<void> _check() async {
    final platform = context.read<PlatformService>();
    final ok = await platform.biometricsAvailable();
    if (mounted) setState(() => _available = ok);
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Text(
            'Кошелёк требует аппаратного ключа с биометрией '
            '(BIOMETRIC_STRONG). Без настроенного отпечатка регистрация '
            'невозможна.',
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 16),
          if (_available == null)
            const CircularProgressIndicator()
          else if (_available!)
            FilledButton(onPressed: widget.onDone, child: const Text('Далее'))
          else
            Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Отпечаток не настроен. Откройте системный экран '
                  'настройки и добавьте отпечаток.',
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: () async {
                    await context
                        .read<PlatformService>()
                        .openBiometricsEnroll();
                    _check();
                  },
                  child: const Text('Настроить отпечаток'),
                ),
              ],
            ),
        ],
      ),
    );
  }
}

/// Шаг 2: PIN обычный (обязательный) и тревожный (опционально) с
/// легендой — правдоподобным остатком и историей (8.2).
class _PinStep extends StatefulWidget {
  final VoidCallback onDone;
  const _PinStep({required this.onDone});

  @override
  State<_PinStep> createState() => _PinStepState();
}

class _PinStepState extends State<_PinStep> {
  final _normal = TextEditingController();
  final _duress = TextEditingController();
  final _decoyBalance = TextEditingController();
  final _decoyHistory = TextEditingController();
  bool _wantDuress = false;
  String? _error;
  bool _busy = false;

  @override
  void dispose() {
    _normal.dispose();
    _duress.dispose();
    _decoyBalance.dispose();
    _decoyHistory.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _busy = true);
    try {
      final duress = _wantDuress && _duress.text.isNotEmpty ? _duress.text : null;
      if (_normal.text.length < 4) {
        throw TnpPlatformException('PIN', 'минимум 4 цифры');
      }
      if (duress != null && duress.length < 4) {
        throw TnpPlatformException('PIN', 'тревожный PIN: минимум 4 цифры');
      }
      if (duress != null && duress == _normal.text) {
        throw TnpPlatformException('PIN', 'PIN-ы совпадают');
      }
      final history = _decoyHistory.text
          .split('\n')
          .map((l) => l.trim())
          .where((l) => l.isNotEmpty)
          .toList();
      // легенда: правдоподобный небольшой остаток (ноль показывать
      // нельзя, 8.2); история — строки «дата — сумма — получатель»
      final balance = int.tryParse(_decoyBalance.text) ??
          (context.read<AppState>().manifest?.issuanceAmount ?? 1000);
      await context.read<PlatformService>().setupPin(
            _normal.text,
            duress,
            balance,
            history,
          );
      widget.onDone();
    } catch (e) {
      setState(() => _error = '$e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        const Text(
          'PIN защищает приложение. Тревожный PIN (по желанию) открывает '
          'режим-приманку: тот же интерфейс, но правдоподобный небольшой '
          'остаток; ключ подписи при этом не используется.',
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _normal,
          decoration: const InputDecoration(
            labelText: 'PIN (обычный)',
            errorText: null,
          ),
          obscureText: true,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        ),
        SwitchListTile(
          value: _wantDuress,
          onChanged: (v) => setState(() => _wantDuress = v),
          title: const Text('Задать тревожный PIN'),
        ),
        if (_wantDuress) ...[
          TextField(
            controller: _duress,
            decoration: const InputDecoration(labelText: 'Тревожный PIN'),
            obscureText: true,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          ),
          const SizedBox(height: 12),
          Text(
            'Легенда режима-приманки (что увидит принуждающий):',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          TextField(
            controller: _decoyBalance,
            decoration: const InputDecoration(
              labelText: 'Правдоподобный остаток (не 0)',
            ),
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          ),
          TextField(
            controller: _decoyHistory,
            maxLines: 3,
            decoration: const InputDecoration(
              labelText: 'История (строки: дата — сумма — получатель)',
              hintText: '2026-09-03 — 150 — aa12…ff90',
            ),
          ),
        ],
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Text(_error!, style: const TextStyle(color: Colors.red)),
          ),
        const SizedBox(height: 16),
        FilledButton(
          onPressed: _busy ? null : _save,
          child: const Text('Далее'),
        ),
      ],
    );
  }
}

/// Шаг 3: гардианы (11, 8.1). Предложение — обязательный шаг;
/// отказаться можно, но добавление нового отпечатка инвалидирует
/// ключ (setInvalidatedByBiometricEnrollment) — без гардианов доступ
/// теряется. Личные гардианы: 2 из 3; назначаются только при создании
/// (вопрос №6).
class _GuardiansStep extends StatefulWidget {
  final void Function(List<String> selected) onDone;
  const _GuardiansStep({required this.onDone});

  @override
  State<_GuardiansStep> createState() => _GuardiansStepState();
}

class _GuardiansStepState extends State<_GuardiansStep> {
  List<RegistryRow> _candidates = [];
  final Set<String> _selected = {};
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    // гардианы — известные адреса реестра (R6); узел только что
    // подключился к сети — синхронизация занимает время: повторяем
    final platform = context.read<PlatformService>();
    for (var attempt = 0; attempt < 10; attempt++) {
      try {
        final rows = await platform.registry();
        if (rows.isNotEmpty || attempt == 9) {
          if (mounted) {
            setState(() {
              _candidates = rows;
              _loading = false;
            });
          }
          return;
        }
      } catch (e) {
        if (attempt == 9 && mounted) {
          setState(() {
            _error = '$e';
            _loading = false;
          });
        }
      }
      await Future.delayed(const Duration(seconds: 2));
    }
  }

  @override
  Widget build(BuildContext context) {
    final manifest = context.watch<AppState>().manifest;
    final maxG = manifest?.guardianPersonalCount ?? 3;
    return Column(
      children: [
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    const Text(
                      'Назначьте личных гардианов (2 из 3). Они смогут '
                      'подтвердить миграцию на новое устройство. Добавление '
                      'нового отпечатка инвалидирует ключ — без гардианов '
                      'доступ к кошельку теряется.',
                    ),
                    const SizedBox(height: 8),
                    if (_error != null)
                      Text(_error!, style: const TextStyle(color: Colors.red)),
                    if (_candidates.isEmpty)
                      const Text(
                        'В сети пока нет других участников. Гардианов можно '
                        'не назначать сейчас, но предложение обязательно: '
                        'позже список не меняется (только при создании).',
                      ),
                    for (final row in _candidates)
                      CheckboxListTile(
                        value: _selected.contains(row.address),
                        onChanged: (v) => setState(() {
                          if (v == true) {
                            if (_selected.length < maxG) {
                              _selected.add(row.address);
                            }
                          } else {
                            _selected.remove(row.address);
                          }
                        }),
                        title: Text(row.address.shortHash()),
                        subtitle: Text(
                          '${row.status}, возраст ${row.ageDays} дн.',
                        ),
                      ),
                  ],
                ),
        ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceEvenly,
            children: [
              // отказаться можно, но предложение — обязательный шаг (11)
              TextButton(
                onPressed: () => widget.onDone(const []),
                child: const Text('Пропустить'),
              ),
              OutlinedButton(
                onPressed: () {
                  setState(() => _loading = true);
                  _load();
                },
                child: const Text('Обновить'),
              ),
              FilledButton(
                onPressed: () => widget.onDone(_selected.toList()),
                child: Text('Выбрано: ${_selected.length}'),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// Шаг 4: register — аттестация + усиленный PoW + биометрическая
/// подпись + публикация; затем запуск узла.
class _RegisterStep extends StatefulWidget {
  const _RegisterStep();

  @override
  State<_RegisterStep> createState() => _RegisterStepState();
}

class _RegisterStepState extends State<_RegisterStep> {
  String _status = 'Готово к регистрации';
  bool _busy = false;

  Future<void> _register() async {
    setState(() {
      _busy = true;
      _status = 'Аттестация и PoW (может занять время)…';
    });
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final flow = context.findAncestorStateOfType<_OnboardingFlowState>();
    try {
      await app.startNode();
      final guardians = flow?.guardians ?? const <String>[];
      final res = await app.signAndSubmit(
        () => platform.buildRegister(guardians),
        'Подтвердите регистрацию',
      );
      if (!mounted) return;
      setState(() => _status = res.accepted
          ? 'Кошелёк зарегистрирован'
          : 'Отклонено: ${res.code}');
      if (res.accepted) {
        await app.onboardingDone();
      }
    } catch (e) {
      setState(() => _status = 'Ошибка: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(_status, textAlign: TextAlign.center),
          const SizedBox(height: 16),
          if (!_busy)
            FilledButton(
              onPressed: _register,
              child: const Text('Зарегистрировать кошелёк'),
            ),
        ],
      ),
    );
  }
}
