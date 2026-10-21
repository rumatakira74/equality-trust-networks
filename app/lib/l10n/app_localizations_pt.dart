// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Portuguese (`pt`).
class AppLocalizationsPt extends AppLocalizations {
  AppLocalizationsPt([String locale = 'pt']) : super(locale);

  @override
  String get appTitle => 'Redes de confiança';

  @override
  String get navWallet => 'Carteira';

  @override
  String get navVouching => 'Avalistas';

  @override
  String get navRegistry => 'Registro';

  @override
  String get navMigration => 'Migração';

  @override
  String get navSettings => 'Ajustes';

  @override
  String get btnNext => 'Avançar';

  @override
  String get btnCancel => 'Cancelar';

  @override
  String get btnConfirm => 'Confirmar';

  @override
  String get btnRefresh => 'Atualizar';

  @override
  String get btnSkip => 'Pular';

  @override
  String get btnLogin => 'Entrar';

  @override
  String get btnFingerprint => 'Digital';

  @override
  String get btnSend => 'Enviar';

  @override
  String get btnSendBt => 'Enviar por Bluetooth';

  @override
  String get btnVouch => 'Avalizar';

  @override
  String get btnMyQr => 'Meu QR';

  @override
  String get btnHideQr => 'Ocultar meu QR';

  @override
  String get btnBtReceive => 'Receber por BT';

  @override
  String get btnMint => 'Emissão (desafio)';

  @override
  String get btnMintDo => 'Emitir (PoW reforçado)';

  @override
  String get btnTransfer => 'Transferir';

  @override
  String get btnPublishAwaiting => 'Publicar (rede disponível)';

  @override
  String get btnChangeThreshold => 'Alterar limite';

  @override
  String get btnCancelTransfer => 'Cancelar';

  @override
  String get btnRegister => 'Registrar carteira';

  @override
  String get btnStartMigration => 'Iniciar migração (novo dispositivo)';

  @override
  String get btnApproveMigration => 'Aprovar migração de outro (sou guardião)';

  @override
  String get btnRestart => 'Reiniciar carteira';

  @override
  String get btnUnderstandRestart => 'Entendo, reiniciar';

  @override
  String get btnAddSig => 'Adicionar assinatura';

  @override
  String get btnPublishProposal => 'Publicar proposta';

  @override
  String get btnSignBiometric => 'Assinar (biometria)';

  @override
  String get enterPin => 'Digite o PIN';

  @override
  String get errWrongPin => 'PIN incorreto';

  @override
  String get confirmOperation => 'Confirme a operação';

  @override
  String get keyUnlockTitle => 'Desbloquear chave';

  @override
  String rejectedWithCode(String code) {
    return 'Rejeitado: $code';
  }

  @override
  String errorWith(String e) {
    return 'Erro: $e';
  }

  @override
  String get obTitle => 'Criação da carteira';

  @override
  String get obBioRequired =>
      'A carteira exige uma chave de hardware com biometria (BIOMETRIC_STRONG). Sem uma digital configurada o registro é impossível.';

  @override
  String get obBioNotSetup =>
      'Digital não configurada. Abra a tela de ajustes do sistema e adicione uma digital.';

  @override
  String get obBioSetup => 'Configurar digital';

  @override
  String get obPinIntro =>
      'O PIN protege o app. Um PIN de coação (opcional) abre o modo isca: a mesma interface, mas um saldo pequeno verossímil; a chave de assinatura não é usada.';

  @override
  String get obPinNormal => 'PIN (normal)';

  @override
  String get obWantDuress => 'Definir PIN de coação';

  @override
  String get obDuressPin => 'PIN de coação';

  @override
  String get obDecoyLegend => 'Lenda do modo isca (o que quem coage verá):';

  @override
  String get obDecoyBalance => 'Saldo verossímil (não 0)';

  @override
  String get obDecoyHistory =>
      'Histórico (linhas: data — valor — destinatário)';

  @override
  String get pinMinDigits => 'mínimo 4 dígitos';

  @override
  String get duressPinMinDigits => 'PIN de coação: mínimo 4 dígitos';

  @override
  String get pinsMatch => 'Os PINs coincidem';

  @override
  String get obGuardiansIntro =>
      'Designe guardiões pessoais (2 de 3). Eles poderão aprovar a migração para um dispositivo novo. Adicionar uma digital nova invalida a chave — sem guardiões o acesso à carteira é perdido.';

  @override
  String get obGuardiansEmpty =>
      'Ainda não há outros membros na rede. Pode não designar guardiões agora, mas a oferta é obrigatória: a lista não muda depois (somente na criação).';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status, idade $age dias';
  }

  @override
  String obSelectedCount(int count) {
    return 'Selecionados: $count';
  }

  @override
  String get regStatusReady => 'Pronto para registrar';

  @override
  String get regStatusPow => 'Atestação e PoW (pode demorar)…';

  @override
  String get regConfirmTitle => 'Confirme o registro';

  @override
  String get regStatusDone => 'Carteira registrada';

  @override
  String pendingCount(int count) {
    return 'Transferências diferidas: $count';
  }

  @override
  String get pendingHint =>
      'Transferências grandes executam após o atraso D; podem ser canceladas dentro de D';

  @override
  String get historyTitle => 'Histórico';

  @override
  String get historyEmpty => 'Nada ainda';

  @override
  String get effBalance => 'saldo efetivo';

  @override
  String get effBalanceDem => 'saldo efetivo (demurrage aplicado)';

  @override
  String lockedReserve(int amount) {
    return 'reservado para diferidas: $amount';
  }

  @override
  String get flaggedNotice =>
      'A carteira está marcada (flagged) — a marca é pública, sem veredictos';

  @override
  String get registeredNeedK =>
      'Estado registered: a ativação exige K avalistas (aba «Avalistas»)';

  @override
  String get btAwaitingTitle =>
      'Recebido por Bluetooth — aguardando publicação';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount de $hash';
  }

  @override
  String btStatus(String status) {
    return 'estado: $status';
  }

  @override
  String get txRegister => 'Registro da carteira';

  @override
  String get txVouch => 'Aval';

  @override
  String txMint(int period) {
    return 'Emissão (período $period)';
  }

  @override
  String txTransfer(int amount, String hash) {
    return 'Transferência $amount → $hash';
  }

  @override
  String get mintTitle => 'Emissão';

  @override
  String mintPeriod(int period) {
    return 'Período: $period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'A janela de desafio está aberta (±$hours h em torno da ranhura). Pode emitir.';
  }

  @override
  String get mintWindowOff =>
      'A verificação da janela de desafio está desativada (sandbox). Pode emitir.';

  @override
  String mintWindowClosed(int periods) {
    return 'A janela de desafio está fechada. A ranhura dentro do mês é imprevisível de antemão; uma perda é transferida dentro da janela de acumulação ($periods períodos), os fundos não são bloqueados.';
  }

  @override
  String mintLastClaimed(int period) {
    return 'Último período reclamado: $period';
  }

  @override
  String mintClaimPeriod(String period) {
    return 'Período a reclamar: $period (períodos concluídos, M4)';
  }

  @override
  String mintClaimAmount(int amount) {
    return 'Valor a emitir: $amount';
  }

  @override
  String mintIssuance(int amount) {
    return 'Emissão por período: $amount';
  }

  @override
  String get mintConfirmTitle => 'Confirme a emissão';

  @override
  String get mintPowStatus => 'PoW reforçado (Argon2id)…';

  @override
  String get mintNoPeriods => 'Não há períodos a reclamar';

  @override
  String get mintPublished => 'Emissão publicada';

  @override
  String get transferTitle => 'Transferência';

  @override
  String get segNetwork => 'Pela rede';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => 'Endereço do destinatário (hex)';

  @override
  String get recipientHint => '64 caracteres hex';

  @override
  String get amountLabel => 'Valor (o destinatário recebe exatamente isto)';

  @override
  String feeHelper(int fee) {
    return 'Taxa: $fee (além do valor, é queimada) — paga o remetente';
  }

  @override
  String delayedWarn(int hours) {
    return 'O valor ultrapassa o limite P: a transferência será diferida. Execução em $hours h; pode ser cancelada durante o atraso (tela «Diferidas»).';
  }

  @override
  String get btModeHint =>
      'Bluetooth offline: o destinatário verá «recebido por Bluetooth — aguardando publicação». Até a confirmação da rede não é um pagamento (12.4).';

  @override
  String moratoriumNote(int days) {
    return 'Moratória: transferências disponíveis $days dias após o registro.';
  }

  @override
  String get transferConfirmTitle => 'Confirme a transferência';

  @override
  String get btConfirmTitle => 'Confirme o pagamento por Bluetooth';

  @override
  String get transferSent => 'Transferência enviada';

  @override
  String transferDelayed(int hours) {
    return 'Transferência diferida: execução em $hours h, pode ser cancelada durante o atraso';
  }

  @override
  String get signNotConfirmed => 'Assinatura não confirmada';

  @override
  String get btNoDevices => 'Não há dispositivos pareados';

  @override
  String get btSent =>
      'Enviado por Bluetooth. O destinatário deve aguardar a confirmação da rede — até a publicação não é um pagamento';

  @override
  String btError(String error) {
    return 'Bluetooth: $error';
  }

  @override
  String get pendingTitle => 'Transferências diferidas';

  @override
  String thresholdP(int value) {
    return 'Limite P: $value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return 'O aumento para $value entra em vigor após $time';
  }

  @override
  String get newThresholdLabel => 'Novo limite P';

  @override
  String newThresholdHelper(int max) {
    return 'limite P_max = $max; aumento — em 48 h, redução — imediata';
  }

  @override
  String get pendingEmpty => 'Não há transferências diferidas ativas';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return 'execução em $time (taxa $fee)';
  }

  @override
  String get cancelConfirmTitle => 'Confirme o cancelamento da transferência';

  @override
  String get canceledMsg => 'Cancelada';

  @override
  String get invalidThreshold => 'Limite inválido';

  @override
  String get thresholdRaised =>
      'O aumento entra em vigor em 48 h (proteção contra coação)';

  @override
  String get thresholdLowered => 'Limite reduzido imediatamente';

  @override
  String timeHm(int h, int m) {
    return '$h h $m min';
  }

  @override
  String get vouchingTitle => 'Avalistas';

  @override
  String get segMyStatus => 'Meu estado';

  @override
  String get segVouch => 'Avalizar';

  @override
  String get segMySlots => 'Minhas ranhuras';

  @override
  String get walletActive => 'A carteira está ativa';

  @override
  String vouchProgress(int received, int k) {
    return 'Avais: $received de $k';
  }

  @override
  String get vouchFindHelpers =>
      'Encontre K avalistas pessoalmente: mostre-lhes o QR do seu endereço na aba «Avalizar» (botão «Meu QR»).';

  @override
  String get vouchFlagged =>
      'Marca flagged: ninguém pode avalizar você (aval de carteira marcada é proibido, decisão n.º 14)';

  @override
  String get vouchConfirmTitle => 'Confirme o aval';

  @override
  String get vouchPublished => 'Aval publicado';

  @override
  String get btWaiting => 'Aguardando endereço por Bluetooth (60 s)…';

  @override
  String get btAddrReceived => 'Endereço recebido por Bluetooth';

  @override
  String get vouchWarning =>
      'O aval presencial (QR ou Bluetooth) NÃO prova a presença da pessoa. Você responde pela escolha: a ranhura é limitada e, se houver evidence contra o avalizado, você perde a recompensa (não dinheiro).';

  @override
  String get vouchOnlyActive =>
      'Somente uma carteira active com idade ≥ 3 meses e ≥ 2 períodos de emissão concluídos pode avalizar (7.4).';

  @override
  String get candidateLabel => 'Endereço do candidato (hex)';

  @override
  String get candidateHint => '64 caracteres hex — do QR ou Bluetooth';

  @override
  String get myQrHint => 'Meu endereço — mostre ao avalista';

  @override
  String slotsSummary(int occupied, int capacity) {
    return 'Ranhuras: $occupied ocupadas de $capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return 'Livres: $free. Estatísticas do avalista: avalizou $total, ativos $active (reputação pública, 7.4)';
  }

  @override
  String get slotsEmpty => 'As ranhuras estão livres';

  @override
  String slotActiveSub(int periods) {
    return 'o candidato está ativo: a ranhura libera após $periods períodos de emissão sem evidence';
  }

  @override
  String slotInactiveSub(int days) {
    return 'o candidato não está ativo: a ranhura libera em $days dias';
  }

  @override
  String get registryTitle => 'Registro da rede';

  @override
  String registryRowSub(String status, int age) {
    return '$status • idade $age dias';
  }

  @override
  String chipAge(int age) {
    return 'idade $age dias';
  }

  @override
  String get chipSeed => 'círculo semente';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => 'guardião público';

  @override
  String vouchDepthLabel(int depth) {
    return 'Profundidade do grafo de avais: $depth';
  }

  @override
  String get vouchDepthUnlinked =>
      'Profundidade do grafo de avais: sem ligação com o círculo semente';

  @override
  String rewardLine(int periods) {
    return 'Recompensa aos avalistas: $periods períodos de emissão concluídos';
  }

  @override
  String asVoucher(int count, int free) {
    return 'Como avalista: por $count (ranhuras livres: $free)';
  }

  @override
  String get publicHistory => 'Histórico público';

  @override
  String get migrationTitle => 'Migração';

  @override
  String get migrationIntro =>
      'Recuperação da carteira: os guardiões são designados antecipadamente (na criação). Pessoais — 2 de 3; com públicos — 3 de 5 (os limites se combinam).';

  @override
  String migrationTo(String hash) {
    return 'Migração para $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return 'execução em $days dias (atraso $delay dias); assinaturas de guardiões: $sigs';
  }

  @override
  String get publicGuardiansTitle =>
      'Guardiões públicos da rede (guardian_offer, 11)';

  @override
  String get noGuardians => 'Ninguém se declarou guardião público ainda';

  @override
  String guardianOfferRow(int age, int wards) {
    return 'idade $age dias, tutelados: $wards';
  }

  @override
  String get restartIntro =>
      'Se não há guardiões: o reinício é um registro comum com chave nova e K avais novos. A carteira antiga permanece no registro intacta e inacessível.';

  @override
  String get cancelMigrationTitle =>
      'Confirme o cancelamento da migração (chave antiga)';

  @override
  String get migrationCanceled => 'Migração cancelada';

  @override
  String get restartDialogTitle => 'Perda de fundos';

  @override
  String get restartDialogBody =>
      'O reinício não dá nada além do registro de uma carteira nova: os fundos do endereço antigo são perdidos, o histórico não é transferido. A migração com guardiões transfere idade, histórico de emissões e avais.';

  @override
  String get restartFlowNote =>
      'Reinício: apague os dados do aplicativo e refaça a criação da carteira (chave nova, K avais novos).';

  @override
  String get startMigrationTitle => 'Início da migração';

  @override
  String get startMigrationIntro =>
      'Você está no dispositivo NOVO. Digite o endereço da carteira antiga e as assinaturas dos guardiões (cada guardião assina a declaração no próprio app — aba «Aprovar migração de outro»).';

  @override
  String get oldAddressLabel => 'Endereço antigo (hex)';

  @override
  String get newAddressLabel => 'Endereço novo (hex)';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return 'Assinaturas reunidas: $count (limite pessoal $personal de $personalCount; com públicos — $public de $publicCount)';
  }

  @override
  String get guardianAddrLabel => 'Endereço do guardião (hex)';

  @override
  String get guardianSigLabel => 'Assinatura do guardião (hex)';

  @override
  String get migrationConfirmTitle => 'Confirme a migração';

  @override
  String migrationStarted(int days) {
    return 'Migração iniciada: execução em $days dias; a chave antiga pode cancelar durante o atraso';
  }

  @override
  String get approveTitle => 'Aprovação da migração (guardião)';

  @override
  String get approveIntro =>
      'Verifique a identidade do tutelado pelos seus próprios canais. A assinatura vincula o par de endereços e o nonce; entregue-a ao tutelado pessoalmente.';

  @override
  String get nonceLabel => 'Nonce (do tutelado)';

  @override
  String get wardSignTitle =>
      'Confirme a migração do tutelado (você é o guardião)';

  @override
  String get sigToCopy => 'Assinatura (entregue ao tutelado):';

  @override
  String get nonceToCopy => 'Nonce de migração (entregue aos guardiões):';

  @override
  String get settingsTitle => 'Ajustes';

  @override
  String networkLine(String name) {
    return 'Rede: $name';
  }

  @override
  String get paramUnit => 'Unidade';

  @override
  String get paramTransfer => 'Transferência';

  @override
  String get paramIssuance => 'Emissão';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$amount a cada $months meses';
  }

  @override
  String get paramVouchK => 'K (avais)';

  @override
  String get paramThresholdP => 'Limite P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p (P_max $max)';
  }

  @override
  String get paramDelayD => 'Atraso D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours h';
  }

  @override
  String get paramMoratorium => 'Moratória';

  @override
  String paramMoratoriumValue(String days) {
    return '$days dias';
  }

  @override
  String get paramDemurrage => 'Demurrage';

  @override
  String get demurrageOn => 'ativado';

  @override
  String get demurrageOff => 'desativado';

  @override
  String get paramAccrual => 'Janela de acumulação';

  @override
  String paramAccrualValue(String periods) {
    return '$periods períodos';
  }

  @override
  String get paramGuardians => 'Guardiões';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return 'pessoais $pt de $pc; públicos $pubt de $pubc';
  }

  @override
  String get paramMigration => 'Migração';

  @override
  String paramMigrationValue(String days) {
    return 'atraso $days dias';
  }

  @override
  String get paramEpoch => 'Época de checkpoint';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds s';
  }

  @override
  String get sandboxNote => 'Sandbox (atestação sintética)';

  @override
  String get guardianOfferTitle => 'Declarar-se guardião público';

  @override
  String guardianAlready(int wards) {
    return 'Você já é guardião público (tutelados: $wards)';
  }

  @override
  String guardianReq(int days) {
    return 'Requisitos: active, idade ≥ $days dias';
  }

  @override
  String get guardianConfirmTitle => 'Confirme declarar-se guardião público';

  @override
  String get guardianPublished => 'Declaração publicada';

  @override
  String get nodeRunning => 'O nó está funcionando';

  @override
  String get nodeStopped => 'O nó está parado';

  @override
  String get nodeLevelHint =>
      'Nível full — carregando e com Wi-Fi, senão light (12.2)';

  @override
  String get lockApp => 'Bloquear o aplicativo';

  @override
  String get relockNote => 'Bloqueado. Reinicie o aplicativo.';

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

  @override
  String get recoveryTitle => 'Cartão de recuperação';

  @override
  String get recoveryIntro =>
      'Guarde estes dados fora do telefone: eles são necessários para recuperar a carteira se o dispositivo for perdido (migração com guardiões, seção 11).';

  @override
  String get recoveryAddress => 'Endereço da carteira';

  @override
  String get recoveryNetwork => 'Rede';

  @override
  String get recoveryGuardians => 'Seus guardiões';

  @override
  String get recoveryNoGuardians =>
      'Nenhum guardião designado: a recuperação é impossível, resta apenas o reinício (registro comum com avais novos).';

  @override
  String get recoveryHowToSave =>
      'Como guardar: copie no papel ou guarde num lugar seguro fora do telefone; não guarde na nuvem. A imagem do cartão pode ser impressa com «Compartilhar».';

  @override
  String get recoveryNeverShare =>
      'Nunca compartilhe os dados de migração (o nonce e as assinaturas dos guardiões) — apenas diretamente ao seu guardião e apenas durante a migração; apague-os das mensagens ao terminar. A terceiros mostre apenas o endereço e o QR deste cartão.';

  @override
  String get recoverySavedDone => 'Salvo — continuar';

  @override
  String get settingsRecovery => 'Cartão de recuperação';

  @override
  String get recoveryApkNote =>
      'Guarde com antecedência o arquivo de instalação (APK) da mesma rede: sem ele, a recuperação num dispositivo novo pode ser impossível.';

  @override
  String get recoveryShareBtn => 'Compartilhar cartão (imagem)';

  @override
  String get guardiansTitle => 'Guardiões';

  @override
  String get guardiansIntro =>
      'Os guardiões pessoais confirmam a migração (2 de 3). Podem ser reeleitos: a mudança entra em vigor em 48 h; até lá vale a lista anterior.';

  @override
  String get guardiansEntrySub =>
      'Guardiões pessoais: reeleição, janela de 48 h';

  @override
  String get guardiansCurrent => 'Guardiões atuais';

  @override
  String get guardiansNone => 'Nenhum guardião designado';

  @override
  String get guardiansReelect => 'Reeleger';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return 'Mudanças de guardiões usadas: $used de $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return 'A nova lista entrará em vigor em $hours h';
  }

  @override
  String get guardiansCancelBtn => 'Cancelar a mudança';

  @override
  String get guardiansStatusCanceled => 'Cancelada';

  @override
  String get guardiansStatusEffective => 'Em vigor';

  @override
  String get guardiansStatusExpired =>
      'Expirada: limite de tutelados do guardião excedido';

  @override
  String get reelectTitle => 'Reeleger guardiões';

  @override
  String get reelectIntro =>
      'Introduza os novos guardiões (até 3, hex, um por linha). Passe o hash da mudança a cada novo guardião: ele o assina no seu aplicativo («Consentimento do guardião») e devolve o hex da assinatura.';

  @override
  String get newGuardiansLabel => 'Novos guardiões (hex, um por linha)';

  @override
  String get changeHashToCopy => 'Hash da mudança (passe aos novos guardiões):';

  @override
  String guardiansApprovals(int count, int need) {
    return 'Consentimentos recolhidos: $count de $need';
  }

  @override
  String get btnPublishChange => 'Publicar a mudança';

  @override
  String get changePublished => 'Mudança publicada: entra em vigor em 48 h';

  @override
  String get changeConfirmTitle => 'Confirme a mudança de guardiões (PIN)';

  @override
  String get cancelChangeConfirmTitle =>
      'Confirme o cancelamento da mudança (PIN)';

  @override
  String get changeCanceled => 'Mudança cancelada';

  @override
  String get consentTitle => 'Consentimento do guardião';

  @override
  String get consentIntro =>
      'Você confirma aceitar ser guardião do tutelado. Introduza o endereço do tutelado e o hash da mudança e assine o hash. Devolva o hex da assinatura ao tutelado.';

  @override
  String get wardAddressLabel => 'Endereço do tutelado (hex)';

  @override
  String get changeHashLabel => 'Hash da mudança (hex)';

  @override
  String get consentSignTitle =>
      'Confirme a mudança de guardiões do tutelado (você é o guardião)';
}
