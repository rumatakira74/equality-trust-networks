// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Italian (`it`).
class AppLocalizationsIt extends AppLocalizations {
  AppLocalizationsIt([String locale = 'it']) : super(locale);

  @override
  String get appTitle => 'Reti di fiducia';

  @override
  String get appLabel => 'Reti di fiducia';

  @override
  String get navWallet => 'Portafoglio';

  @override
  String get navVouching => 'Avallo';

  @override
  String get navRegistry => 'Registro';

  @override
  String get navMigration => 'Migrazione';

  @override
  String get navSettings => 'Impostazioni';

  @override
  String get btnNext => 'Avanti';

  @override
  String get btnCancel => 'Annulla';

  @override
  String get btnConfirm => 'Conferma';

  @override
  String get btnRefresh => 'Aggiorna';

  @override
  String get btnSkip => 'Salta';

  @override
  String get btnLogin => 'Sblocca';

  @override
  String get btnFingerprint => 'Impronta';

  @override
  String get btnSend => 'Invia';

  @override
  String get btnSendBt => 'Invia via Bluetooth';

  @override
  String get btnVouch => 'Avallare';

  @override
  String get btnMyQr => 'Il mio QR';

  @override
  String get btnHideQr => 'Nascondi il mio QR';

  @override
  String get btnBtReceive => 'Ricevi via BT';

  @override
  String get btnMint => 'Emissione (sfida)';

  @override
  String get btnMintDo => 'Emetti (PoW pesante)';

  @override
  String get btnTransfer => 'Trasferisci';

  @override
  String get btnPublishAwaiting => 'Pubblica (rete disponibile)';

  @override
  String get btnChangeThreshold => 'Cambia soglia';

  @override
  String get btnCancelTransfer => 'Annulla';

  @override
  String get btnRegister => 'Registra il portafoglio';

  @override
  String get btnStartMigration => 'Avvia la migrazione (nuovo dispositivo)';

  @override
  String get btnApproveMigration =>
      'Approva la migrazione di un altro (sono un guardiano)';

  @override
  String get btnRestart => 'Ripristina il portafoglio';

  @override
  String get btnUnderstandRestart => 'Ho capito, ripristina';

  @override
  String get btnAddSig => 'Aggiungi firma';

  @override
  String get btnPublishProposal => 'Pubblica la proposta';

  @override
  String get btnSignBiometric => 'Firma (biometria)';

  @override
  String get enterPin => 'Inserisca il PIN';

  @override
  String get errWrongPin => 'PIN errato';

  @override
  String get errNoNode => 'Il nodo non è in esecuzione';

  @override
  String get errTxNotFound => 'Transazione non trovata';

  @override
  String lockTooManyAttempts(int seconds) {
    return 'Troppi tentativi falliti. Riprovi tra $seconds s';
  }

  @override
  String get confirmOperation => 'Confermi l\'operazione';

  @override
  String get signDefaultTitle => 'Confermi l\'operazione';

  @override
  String get keyUnlockTitle => 'Sblocca la chiave';

  @override
  String rejectedWithCode(String code) {
    return 'Rifiutato: $code';
  }

  @override
  String errorWith(String e) {
    return 'Errore: $e';
  }

  @override
  String get obTitle => 'Creazione del portafoglio';

  @override
  String get obBioRequired =>
      'Il portafoglio richiede una chiave hardware con biometria (BIOMETRIC_STRONG). Senza un\'impronta configurata la registrazione è impossibile.';

  @override
  String get obBioNotSetup =>
      'Nessuna impronta configurata. Apra la schermata delle impostazioni di sistema e aggiunga un\'impronta.';

  @override
  String get obBioSetup => 'Configura l\'impronta';

  @override
  String get obPinIntro =>
      'Il PIN protegge l\'app. Un PIN di coercizione (facoltativo) apre la modalità esca: la stessa interfaccia, ma un saldo piccolo verosimile; la chiave di firma non viene usata.';

  @override
  String get obPinNormal => 'PIN (normale)';

  @override
  String get obWantDuress => 'Imposta un PIN di coercizione';

  @override
  String get obDuressPin => 'PIN di coercizione';

  @override
  String get obDecoyLegend =>
      'Leggenda della modalità esca (ciò che vedrà chi costringe):';

  @override
  String get obDecoyBalance => 'Saldo verosimile (non 0)';

  @override
  String get obDecoyHistory =>
      'Cronologia (righe: data — importo — destinatario)';

  @override
  String get pinMinDigits => 'minimo 4 cifre';

  @override
  String get duressPinMinDigits => 'PIN di coercizione: minimo 4 cifre';

  @override
  String get pinsMatch => 'I PIN coincidono';

  @override
  String get obGuardiansIntro =>
      'Designi guardiani personali (2 su 3). Potranno approvare la migrazione a un nuovo dispositivo. Aggiungere una nuova impronta invalida la chiave — senza guardiani l\'accesso al portafoglio è perso.';

  @override
  String get obGuardiansEmpty =>
      'Non ci sono ancora altri membri nella rete. Può non designare guardiani per ora, ma l\'offerta è obbligatoria: l\'elenco non si può cambiare dopo (solo alla creazione).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, età $age giorni';
  }

  @override
  String obSelectedCount(int count) {
    return 'Selezionati: $count';
  }

  @override
  String get regStatusReady => 'Pronto per la registrazione';

  @override
  String get regStatusPow => 'Attestazione e PoW (può richiedere tempo)…';

  @override
  String get regConfirmTitle => 'Confermi la registrazione';

  @override
  String get regStatusDone => 'Portafoglio registrato';

  @override
  String pendingCount(int count) {
    return 'Trasferimenti differiti: $count';
  }

  @override
  String get pendingHint =>
      'I trasferimenti grandi si eseguono dopo il ritardo D; possono essere annullati entro D';

  @override
  String get historyTitle => 'Cronologia';

  @override
  String get historyEmpty => 'Ancora niente';

  @override
  String get effBalance => 'saldo effettivo';

  @override
  String get effBalanceDem => 'saldo effettivo (demurrage applicato)';

  @override
  String lockedReserve(int amount) {
    return 'riservato ai differiti: $amount';
  }

  @override
  String get flaggedNotice =>
      'Il portafoglio è contrassegnato (flagged) — il contrassegno è pubblico, nessun verdetto';

  @override
  String get registeredNeedK =>
      'Stato registered: l\'attivazione richiede K avalli (scheda «Avallare»)';

  @override
  String get btAwaitingTitle =>
      'Ricevuto via Bluetooth — in attesa di pubblicazione';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount da $hash';
  }

  @override
  String btStatus(String status) {
    return 'stato: $status';
  }

  @override
  String get txRegister => 'Registrazione del portafoglio';

  @override
  String get txVouch => 'Avallo';

  @override
  String txMint(int period) {
    return 'Emissione (periodo $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Trasferimento $amount → $hash';
  }

  @override
  String get mintTitle => 'Emissione';

  @override
  String mintPeriod(int period) {
    return 'Periodo: $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'La finestra della sfida è aperta (±$hours h intorno allo slot). Può emettere.';
  }

  @override
  String get mintWindowOff =>
      'La verifica della finestra della sfida è disattivata (sandbox). Può emettere.';

  @override
  String mintWindowClosed(int periods) {
    return 'La finestra della sfida è chiusa. Lo slot nel mese è imprevedibile in anticipo; un mancamento viene riportato nella finestra di accumulo ($periods periodi), i fondi non sono bloccati.';
  }

  @override
  String mintWindowJitter(int minutes) {
    return 'La finestra è aperta; il client programma l\'invio tra $minutes min (il jitter attenua il picco di carico).';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Ultimo periodo rivendicato: $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Periodo da rivendicare: $period (periodi completati, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Importo da emettere: $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Emissione per periodo: $amount';
  }

  @override
  String get mintConfirmTitle => 'Confermi l\'emissione';

  @override
  String get mintPowStatus => 'PoW pesante (Argon2id)…';

  @override
  String get mintNoPeriods => 'Nessun periodo da rivendicare';

  @override
  String get mintPublished => 'Emissione pubblicata';

  @override
  String get transferTitle => 'Trasferimento';

  @override
  String get segNetwork => 'Tramite la rete';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Indirizzo del destinatario (hex)';

  @override
  String get recipientHint => '64 caratteri hex';

  @override
  String get amountLabel =>
      'Importo (il destinatario riceve esattamente questo)';

  @override
  String feeHelper(int fee) {
    return 'Commissione: $fee (oltre l\'importo, bruciata) — la paga il mittente';
  }

  @override
  String delayedWarn(int hours) {
    return 'L\'importo supera la soglia P: il trasferimento sarà differito. Esecuzione tra $hours h; può essere annullato durante il ritardo (schermata «Differiti»).';
  }

  @override
  String get btModeHint =>
      'Bluetooth offline: il destinatario vedrà «ricevuto via Bluetooth — in attesa di pubblicazione». Fino alla conferma della rete non è un pagamento (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Moratoria: i trasferimenti sono disponibili $days giorni dopo la registrazione.';
  }

  @override
  String get transferConfirmTitle => 'Confermi il trasferimento';

  @override
  String get btConfirmTitle => 'Confermi il pagamento via Bluetooth';

  @override
  String get transferSent => 'Trasferimento inviato';

  @override
  String get transferAwaiting =>
      'Trasferimento inviato — in attesa di conferma';

  @override
  String transferDelayed(int hours) {
    return 'Trasferimento differito: esecuzione tra $hours h, può essere annullato durante il ritardo';
  }

  @override
  String get signNotConfirmed => 'Firma non confermata';

  @override
  String get btNoDevices => 'Nessun dispositivo associato';

  @override
  String get btSent =>
      'Inviato via Bluetooth. Il destinatario deve attendere la conferma della rete — fino alla pubblicazione non è un pagamento';

  @override
  String btError(String error) {
    return 'Bluetooth: $error';
  }

  @override
  String get pendingTitle => 'Trasferimenti differiti';

  @override
  String thresholdP(int value) {
    return 'Soglia P: $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'L\'aumento a $value entra in vigore dopo $time';
  }

  @override
  String get newThresholdLabel => 'Nuova soglia P';

  @override
  String newThresholdHelper(int max) {
    return 'limite P_max = $max; aumento — dopo 48 h, riduzione — immediata';
  }

  @override
  String get pendingEmpty => 'Nessun trasferimento differito attivo';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'esecuzione tra $time (commissione $fee)';
  }

  @override
  String get cancelConfirmTitle => 'Confermi l\'annullamento del trasferimento';

  @override
  String get canceledMsg => 'Annullato';

  @override
  String get invalidThreshold => 'Soglia non valida';

  @override
  String get thresholdRaised =>
      'L\'aumento entra in vigore tra 48 h (protezione dalla coercizione)';

  @override
  String get thresholdLowered => 'Soglia ridotta immediatamente';

  @override
  String timeHm(int h, int m) {
    return '$h h $m min';
  }

  @override
  String get vouchingTitle => 'Avallo';

  @override
  String get segMyStatus => 'Il mio stato';

  @override
  String get segVouch => 'Avallare';

  @override
  String get segMySlots => 'I miei slot';

  @override
  String get walletActive => 'Il portafoglio è attivo';

  @override
  String vouchProgress(int received, int k) {
    return 'Avalli: $received su $k';
  }

  @override
  String get vouchFindHelpers =>
      'Trovi K avallatori di persona: mostri loro il codice QR del suo indirizzo nella scheda «Avallare» (pulsante «Il mio QR»).';

  @override
  String get vouchFlagged =>
      'Contrassegno flagged: nessuno può avallarla (l\'avallo da un portafoglio contrassegnato è vietato, decisione n. 14)';

  @override
  String get vouchConfirmTitle => 'Confermi l\'avallo';

  @override
  String get vouchPublished => 'Avallo pubblicato';

  @override
  String get btWaiting => 'In attesa dell\'indirizzo via Bluetooth (60 s)…';

  @override
  String get btAddrReceived => 'Indirizzo ricevuto via Bluetooth';

  @override
  String get vouchWarning =>
      'L\'avallo di persona (QR o Bluetooth) NON prova la presenza della persona. Lei risponde della scelta: lo slot è limitato e, se viene presentata una prova contro l\'avallato, perde la ricompensa (non denaro).';

  @override
  String get vouchOnlyActive =>
      'Solo un portafoglio active con età ≥ 3 mesi e ≥ 2 periodi di emissione completati può avallare (7.4).';

  @override
  String get candidateLabel => 'Indirizzo del candidato (hex)';

  @override
  String get candidateHint => '64 caratteri hex — dal QR o dal Bluetooth';

  @override
  String get myQrHint => 'Il mio indirizzo — lo mostri all\'avallatore';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Slot: $occupied occupati su $capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Liberi: $free. Statistiche dell\'avallatore: ha avallato $total, attivi $active (reputazione pubblica, 7.4)';
  }

  @override
  String get slotsEmpty => 'Gli slot sono liberi';

  @override
  String slotActiveSub(int periods) {
    return 'il candidato è attivo: lo slot si libera dopo $periods periodi di emissione senza prova';
  }

  @override
  String slotInactiveSub(int days) {
    return 'il candidato non è attivo: lo slot si libera tra $days giorni';
  }

  @override
  String get registryTitle => 'Registro della rete';

  @override
  String registryRowSub(String status, int age) {
    return '$status • età $age giorni';
  }

  @override
  String chipAge(int age) {
    return 'età $age giorni';
  }

  @override
  String get chipSeed => 'cerchio iniziale';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'guardiano pubblico';

  @override
  String vouchDepthLabel(int depth) {
    return 'Profondità del grafo degli avalli: $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Profondità del grafo degli avalli: nessun collegamento con il cerchio iniziale';

  @override
  String rewardLine(int periods) {
    return 'Ricompensa agli avallatori: $periods periodi di emissione completati';
  }

  @override
  String asVoucher(int count, int free) {
    return 'Come avallatore: per $count (slot liberi: $free)';
  }

  @override
  String get publicHistory => 'Cronologia pubblica';

  @override
  String get migrationTitle => 'Migrazione';

  @override
  String get migrationIntro =>
      'Recupero del portafoglio: i guardiani sono designati in anticipo (alla creazione). Personali — 2 su 3; con i pubblici — 3 su 5 (le soglie si combinano).';

  @override
  String migrationTo(String hash) {
    return 'Migrazione verso $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'esecuzione tra $days giorni (ritardo $delay giorni); firme dei guardiani: $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Guardiani pubblici della rete (guardian_offer, 11)';

  @override
  String get noGuardians => 'Nessuno si è ancora dichiarato guardiano pubblico';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'età $age giorni, tutelati: $wards';
  }

  @override
  String get restartIntro =>
      'Se non ci sono guardiani: il ripristino è una registrazione ordinaria con una chiave nuova e K nuovi avalli. Il vecchio portafoglio resta nel registro intatto e inaccessibile.';

  @override
  String get cancelMigrationTitle =>
      'Confermi l\'annullamento della migrazione (chiave vecchia)';

  @override
  String get migrationCanceled => 'Migrazione annullata';

  @override
  String get restartDialogTitle => 'Perdita di fondi';

  @override
  String get restartDialogBody =>
      'Il ripristino non dà nulla oltre alla registrazione di un nuovo portafoglio: i fondi del vecchio indirizzo sono perduti, la cronologia non è trasferita. La migrazione con guardiani trasferisce l\'età, la cronologia delle emissioni e gli avalli.';

  @override
  String get restartFlowNote =>
      'Ripristino: cancelli i dati dell\'applicazione e rifaccia la creazione del portafoglio (chiave nuova, K nuovi avalli).';

  @override
  String get startMigrationTitle => 'Avvio della migrazione';

  @override
  String get startMigrationIntro =>
      'Lei è sul dispositivo NUOVO. Inserisca il vecchio indirizzo del portafoglio e le firme dei guardiani (ogni guardiano firma la dichiarazione nella sua app — scheda «Approva la migrazione di un altro»).';

  @override
  String get oldAddressLabel => 'Vecchio indirizzo (hex)';

  @override
  String get newAddressLabel => 'Nuovo indirizzo (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Firme raccolte: $count (soglia personale $personal su $personalCount; con i pubblici — $public su $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Indirizzo del guardiano (hex)';

  @override
  String get guardianSigLabel => 'Firma del guardiano (hex)';

  @override
  String get migrationConfirmTitle => 'Confermi la migrazione';

  @override
  String migrationStarted(int days) {
    return 'Migrazione avviata: esecuzione tra $days giorni; la chiave vecchia può annullare durante il ritardo';
  }

  @override
  String get approveTitle => 'Approvazione della migrazione (guardiano)';

  @override
  String get approveIntro =>
      'Verifichi l\'identità del tutelato attraverso i propri canali. La firma vincola la coppia di indirizzi e il nonce; la consegni al tutelato di persona.';

  @override
  String get nonceLabel => 'Nonce (del tutelato)';

  @override
  String get wardSignTitle =>
      'Confermi la migrazione del tutelato (Lei è il guardiano)';

  @override
  String get sigToCopy => 'Firma (la consegni al tutelato):';

  @override
  String get nonceToCopy => 'Nonce di migrazione (lo consegni ai guardiani):';

  @override
  String get settingsTitle => 'Impostazioni';

  @override
  String networkLine(String name) {
    return 'Rete: $name';
  }

  @override
  String get paramUnit => 'Unità';

  @override
  String get paramTransfer => 'Trasferimento';

  @override
  String get paramIssuance => 'Emissione';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount ogni $months mesi';
  }

  @override
  String get paramVouchK => 'K (avalli)';

  @override
  String get paramThresholdP => 'Soglia P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Ritardo D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours h';
  }

  @override
  String get paramMoratorium => 'Moratoria';

  @override
  String paramMoratoriumValue(String days) {
    return '$days giorni';
  }

  @override
  String get paramDemurrage => 'Demurrage';

  @override
  String get demurrageOn => 'attivato';

  @override
  String get demurrageOff => 'disattivato';

  @override
  String get paramAccrual => 'Finestra di accumulo';

  @override
  String paramAccrualValue(String periods) {
    return '$periods periodi';
  }

  @override
  String get paramGuardians => 'Guardiani';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'personali $pt su $pc; pubblici $pubt su $pubc';
  }

  @override
  String get paramMigration => 'Migrazione';

  @override
  String paramMigrationValue(String days) {
    return 'ritardo $days giorni';
  }

  @override
  String get paramEpoch => 'Epoca di checkpoint';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds s';
  }

  @override
  String get sandboxNote => 'Sandbox (attestazione sintetica)';

  @override
  String get guardianOfferTitle => 'Dichiararsi guardiano pubblico';

  @override
  String guardianAlready(int wards) {
    return 'Lei è già guardiano pubblico (tutelati: $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Requisiti: active, età ≥ $days giorni';
  }

  @override
  String get guardianConfirmTitle =>
      'Confermi di dichiararsi guardiano pubblico';

  @override
  String get guardianPublished => 'Dichiarazione pubblicata';

  @override
  String get nodeRunning => 'Il nodo è in funzione';

  @override
  String get nodeStopped => 'Il nodo è fermo';

  @override
  String get nodeChannelName => 'Nodo di rete';

  @override
  String get nodeNotificationTitle => 'Reti di fiducia';

  @override
  String get nodeNotificationText =>
      'Il nodo della rete di fiducia è in funzione';

  @override
  String get nodeLevelHint =>
      'Livello full — in carica e con Wi-Fi, altrimenti light (12.2)';

  @override
  String get lockApp => 'Blocca l\'applicazione';

  @override
  String get relockNote => 'Bloccato. Riavvii l\'applicazione.';

  @override
  String get languageTitle => 'Lingua';

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

  @override
  String get languageFr => 'Français';

  @override
  String get languageDe => 'Deutsch';

  @override
  String get languageIt => 'Italiano';

  @override
  String get languageJa => '日本語';

  @override
  String get recoveryTitle => 'Scheda di recupero';

  @override
  String get recoveryIntro =>
      'Conservi questi dati fuori dal telefono: servono per recuperare il portafoglio se il dispositivo è perso (migrazione con guardiani, sezione 11).';

  @override
  String get recoveryAddress => 'Indirizzo del portafoglio';

  @override
  String get recoveryNetwork => 'Rete';

  @override
  String get recoveryGuardians => 'I suoi guardiani';

  @override
  String get recoveryNoGuardians =>
      'Nessun guardiano designato: il recupero è impossibile, resta solo il ripristino (registrazione ordinaria con nuovi avalli).';

  @override
  String get recoveryHowToSave =>
      'Come conservare: lo copri su carta o lo custodisca in un luogo sicuro fuori dal telefono; non lo conservi nel cloud. L\'immagine della scheda può essere stampata tramite «Condividi».';

  @override
  String get recoveryNeverShare =>
      'Non condivida mai i dati di migrazione (il nonce e le firme dei guardiani) — solo direttamente al suo guardiano e solo durante la migrazione; li cancelli dai messaggi dopo. A terzi mostri solo l\'indirizzo e il QR di questa scheda.';

  @override
  String get recoverySavedDone => 'Salvato — continuare';

  @override
  String get settingsRecovery => 'Scheda di recupero';

  @override
  String get recoveryApkNote =>
      'Conservi in anticipo il file di installazione (APK) della stessa rete: senza di esso il recupero su un nuovo dispositivo può essere impossibile.';

  @override
  String get recoveryShareBtn => 'Condividi la scheda (immagine)';

  @override
  String get guardiansTitle => 'Guardiani';

  @override
  String get guardiansIntro =>
      'I guardiani personali confermano la migrazione (2 su 3). Possono essere rieletti: il cambio entra in vigore tra 48 h; fino ad allora vale l\'elenco precedente.';

  @override
  String get guardiansEntrySub =>
      'Guardiani personali: rielezione, finestra di 48 h';

  @override
  String get guardiansCurrent => 'Guardiani attuali';

  @override
  String get guardiansNone => 'Nessun guardiano designato';

  @override
  String get guardiansReelect => 'Rieleggi';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return 'Cambi di guardiano usati: $used di $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return 'Il nuovo elenco entrerà in vigore tra $hours h';
  }

  @override
  String get guardiansCancelBtn => 'Annulla il cambio';

  @override
  String get guardiansStatusCanceled => 'Annullato';

  @override
  String get guardiansStatusEffective => 'In vigore';

  @override
  String get guardiansStatusExpired =>
      'Scaduto: superato il limite di tutelati del guardiano';

  @override
  String get reelectTitle => 'Rieleggere i guardiani';

  @override
  String get reelectIntro =>
      'Inserisca i nuovi guardiani (fino a 3, hex, uno per riga). Passi l\'hash del cambio a ogni nuovo guardiano: lo firma nella sua app («Consenso del guardiano») e le restituisce l\'hex della firma.';

  @override
  String get newGuardiansLabel => 'Nuovi guardiani (hex, uno per riga)';

  @override
  String get changeHashToCopy =>
      'Hash del cambio (lo passi ai nuovi guardiani):';

  @override
  String guardiansApprovals(int count, int need) {
    return 'Consensi raccolti: $count di $need';
  }

  @override
  String get btnPublishChange => 'Pubblica il cambio';

  @override
  String get changePublished => 'Cambio pubblicato: entra in vigore tra 48 h';

  @override
  String get changeConfirmTitle => 'Confermi il cambio di guardiani (PIN)';

  @override
  String get cancelChangeConfirmTitle =>
      'Confermi l\'annullamento del cambio (PIN)';

  @override
  String get changeCanceled => 'Cambio annullato';

  @override
  String get consentTitle => 'Consenso del guardiano';

  @override
  String get consentIntro =>
      'Lei conferma di accettare di essere il guardiano del tutelato. Inserisca l\'indirizzo del tutelato e l\'hash del cambio e firmi l\'hash. Restituisca l\'hex della firma al tutelato.';

  @override
  String get wardAddressLabel => 'Indirizzo del tutelato (hex)';

  @override
  String get changeHashLabel => 'Hash del cambio (hex)';

  @override
  String get consentSignTitle =>
      'Confermi il cambio di guardiani del tutelato (Lei è il guardiano)';

  @override
  String get biometricTimeout =>
      'La biometria non risponde (tempo scaduto) — riprovi';

  @override
  String hexChunksCount(int n, int total) {
    return 'Inseriti $n di $total';
  }

  @override
  String hexChunksSigCount(int n) {
    return 'Caratteri inseriti: $n';
  }

  @override
  String hexLenError(int total) {
    return 'Attesi esattamente $total caratteri hex';
  }

  @override
  String get hexSigEvenError =>
      'La firma deve avere un numero pari di caratteri hex';

  @override
  String get hexCharsDropped => 'Caratteri non validi rimossi (solo 0-9, a-f)';

  @override
  String get aboutNetwork => 'Informazioni sulla rete';

  @override
  String networkSlug(String slug) {
    return 'slug: $slug';
  }

  @override
  String get paramRecurring => 'Unità periodiche (recurring)';

  @override
  String get recurringOn => 'attivate (estinzione al limite del periodo)';

  @override
  String get recurringOff => 'disattivate';

  @override
  String get recurringNote =>
      'Ogni unità porta il contrassegno del periodo: il trasferimento di un\'unità del periodo P è rifiutato dopo il limite del periodo P+1. Il saldo non è azzerato — la cronologia resta pubblica e verificabile.';

  @override
  String get burnedUnitError =>
      'L\'unità è estinta: il limite del periodo P+1 è passato. Il trasferimento e il riscatto sono impossibili, ma il saldo non è azzerato — la cronologia è conservata (E_BURNED).';

  @override
  String obNetworkLine(String pair) {
    return 'Rete: $pair';
  }
}
