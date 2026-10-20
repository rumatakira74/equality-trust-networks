// Онбординг (этап 7, промпт п.2): биометрия (отказ без настроенной,
// 8.1) -> PIN обычный и тревожный с легендой (8.2) -> обязательное
// предложение гардианов (11; отказаться можно, шаг — нельзя) ->
// register (аттестация + усиленный PoW).
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../l10n/app_localizations.dart';
import '../../services/models.dart';
import '../../services/platform_service.dart';
import '../../state/app_state.dart';
import '../recovery/recovery_card_screen.dart';

class OnboardingFlow extends StatefulWidget {
  const OnboardingFlow({super.key});

  @override
  State<OnboardingFlow> createState() => _OnboardingFlowState();
}

class _OnboardingFlowState extends State<OnboardingFlow> {
  // 0 биометрия, 1 PIN, 2 гардианы, 3 регистрация, 4 карточка
  // восстановления (решение владельца 20.10.2026: «сохраните эти данные»)
  int _step = 0;

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
    final l = AppLocalizations.of(context)!;
    return Scaffold(
      appBar: AppBar(title: Text(l.obTitle)),
      body: switch (_step) {
        0 => _BiometricsStep(onDone: _next),
        1 => _PinStep(onDone: _next),
        2 => _GuardiansStep(onDone: _toRegister),
        3 => _RegisterStep(onDone: () => setState(() => _step = 4)),
        _ => RecoveryCardScreen(onDone: () {
          context.read<AppState>().onboardingDone();
        }),
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
    final l = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            l.obBioRequired,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 16),
          if (_available == null)
            const CircularProgressIndicator()
          else if (_available!)
            FilledButton(onPressed: widget.onDone, child: Text(l.btnNext))
          else
            Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  l.obBioNotSetup,
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
                  child: Text(l.obBioSetup),
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
    final l = AppLocalizations.of(context)!;
    setState(() => _busy = true);
    try {
      final duress = _wantDuress && _duress.text.isNotEmpty ? _duress.text : null;
      if (_normal.text.length < 4) {
        throw TnpPlatformException('PIN', l.pinMinDigits);
      }
      if (duress != null && duress.length < 4) {
        throw TnpPlatformException('PIN', l.duressPinMinDigits);
      }
      if (duress != null && duress == _normal.text) {
        throw TnpPlatformException('PIN', l.pinsMatch);
      }
      final history = _decoyHistory.text
          .split('\n')
          .map((line) => line.trim())
          .where((line) => line.isNotEmpty)
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
    final l = AppLocalizations.of(context)!;
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        Text(l.obPinIntro),
        const SizedBox(height: 16),
        TextField(
          controller: _normal,
          decoration: InputDecoration(
            labelText: l.obPinNormal,
            errorText: null,
          ),
          obscureText: true,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        ),
        SwitchListTile(
          value: _wantDuress,
          onChanged: (v) => setState(() => _wantDuress = v),
          title: Text(l.obWantDuress),
        ),
        if (_wantDuress) ...[
          TextField(
            controller: _duress,
            decoration: InputDecoration(labelText: l.obDuressPin),
            obscureText: true,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          ),
          const SizedBox(height: 12),
          Text(
            l.obDecoyLegend,
            style: Theme.of(context).textTheme.bodySmall,
          ),
          TextField(
            controller: _decoyBalance,
            decoration: InputDecoration(
              labelText: l.obDecoyBalance,
            ),
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          ),
          TextField(
            controller: _decoyHistory,
            maxLines: 3,
            decoration: InputDecoration(
              labelText: l.obDecoyHistory,
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
          child: Text(l.btnNext),
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
    final l = AppLocalizations.of(context)!;
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
                    Text(l.obGuardiansIntro),
                    const SizedBox(height: 8),
                    if (_error != null)
                      Text(_error!, style: const TextStyle(color: Colors.red)),
                    if (_candidates.isEmpty)
                      Text(l.obGuardiansEmpty),
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
                        subtitle: Text(l.guardianRowStatus(row.status, row.ageDays)),
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
                child: Text(l.btnSkip),
              ),
              OutlinedButton(
                onPressed: () {
                  setState(() => _loading = true);
                  _load();
                },
                child: Text(l.btnRefresh),
              ),
              FilledButton(
                onPressed: () => widget.onDone(_selected.toList()),
                child: Text(l.obSelectedCount(_selected.length)),
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
  final VoidCallback onDone;
  const _RegisterStep({required this.onDone});

  @override
  State<_RegisterStep> createState() => _RegisterStepState();
}

class _RegisterStepState extends State<_RegisterStep> {
  String? _status;
  bool _busy = false;

  Future<void> _register() async {
    final l = AppLocalizations.of(context)!;
    setState(() {
      _busy = true;
      _status = l.regStatusPow;
    });
    final app = context.read<AppState>();
    final platform = context.read<PlatformService>();
    final flow = context.findAncestorStateOfType<_OnboardingFlowState>();
    try {
      await app.startNode();
      final guardians = flow?.guardians ?? const <String>[];
      final res = await app.signAndSubmit(
        () => platform.buildRegister(guardians),
        l.regConfirmTitle,
      );
      if (!mounted) return;
      setState(() => _status = res.accepted
          ? l.regStatusDone
          : l.rejectedWithCode(res.code));
      if (res.accepted) {
        // карточка восстановления — до завершения онбординга (этап 10)
        widget.onDone();
      }
    } catch (e) {
      setState(() => _status = l.errorWith('$e'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(_status ?? l.regStatusReady, textAlign: TextAlign.center),
          const SizedBox(height: 16),
          if (!_busy)
            FilledButton(
              onPressed: _register,
              child: Text(l.btnRegister),
            ),
        ],
      ),
    );
  }
}
