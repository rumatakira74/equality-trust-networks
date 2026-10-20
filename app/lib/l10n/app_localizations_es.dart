// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Spanish Castilian (`es`).
class AppLocalizationsEs extends AppLocalizations {
  AppLocalizationsEs([String locale = 'es']) : super(locale);

  @override
  String get appTitle => 'Redes de confianza';

  @override
  String get navWallet => 'Monedero';

  @override
  String get navVouching => 'Avales';

  @override
  String get navRegistry => 'Registro';

  @override
  String get navMigration => 'Migración';

  @override
  String get navSettings => 'Ajustes';

  @override
  String get btnNext => 'Siguiente';

  @override
  String get btnCancel => 'Cancelar';

  @override
  String get btnConfirm => 'Confirmar';

  @override
  String get btnRefresh => 'Actualizar';

  @override
  String get btnSkip => 'Omitir';

  @override
  String get btnLogin => 'Entrar';

  @override
  String get btnFingerprint => 'Huella';

  @override
  String get btnSend => 'Enviar';

  @override
  String get btnSendBt => 'Enviar por Bluetooth';

  @override
  String get btnVouch => 'Avalar';

  @override
  String get btnMyQr => 'Mi QR';

  @override
  String get btnHideQr => 'Ocultar mi QR';

  @override
  String get btnBtReceive => 'Recibir por BT';

  @override
  String get btnMint => 'Emisión (desafío)';

  @override
  String get btnMintDo => 'Emitir (PoW reforzado)';

  @override
  String get btnTransfer => 'Transferir';

  @override
  String get btnPublishAwaiting => 'Publicar (red disponible)';

  @override
  String get btnChangeThreshold => 'Cambiar umbral';

  @override
  String get btnCancelTransfer => 'Cancelar';

  @override
  String get btnRegister => 'Registrar monedero';

  @override
  String get btnStartMigration => 'Iniciar migración (nuevo dispositivo)';

  @override
  String get btnApproveMigration =>
      'Aprobar la migración de otro (soy guardián)';

  @override
  String get btnRestart => 'Reiniciar monedero';

  @override
  String get btnUnderstandRestart => 'Entiendo, reiniciar';

  @override
  String get btnAddSig => 'Añadir firma';

  @override
  String get btnPublishProposal => 'Publicar propuesta';

  @override
  String get btnSignBiometric => 'Firmar (biometría)';

  @override
  String get enterPin => 'Introduzca el PIN';

  @override
  String get errWrongPin => 'PIN incorrecto';

  @override
  String get confirmOperation => 'Confirme la operación';

  @override
  String get keyUnlockTitle => 'Desbloquear clave';

  @override
  String rejectedWithCode(String code) {
    return 'Rechazado: $code';
  }

  @override
  String errorWith(String e) {
    return 'Error: $e';
  }

  @override
  String get obTitle => 'Creación del monedero';

  @override
  String get obBioRequired =>
      'El monedero requiere una clave hardware con biometría (BIOMETRIC_STRONG). Sin una huella configurada el registro es imposible.';

  @override
  String get obBioNotSetup =>
      'No hay huella configurada. Abra la pantalla de ajustes del sistema y añada una huella.';

  @override
  String get obBioSetup => 'Configurar huella';

  @override
  String get obPinIntro =>
      'El PIN protege la app. Un PIN de coacción (opcional) abre el modo señuelo: la misma interfaz, pero un saldo pequeño verosímil; la clave de firma no se usa.';

  @override
  String get obPinNormal => 'PIN (normal)';

  @override
  String get obWantDuress => 'Definir PIN de coacción';

  @override
  String get obDuressPin => 'PIN de coacción';

  @override
  String get obDecoyLegend =>
      'Leyenda del modo señuelo (lo que verá quien coaccione):';

  @override
  String get obDecoyBalance => 'Saldo verosímil (no 0)';

  @override
  String get obDecoyHistory =>
      'Historial (líneas: fecha — importe — destinatario)';

  @override
  String get pinMinDigits => 'mínimo 4 dígitos';

  @override
  String get duressPinMinDigits => 'PIN de coacción: mínimo 4 dígitos';

  @override
  String get pinsMatch => 'Los PIN coinciden';

  @override
  String get obGuardiansIntro =>
      'Asigne guardianes personales (2 de 3). Podrán aprobar la migración a un dispositivo nuevo. Añadir una huella nueva invalida la clave — sin guardianes se pierde el acceso al monedero.';

  @override
  String get obGuardiansEmpty =>
      'Todavía no hay otros miembros en la red. Puede no asignar guardianes ahora, pero la oferta es obligatoria: la lista no se puede cambiar después (solo al crear).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, edad $age días';
  }

  @override
  String obSelectedCount(int count) {
    return 'Seleccionados: $count';
  }

  @override
  String get regStatusReady => 'Listo para registrar';

  @override
  String get regStatusPow => 'Atestación y PoW (puede tardar)…';

  @override
  String get regConfirmTitle => 'Confirme el registro';

  @override
  String get regStatusDone => 'Monedero registrado';

  @override
  String pendingCount(int count) {
    return 'Transferencias aplazadas: $count';
  }

  @override
  String get pendingHint =>
      'Las transferencias grandes se ejecutan tras el retraso D; se pueden cancelar dentro de D';

  @override
  String get historyTitle => 'Historial';

  @override
  String get historyEmpty => 'Todavía nada';

  @override
  String get effBalance => 'saldo efectivo';

  @override
  String get effBalanceDem => 'saldo efectivo (demurrage aplicado)';

  @override
  String lockedReserve(int amount) {
    return 'reservado para aplazadas: $amount';
  }

  @override
  String get flaggedNotice =>
      'El monedero está marcado (flagged) — la marca es pública, sin veredictos';

  @override
  String get registeredNeedK =>
      'Estado registered: la activación requiere K avales (pestaña «Avales»)';

  @override
  String get btAwaitingTitle =>
      'Recibido por Bluetooth — pendiente de publicación';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount de $hash';
  }

  @override
  String btStatus(String status) {
    return 'estado: $status';
  }

  @override
  String get txRegister => 'Registro del monedero';

  @override
  String get txVouch => 'Aval';

  @override
  String txMint(int period) {
    return 'Emisión (periodo $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Transferencia $amount → $hash';
  }

  @override
  String get mintTitle => 'Emisión';

  @override
  String mintPeriod(int period) {
    return 'Periodo: $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'La ventana de desafío está abierta (±$hours h alrededor de la ranura). Puede emitir.';
  }

  @override
  String get mintWindowOff =>
      'La comprobación de la ventana de desafío está desactivada (sandbox). Puede emitir.';

  @override
  String mintWindowClosed(int periods) {
    return 'La ventana de desafío está cerrada. La ranura dentro del mes es impredecible por adelantado; un fallo se traslada dentro de la ventana de acumulación ($periods periodos), los fondos no se bloquean.';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Último periodo reclamado: $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Periodo a reclamar: $period (periodos completados, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Importe a emitir: $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Emisión por periodo: $amount';
  }

  @override
  String get mintConfirmTitle => 'Confirme la emisión';

  @override
  String get mintPowStatus => 'PoW reforzado (Argon2id)…';

  @override
  String get mintNoPeriods => 'No hay periodos que reclamar';

  @override
  String get mintPublished => 'Emisión publicada';

  @override
  String get transferTitle => 'Transferencia';

  @override
  String get segNetwork => 'Por la red';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Dirección del destinatario (hex)';

  @override
  String get recipientHint => '64 caracteres hex';

  @override
  String get amountLabel => 'Importe (el destinatario recibe exactamente esto)';

  @override
  String feeHelper(int fee) {
    return 'Comisión: $fee (además del importe, se quema) — la paga el remitente';
  }

  @override
  String delayedWarn(int hours) {
    return 'El importe supera el umbral P: la transferencia se aplazará. Ejecución en $hours h; se puede cancelar durante el retraso (pantalla «Aplazadas»).';
  }

  @override
  String get btModeHint =>
      'Bluetooth sin conexión: el destinatario verá «recibido por Bluetooth — pendiente de publicación». Hasta la confirmación de la red no es un pago (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Moratoria: las transferencias están disponibles $days días después del registro.';
  }

  @override
  String get transferConfirmTitle => 'Confirme la transferencia';

  @override
  String get btConfirmTitle => 'Confirme el pago por Bluetooth';

  @override
  String get transferSent => 'Transferencia enviada';

  @override
  String transferDelayed(int hours) {
    return 'Transferencia aplazada: ejecución en $hours h, se puede cancelar durante el retraso';
  }

  @override
  String get signNotConfirmed => 'Firma no confirmada';

  @override
  String get btNoDevices => 'No hay dispositivos emparejados';

  @override
  String get btSent =>
      'Enviado por Bluetooth. El destinatario debe esperar la confirmación de la red — hasta la publicación no es un pago';

  @override
  String btError(String error) {
    return 'Bluetooth: $error';
  }

  @override
  String get pendingTitle => 'Transferencias aplazadas';

  @override
  String thresholdP(int value) {
    return 'Umbral P: $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'El aumento a $value surte efecto tras $time';
  }

  @override
  String get newThresholdLabel => 'Nuevo umbral P';

  @override
  String newThresholdHelper(int max) {
    return 'límite P_max = $max; aumento — en 48 h, reducción — inmediata';
  }

  @override
  String get pendingEmpty => 'No hay transferencias aplazadas activas';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'ejecución en $time (comisión $fee)';
  }

  @override
  String get cancelConfirmTitle =>
      'Confirme la cancelación de la transferencia';

  @override
  String get canceledMsg => 'Cancelada';

  @override
  String get invalidThreshold => 'Umbral incorrecto';

  @override
  String get thresholdRaised =>
      'El aumento surte efecto en 48 h (protección contra coacción)';

  @override
  String get thresholdLowered => 'Umbral reducido inmediatamente';

  @override
  String timeHm(int h, int m) {
    return '$h h $m min';
  }

  @override
  String get vouchingTitle => 'Avales';

  @override
  String get segMyStatus => 'Mi estado';

  @override
  String get segVouch => 'Avalar';

  @override
  String get segMySlots => 'Mis ranuras';

  @override
  String get walletActive => 'El monedero está activo';

  @override
  String vouchProgress(int received, int k) {
    return 'Avales: $received de $k';
  }

  @override
  String get vouchFindHelpers =>
      'Encuentre K avaladores en persona: muéstreles el código QR de su dirección en la pestaña «Avalar» (botón «Mi QR»).';

  @override
  String get vouchFlagged =>
      'Marca flagged: nadie puede avalarle (el aval desde un monedero marcado está prohibido, decisión n.º 14)';

  @override
  String get vouchConfirmTitle => 'Confirme el aval';

  @override
  String get vouchPublished => 'Aval publicado';

  @override
  String get btWaiting => 'Esperando dirección por Bluetooth (60 s)…';

  @override
  String get btAddrReceived => 'Dirección recibida por Bluetooth';

  @override
  String get vouchWarning =>
      'El aval en persona (QR o Bluetooth) NO prueba la presencia de la persona. Usted responde de su elección: la ranura es limitada y si se presenta evidence contra el avalado pierde la recompensa (no dinero).';

  @override
  String get vouchOnlyActive =>
      'Solo un monedero active con edad ≥ 3 meses y ≥ 2 periodos de emisión completados puede avalar (7.4).';

  @override
  String get candidateLabel => 'Dirección del candidato (hex)';

  @override
  String get candidateHint => '64 caracteres hex — del QR o Bluetooth';

  @override
  String get myQrHint => 'Mi dirección — muéstrela al avalador';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Ranuras: $occupied ocupadas de $capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Libres: $free. Estadísticas del avalador: avaló a $total, activos $active (reputación pública, 7.4)';
  }

  @override
  String get slotsEmpty => 'Las ranuras están libres';

  @override
  String slotActiveSub(int periods) {
    return 'el candidato está activo: la ranura se libera tras $periods periodos de emisión sin evidence';
  }

  @override
  String slotInactiveSub(int days) {
    return 'el candidato no está activo: la ranura se libera en $days días';
  }

  @override
  String get registryTitle => 'Registro de la red';

  @override
  String registryRowSub(String status, int age) {
    return '$status • edad $age días';
  }

  @override
  String chipAge(int age) {
    return 'edad $age días';
  }

  @override
  String get chipSeed => 'círculo semilla';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'guardián público';

  @override
  String vouchDepthLabel(int depth) {
    return 'Profundidad del grafo de avales: $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Profundidad del grafo de avales: sin enlace con el círculo semilla';

  @override
  String rewardLine(int periods) {
    return 'Recompensa a avaladores: $periods periodos de emisión completados';
  }

  @override
  String asVoucher(int count, int free) {
    return 'Como avalador: por $count (ranuras libres: $free)';
  }

  @override
  String get publicHistory => 'Historial público';

  @override
  String get migrationTitle => 'Migración';

  @override
  String get migrationIntro =>
      'Recuperación del monedero: los guardianes se asignan por adelantado (al crear). Personales — 2 de 3; con públicos — 3 de 5 (los umbrales se combinan).';

  @override
  String migrationTo(String hash) {
    return 'Migración a $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'ejecución en $days días (retraso $delay días); firmas de guardianes: $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Guardianes públicos de la red (guardian_offer, 11)';

  @override
  String get noGuardians => 'Nadie se ha declarado guardián público todavía';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'edad $age días, tutelados: $wards';
  }

  @override
  String get restartIntro =>
      'Si no hay guardianes: el reinicio es un registro normal con clave nueva y K avales nuevos. El monedero antiguo queda en el registro intacto e inaccesible.';

  @override
  String get cancelMigrationTitle =>
      'Confirme la cancelación de la migración (clave antigua)';

  @override
  String get migrationCanceled => 'Migración cancelada';

  @override
  String get restartDialogTitle => 'Pérdida de fondos';

  @override
  String get restartDialogBody =>
      'El reinicio no da nada más que registrar un monedero nuevo: los fondos de la dirección antigua se pierden, el historial no se transfiere. La migración con guardianes transfiere la edad, el historial de emisiones y los avales.';

  @override
  String get restartFlowNote =>
      'Reinicio: borre los datos de la aplicación y repita la creación del monedero (clave nueva, K avales nuevos).';

  @override
  String get startMigrationTitle => 'Inicio de la migración';

  @override
  String get startMigrationIntro =>
      'Está en el dispositivo NUEVO. Introduzca la dirección del monedero antiguo y las firmas de los guardianes (cada guardián firma la declaración en su propia app — pestaña «Aprobar la migración de otro»).';

  @override
  String get oldAddressLabel => 'Dirección antigua (hex)';

  @override
  String get newAddressLabel => 'Dirección nueva (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Firmas recogidas: $count (umbral personal $personal de $personalCount; con públicos — $public de $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Dirección del guardián (hex)';

  @override
  String get guardianSigLabel => 'Firma del guardián (hex)';

  @override
  String get migrationConfirmTitle => 'Confirme la migración';

  @override
  String migrationStarted(int days) {
    return 'Migración iniciada: ejecución en $days días; la clave antigua puede cancelar durante el retraso';
  }

  @override
  String get approveTitle => 'Aprobación de la migración (guardián)';

  @override
  String get approveIntro =>
      'Verifique la identidad del tutelado por sus propios canales. La firma vincula el par de direcciones y el nonce; entréguela al tutelado en persona.';

  @override
  String get nonceLabel => 'Nonce (del tutelado)';

  @override
  String get wardSignTitle =>
      'Confirme la migración del tutelado (usted es el guardián)';

  @override
  String get sigToCopy => 'Firma (entréguela al tutelado):';

  @override
  String get settingsTitle => 'Ajustes';

  @override
  String networkLine(String name) {
    return 'Red: $name';
  }

  @override
  String get paramUnit => 'Unidad';

  @override
  String get paramTransfer => 'Transferencia';

  @override
  String get paramIssuance => 'Emisión';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount cada $months meses';
  }

  @override
  String get paramVouchK => 'K (avales)';

  @override
  String get paramThresholdP => 'Umbral P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Retraso D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours h';
  }

  @override
  String get paramMoratorium => 'Moratoria';

  @override
  String paramMoratoriumValue(String days) {
    return '$days días';
  }

  @override
  String get paramDemurrage => 'Demurrage';

  @override
  String get demurrageOn => 'activado';

  @override
  String get demurrageOff => 'desactivado';

  @override
  String get paramAccrual => 'Ventana de acumulación';

  @override
  String paramAccrualValue(String periods) {
    return '$periods periodos';
  }

  @override
  String get paramGuardians => 'Guardianes';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'personales $pt de $pc; públicos $pubt de $pubc';
  }

  @override
  String get paramMigration => 'Migración';

  @override
  String paramMigrationValue(String days) {
    return 'retraso $days días';
  }

  @override
  String get paramEpoch => 'Época de checkpoint';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds s';
  }

  @override
  String get sandboxNote => 'Sandbox (atestación sintética)';

  @override
  String get guardianOfferTitle => 'Declararse guardián público';

  @override
  String guardianAlready(int wards) {
    return 'Ya es guardián público (tutelados: $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Requisitos: active, edad ≥ $days días';
  }

  @override
  String get guardianConfirmTitle => 'Confirme declararse guardián público';

  @override
  String get guardianPublished => 'Declaración publicada';

  @override
  String get nodeRunning => 'El nodo está funcionando';

  @override
  String get nodeStopped => 'El nodo está detenido';

  @override
  String get nodeLevelHint =>
      'Nivel full — con carga y Wi-Fi, si no light (12.2)';

  @override
  String get lockApp => 'Bloquear la aplicación';

  @override
  String get relockNote => 'Bloqueado. Reinicie la aplicación.';

  @override
  String get languageTitle => 'Idioma';

  @override
  String get languageSystem => 'Sistema';

  @override
  String get languageRu => 'Русский';

  @override
  String get languageEn => 'English';

  @override
  String get languageEs => 'Español';

  @override
  String get languagePt => 'Português';

  @override
  String get languageZh => '中文';
}
