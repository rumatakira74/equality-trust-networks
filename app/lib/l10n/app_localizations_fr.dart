// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for French (`fr`).
class AppLocalizationsFr extends AppLocalizations {
  AppLocalizationsFr([String locale = 'fr']) : super(locale);

  @override
  String get appTitle => 'Réseaux de confiance';

  @override
  String get navWallet => 'Portefeuille';

  @override
  String get navVouching => 'Parrainage';

  @override
  String get navRegistry => 'Registre';

  @override
  String get navMigration => 'Migration';

  @override
  String get navSettings => 'Réglages';

  @override
  String get btnNext => 'Suivant';

  @override
  String get btnCancel => 'Annuler';

  @override
  String get btnConfirm => 'Confirmer';

  @override
  String get btnRefresh => 'Actualiser';

  @override
  String get btnSkip => 'Passer';

  @override
  String get btnLogin => 'Déverrouiller';

  @override
  String get btnFingerprint => 'Empreinte';

  @override
  String get btnSend => 'Envoyer';

  @override
  String get btnSendBt => 'Envoyer par Bluetooth';

  @override
  String get btnVouch => 'Parrainer';

  @override
  String get btnMyQr => 'Mon QR';

  @override
  String get btnHideQr => 'Masquer mon QR';

  @override
  String get btnBtReceive => 'Recevoir par BT';

  @override
  String get btnMint => 'Émission (défi)';

  @override
  String get btnMintDo => 'Émettre (PoW renforcé)';

  @override
  String get btnTransfer => 'Transférer';

  @override
  String get btnPublishAwaiting => 'Publier (réseau disponible)';

  @override
  String get btnChangeThreshold => 'Changer le seuil';

  @override
  String get btnCancelTransfer => 'Annuler';

  @override
  String get btnRegister => 'Enregistrer le portefeuille';

  @override
  String get btnStartMigration => 'Démarrer la migration (nouvel appareil)';

  @override
  String get btnApproveMigration =>
      'Approuver la migration d\'autrui (je suis gardien)';

  @override
  String get btnRestart => 'Réinitialiser le portefeuille';

  @override
  String get btnUnderstandRestart => 'Je comprends, réinitialiser';

  @override
  String get btnAddSig => 'Ajouter une signature';

  @override
  String get btnPublishProposal => 'Publier la proposition';

  @override
  String get btnSignBiometric => 'Signer (biométrie)';

  @override
  String get enterPin => 'Saisissez le PIN';

  @override
  String get errWrongPin => 'PIN incorrect';

  @override
  String get errNoNode => 'Le nœud n\'est pas démarré';

  @override
  String lockTooManyAttempts(int seconds) {
    return 'Trop de tentatives échouées. Réessayez dans $seconds s';
  }

  @override
  String get confirmOperation => 'Confirmez l\'opération';

  @override
  String get keyUnlockTitle => 'Déverrouiller la clé';

  @override
  String rejectedWithCode(String code) {
    return 'Rejeté : $code';
  }

  @override
  String errorWith(String e) {
    return 'Erreur : $e';
  }

  @override
  String get obTitle => 'Création du portefeuille';

  @override
  String get obBioRequired =>
      'Le portefeuille exige une clé matérielle avec biométrie (BIOMETRIC_STRONG). Sans empreinte configurée, l\'enregistrement est impossible.';

  @override
  String get obBioNotSetup =>
      'Aucune empreinte configurée. Ouvrez l\'écran des réglages du système et ajoutez une empreinte.';

  @override
  String get obBioSetup => 'Configurer l\'empreinte';

  @override
  String get obPinIntro =>
      'Le PIN protège l\'application. Un PIN de contrainte (facultatif) ouvre le mode leurre : la même interface, mais un solde réduit plausible ; la clé de signature n\'est pas utilisée.';

  @override
  String get obPinNormal => 'PIN (normal)';

  @override
  String get obWantDuress => 'Définir un PIN de contrainte';

  @override
  String get obDuressPin => 'PIN de contrainte';

  @override
  String get obDecoyLegend =>
      'Légende du mode leurre (ce que verra la personne qui vous contraint) :';

  @override
  String get obDecoyBalance => 'Solde plausible (pas 0)';

  @override
  String get obDecoyHistory =>
      'Historique (lignes : date — montant — destinataire)';

  @override
  String get pinMinDigits => 'minimum 4 chiffres';

  @override
  String get duressPinMinDigits => 'PIN de contrainte : minimum 4 chiffres';

  @override
  String get pinsMatch => 'Les PIN correspondent';

  @override
  String get obGuardiansIntro =>
      'Désignez des gardiens personnels (2 sur 3). Ils pourront approuver la migration vers un nouvel appareil. Ajouter une nouvelle empreinte invalide la clé — sans gardiens, l\'accès au portefeuille est perdu.';

  @override
  String get obGuardiansEmpty =>
      'Il n\'y a pas encore d\'autres membres dans le réseau. Vous pouvez ne pas désigner de gardiens pour l\'instant, mais l\'offre est obligatoire : la liste ne peut plus être modifiée ensuite (uniquement à la création).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, âge $age jours';
  }

  @override
  String obSelectedCount(int count) {
    return 'Sélectionnés : $count';
  }

  @override
  String get regStatusReady => 'Prêt à enregistrer';

  @override
  String get regStatusPow => 'Attestation et PoW (peut prendre du temps)…';

  @override
  String get regConfirmTitle => 'Confirmez l\'enregistrement';

  @override
  String get regStatusDone => 'Portefeuille enregistré';

  @override
  String pendingCount(int count) {
    return 'Transferts différés : $count';
  }

  @override
  String get pendingHint =>
      'Les transferts importants s\'exécutent après le délai D ; ils peuvent être annulés pendant D';

  @override
  String get historyTitle => 'Historique';

  @override
  String get historyEmpty => 'Rien pour l\'instant';

  @override
  String get effBalance => 'solde effectif';

  @override
  String get effBalanceDem => 'solde effectif (demurrage appliqué)';

  @override
  String lockedReserve(int amount) {
    return 'réservé aux différés : $amount';
  }

  @override
  String get flaggedNotice =>
      'Le portefeuille est marqué (flagged) — la marque est publique, aucun verdict';

  @override
  String get registeredNeedK =>
      'État registered : l\'activation exige K parrainages (onglet « Parrainage »)';

  @override
  String get btAwaitingTitle =>
      'Reçu par Bluetooth — en attente de publication';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount de $hash';
  }

  @override
  String btStatus(String status) {
    return 'état : $status';
  }

  @override
  String get txRegister => 'Enregistrement du portefeuille';

  @override
  String get txVouch => 'Parrainage';

  @override
  String txMint(int period) {
    return 'Émission (période $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Transfert $amount → $hash';
  }

  @override
  String get mintTitle => 'Émission';

  @override
  String mintPeriod(int period) {
    return 'Période : $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'La fenêtre du défi est ouverte (±$hours h autour du créneau). Vous pouvez émettre.';
  }

  @override
  String get mintWindowOff =>
      'La vérification de la fenêtre du défi est désactivée (sandbox). Vous pouvez émettre.';

  @override
  String mintWindowClosed(int periods) {
    return 'La fenêtre du défi est fermée. Le créneau dans le mois est imprévisible à l\'avance ; un manquement est reporté dans la fenêtre d\'accumulation ($periods périodes), les fonds ne sont pas bloqués.';
  }

  @override
  String mintWindowJitter(int minutes) {
    return 'La fenêtre est ouverte ; le client programme l\'envoi dans $minutes min (le jitter lisse le pic de charge).';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Dernière période réclamée : $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Période à réclamer : $period (périodes achevées, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Montant à émettre : $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Émission par période : $amount';
  }

  @override
  String get mintConfirmTitle => 'Confirmez l\'émission';

  @override
  String get mintPowStatus => 'PoW renforcé (Argon2id)…';

  @override
  String get mintNoPeriods => 'Aucune période à réclamer';

  @override
  String get mintPublished => 'Émission publiée';

  @override
  String get transferTitle => 'Transfert';

  @override
  String get segNetwork => 'Par le réseau';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Adresse du destinataire (hex)';

  @override
  String get recipientHint => '64 caractères hex';

  @override
  String get amountLabel => 'Montant (le destinataire reçoit exactement cela)';

  @override
  String feeHelper(int fee) {
    return 'Frais : $fee (en plus du montant, brûlés) — payés par l\'expéditeur';
  }

  @override
  String delayedWarn(int hours) {
    return 'Le montant dépasse le seuil P : le transfert sera différé. Exécution dans $hours h ; il peut être annulé pendant le délai (écran « Différés »).';
  }

  @override
  String get btModeHint =>
      'Bluetooth hors ligne : le destinataire verra « reçu par Bluetooth — en attente de publication ». Jusqu\'à la confirmation du réseau, ce n\'est pas un paiement (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Moratoire : les transferts sont disponibles $days jours après l\'enregistrement.';
  }

  @override
  String get transferConfirmTitle => 'Confirmez le transfert';

  @override
  String get btConfirmTitle => 'Confirmez le paiement par Bluetooth';

  @override
  String get transferSent => 'Transfert envoyé';

  @override
  String get transferAwaiting =>
      'Transfert envoyé — en attente de confirmation';

  @override
  String transferDelayed(int hours) {
    return 'Transfert différé : exécution dans $hours h, il peut être annulé pendant le délai';
  }

  @override
  String get signNotConfirmed => 'Signature non confirmée';

  @override
  String get btNoDevices => 'Aucun appareil apparié';

  @override
  String get btSent =>
      'Envoyé par Bluetooth. Le destinataire doit attendre la confirmation du réseau — jusqu\'à la publication, ce n\'est pas un paiement';

  @override
  String btError(String error) {
    return 'Bluetooth : $error';
  }

  @override
  String get pendingTitle => 'Transferts différés';

  @override
  String thresholdP(int value) {
    return 'Seuil P : $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'L\'augmentation à $value prend effet après $time';
  }

  @override
  String get newThresholdLabel => 'Nouveau seuil P';

  @override
  String newThresholdHelper(int max) {
    return 'limite P_max = $max ; augmentation — après 48 h, réduction — immédiate';
  }

  @override
  String get pendingEmpty => 'Aucun transfert différé actif';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'exécution dans $time (frais $fee)';
  }

  @override
  String get cancelConfirmTitle => 'Confirmez l\'annulation du transfert';

  @override
  String get canceledMsg => 'Annulé';

  @override
  String get invalidThreshold => 'Seuil incorrect';

  @override
  String get thresholdRaised =>
      'L\'augmentation prend effet dans 48 h (protection contre la contrainte)';

  @override
  String get thresholdLowered => 'Seuil réduit immédiatement';

  @override
  String timeHm(int h, int m) {
    return '$h h $m min';
  }

  @override
  String get vouchingTitle => 'Parrainage';

  @override
  String get segMyStatus => 'Mon état';

  @override
  String get segVouch => 'Parrainer';

  @override
  String get segMySlots => 'Mes emplacements';

  @override
  String get walletActive => 'Le portefeuille est actif';

  @override
  String vouchProgress(int received, int k) {
    return 'Parrainages : $received sur $k';
  }

  @override
  String get vouchFindHelpers =>
      'Trouvez K parrains en personne : montrez-leur le code QR de votre adresse dans l\'onglet « Parrainer » (bouton « Mon QR »).';

  @override
  String get vouchFlagged =>
      'Marque flagged : personne ne peut vous parrainer (le parrainage depuis un portefeuille marqué est interdit, décision n° 14)';

  @override
  String get vouchConfirmTitle => 'Confirmez le parrainage';

  @override
  String get vouchPublished => 'Parrainage publié';

  @override
  String get btWaiting => 'En attente de l\'adresse par Bluetooth (60 s)…';

  @override
  String get btAddrReceived => 'Adresse reçue par Bluetooth';

  @override
  String get vouchWarning =>
      'Le parrainage en personne (QR ou Bluetooth) ne prouve PAS la présence de la personne. Vous répondez de votre choix : l\'emplacement est limité et, si une preuve est présentée contre le parrainé, vous perdez la récompense (pas d\'argent).';

  @override
  String get vouchOnlyActive =>
      'Seul un portefeuille active âgé d\'au moins 3 mois et comptant au moins 2 périodes d\'émission achevées peut parrainer (7.4).';

  @override
  String get candidateLabel => 'Adresse du candidat (hex)';

  @override
  String get candidateHint => '64 caractères hex — du QR ou du Bluetooth';

  @override
  String get myQrHint => 'Mon adresse — montrez-la au parrain';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Emplacements : $occupied occupés sur $capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Libres : $free. Statistiques du parrain : a parrainé $total, actifs $active (réputation publique, 7.4)';
  }

  @override
  String get slotsEmpty => 'Les emplacements sont libres';

  @override
  String slotActiveSub(int periods) {
    return 'le candidat est actif : l\'emplacement se libère après $periods périodes d\'émission sans preuve';
  }

  @override
  String slotInactiveSub(int days) {
    return 'le candidat n\'est pas actif : l\'emplacement se libère dans $days jours';
  }

  @override
  String get registryTitle => 'Registre du réseau';

  @override
  String registryRowSub(String status, int age) {
    return '$status • âge $age jours';
  }

  @override
  String chipAge(int age) {
    return 'âge $age jours';
  }

  @override
  String get chipSeed => 'cercle initial';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'gardien public';

  @override
  String vouchDepthLabel(int depth) {
    return 'Profondeur du graphe de parrainages : $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Profondeur du graphe de parrainages : sans lien avec le cercle initial';

  @override
  String rewardLine(int periods) {
    return 'Récompense aux parrains : $periods périodes d\'émission achevées';
  }

  @override
  String asVoucher(int count, int free) {
    return 'Comme parrain : pour $count (emplacements libres : $free)';
  }

  @override
  String get publicHistory => 'Historique public';

  @override
  String get migrationTitle => 'Migration';

  @override
  String get migrationIntro =>
      'Récupération du portefeuille : les gardiens sont désignés à l\'avance (à la création). Personnels — 2 sur 3 ; avec des publics — 3 sur 5 (les seuils se combinent).';

  @override
  String migrationTo(String hash) {
    return 'Migration vers $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'exécution dans $days jours (délai $delay jours) ; signatures des gardiens : $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Gardiens publics du réseau (guardian_offer, 11)';

  @override
  String get noGuardians => 'Personne ne s\'est encore déclaré gardien public';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'âge $age jours, pupilles : $wards';
  }

  @override
  String get restartIntro =>
      'S\'il n\'y a pas de gardiens : la réinitialisation est un enregistrement ordinaire avec une clé nouvelle et K nouveaux parrainages. L\'ancien portefeuille reste dans le registre, intact et inaccessible.';

  @override
  String get cancelMigrationTitle =>
      'Confirmez l\'annulation de la migration (ancienne clé)';

  @override
  String get migrationCanceled => 'Migration annulée';

  @override
  String get restartDialogTitle => 'Perte de fonds';

  @override
  String get restartDialogBody =>
      'La réinitialisation n\'apporte rien de plus que l\'enregistrement d\'un nouveau portefeuille : les fonds de l\'ancienne adresse sont perdus, l\'historique n\'est pas transféré. La migration avec gardiens transfère l\'âge, l\'historique des émissions et les parrainages.';

  @override
  String get restartFlowNote =>
      'Réinitialisation : effacez les données de l\'application et refaites la création du portefeuille (clé nouvelle, K nouveaux parrainages).';

  @override
  String get startMigrationTitle => 'Démarrage de la migration';

  @override
  String get startMigrationIntro =>
      'Vous êtes sur le NOUVEL appareil. Saisissez l\'ancienne adresse du portefeuille et les signatures des gardiens (chaque gardien signe la déclaration dans sa propre application — onglet « Approuver la migration d\'autrui »).';

  @override
  String get oldAddressLabel => 'Ancienne adresse (hex)';

  @override
  String get newAddressLabel => 'Nouvelle adresse (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Signatures recueillies : $count (seuil personnel $personal sur $personalCount ; avec des publics — $public sur $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Adresse du gardien (hex)';

  @override
  String get guardianSigLabel => 'Signature du gardien (hex)';

  @override
  String get migrationConfirmTitle => 'Confirmez la migration';

  @override
  String migrationStarted(int days) {
    return 'Migration démarrée : exécution dans $days jours ; l\'ancienne clé peut annuler pendant le délai';
  }

  @override
  String get approveTitle => 'Approbation de la migration (gardien)';

  @override
  String get approveIntro =>
      'Vérifiez l\'identité du pupille par vos propres canaux. La signature lie la paire d\'adresses et le nonce ; remettez-la au pupille en personne.';

  @override
  String get nonceLabel => 'Nonce (du pupille)';

  @override
  String get wardSignTitle =>
      'Confirmez la migration du pupille (vous êtes le gardien)';

  @override
  String get sigToCopy => 'Signature (remettez-la au pupille) :';

  @override
  String get nonceToCopy => 'Nonce de migration (remettez-le aux gardiens) :';

  @override
  String get settingsTitle => 'Réglages';

  @override
  String networkLine(String name) {
    return 'Réseau : $name';
  }

  @override
  String get paramUnit => 'Unité';

  @override
  String get paramTransfer => 'Transfert';

  @override
  String get paramIssuance => 'Émission';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount tous les $months mois';
  }

  @override
  String get paramVouchK => 'K (parrainages)';

  @override
  String get paramThresholdP => 'Seuil P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Délai D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours h';
  }

  @override
  String get paramMoratorium => 'Moratoire';

  @override
  String paramMoratoriumValue(String days) {
    return '$days jours';
  }

  @override
  String get paramDemurrage => 'Demurrage';

  @override
  String get demurrageOn => 'activé';

  @override
  String get demurrageOff => 'désactivé';

  @override
  String get paramAccrual => 'Fenêtre d\'accumulation';

  @override
  String paramAccrualValue(String periods) {
    return '$periods périodes';
  }

  @override
  String get paramGuardians => 'Gardiens';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'personnels $pt sur $pc ; publics $pubt sur $pubc';
  }

  @override
  String get paramMigration => 'Migration';

  @override
  String paramMigrationValue(String days) {
    return 'délai $days jours';
  }

  @override
  String get paramEpoch => 'Époque de checkpoint';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds s';
  }

  @override
  String get sandboxNote => 'Sandbox (attestation synthétique)';

  @override
  String get guardianOfferTitle => 'Se déclarer gardien public';

  @override
  String guardianAlready(int wards) {
    return 'Vous êtes déjà gardien public (pupilles : $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Conditions requises : active, âge ≥ $days jours';
  }

  @override
  String get guardianConfirmTitle =>
      'Confirmez votre déclaration de gardien public';

  @override
  String get guardianPublished => 'Déclaration publiée';

  @override
  String get nodeRunning => 'Le nœud fonctionne';

  @override
  String get nodeStopped => 'Le nœud est arrêté';

  @override
  String get nodeLevelHint =>
      'Niveau full — en charge et en Wi-Fi, sinon light (12.2)';

  @override
  String get lockApp => 'Verrouiller l\'application';

  @override
  String get relockNote => 'Verrouillé. Redémarrez l\'application.';

  @override
  String get languageTitle => 'Langue';

  @override
  String get languageSystem => 'Système';

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
  String get recoveryTitle => 'Carte de récupération';

  @override
  String get recoveryIntro =>
      'Conservez ces données hors du téléphone : elles sont nécessaires pour récupérer le portefeuille en cas de perte de l\'appareil (migration avec gardiens, section 11).';

  @override
  String get recoveryAddress => 'Adresse du portefeuille';

  @override
  String get recoveryNetwork => 'Réseau';

  @override
  String get recoveryGuardians => 'Vos gardiens';

  @override
  String get recoveryNoGuardians =>
      'Aucun gardien désigné : la récupération est impossible, seule reste la réinitialisation (enregistrement ordinaire avec de nouveaux parrainages).';

  @override
  String get recoveryHowToSave =>
      'Comment conserver : copiez-le sur papier ou gardez-le dans un lieu sûr hors du téléphone ; ne le stockez pas dans le cloud. L\'image de la carte peut être imprimée via « Partager ».';

  @override
  String get recoveryNeverShare =>
      'Ne partagez jamais les données de migration (le nonce et les signatures des gardiens) — uniquement directement à votre gardien et seulement pendant la migration ; supprimez-les de vos messages ensuite. À des tiers, ne montrez que l\'adresse et le QR de cette carte.';

  @override
  String get recoverySavedDone => 'Enregistré — continuer';

  @override
  String get settingsRecovery => 'Carte de récupération';

  @override
  String get recoveryApkNote =>
      'Conservez à l\'avance le fichier d\'installation (APK) du même réseau : sans lui, la récupération sur un nouvel appareil peut être impossible.';

  @override
  String get recoveryShareBtn => 'Partager la carte (image)';

  @override
  String get guardiansTitle => 'Gardiens';

  @override
  String get guardiansIntro =>
      'Les gardiens personnels confirment la migration (2 sur 3). Ils peuvent être réélus : le changement prend effet dans 48 h ; jusque-là, l\'ancienne liste s\'applique.';

  @override
  String get guardiansEntrySub =>
      'Gardiens personnels : réélection, fenêtre de 48 h';

  @override
  String get guardiansCurrent => 'Gardiens actuels';

  @override
  String get guardiansNone => 'Aucun gardien désigné';

  @override
  String get guardiansReelect => 'Réélire';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return 'Changements de gardiens utilisés : $used sur $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return 'La nouvelle liste prendra effet dans $hours h';
  }

  @override
  String get guardiansCancelBtn => 'Annuler le changement';

  @override
  String get guardiansStatusCanceled => 'Annulé';

  @override
  String get guardiansStatusEffective => 'Pris effet';

  @override
  String get guardiansStatusExpired =>
      'Expiré : la limite de pupilles du gardien est dépassée';

  @override
  String get reelectTitle => 'Réélire les gardiens';

  @override
  String get reelectIntro =>
      'Saisissez les nouveaux gardiens (jusqu\'à 3, hex, un par ligne). Transmettez le hash du changement à chaque nouveau gardien : il le signe dans son application (« Consentement du gardien ») et vous renvoie le hex de la signature.';

  @override
  String get newGuardiansLabel => 'Nouveaux gardiens (hex, un par ligne)';

  @override
  String get changeHashToCopy =>
      'Hash du changement (transmettez-le aux nouveaux gardiens) :';

  @override
  String guardiansApprovals(int count, int need) {
    return 'Consentements recueillis : $count sur $need';
  }

  @override
  String get btnPublishChange => 'Publier le changement';

  @override
  String get changePublished => 'Changement publié : prend effet dans 48 h';

  @override
  String get changeConfirmTitle => 'Confirmez le changement de gardiens (PIN)';

  @override
  String get cancelChangeConfirmTitle =>
      'Confirmez l\'annulation du changement (PIN)';

  @override
  String get changeCanceled => 'Changement annulé';

  @override
  String get consentTitle => 'Consentement du gardien';

  @override
  String get consentIntro =>
      'Vous confirmez accepter d\'être le gardien du pupille. Saisissez l\'adresse du pupille et le hash du changement, puis signez le hash. Renvoyez le hex de la signature au pupille.';

  @override
  String get wardAddressLabel => 'Adresse du pupille (hex)';

  @override
  String get changeHashLabel => 'Hash du changement (hex)';

  @override
  String get consentSignTitle =>
      'Confirmez le changement de gardiens du pupille (vous êtes le gardien)';

  @override
  String get biometricTimeout =>
      'La biométrie ne répond pas (délai dépassé) — réessayez';

  @override
  String hexChunksCount(int n, int total) {
    return 'Saisi $n sur $total';
  }

  @override
  String hexChunksSigCount(int n) {
    return 'Caractères saisis : $n';
  }

  @override
  String hexLenError(int total) {
    return 'Exactement $total caractères hex attendus';
  }

  @override
  String get hexSigEvenError =>
      'La signature doit comporter un nombre pair de caractères hex';

  @override
  String get hexCharsDropped =>
      'Caractères non valides supprimés (uniquement 0-9, a-f)';

  @override
  String get aboutNetwork => 'À propos du réseau';

  @override
  String networkSlug(String slug) {
    return 'slug : $slug';
  }

  @override
  String get paramRecurring => 'Unités périodiques (recurring)';

  @override
  String get recurringOn => 'activées (extinction à la limite de la période)';

  @override
  String get recurringOff => 'désactivées';

  @override
  String get recurringNote =>
      'Chaque unité porte la marque de la période : le transfert d\'une unité de la période P est rejeté après la limite de la période P+1. Le solde n\'est pas remis à zéro — l\'historique reste public et vérifiable.';

  @override
  String get burnedUnitError =>
      'L\'unité s\'est éteinte : la limite de la période P+1 est passée. Le transfert et l\'échange sont impossibles, mais le solde n\'est pas remis à zéro — l\'historique est conservé (E_BURNED).';

  @override
  String obNetworkLine(String pair) {
    return 'Réseau : $pair';
  }
}
