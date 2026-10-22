// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => '信任网络';

  @override
  String get navWallet => '钱包';

  @override
  String get navVouching => '担保';

  @override
  String get navRegistry => '名录';

  @override
  String get navMigration => '迁移';

  @override
  String get navSettings => '设置';

  @override
  String get btnNext => '下一步';

  @override
  String get btnCancel => '取消';

  @override
  String get btnConfirm => '确认';

  @override
  String get btnRefresh => '刷新';

  @override
  String get btnSkip => '跳过';

  @override
  String get btnLogin => '解锁';

  @override
  String get btnFingerprint => '指纹';

  @override
  String get btnSend => '发送';

  @override
  String get btnSendBt => '通过蓝牙发送';

  @override
  String get btnVouch => '担保';

  @override
  String get btnMyQr => '我的二维码';

  @override
  String get btnHideQr => '隐藏我的二维码';

  @override
  String get btnBtReceive => '通过蓝牙接收';

  @override
  String get btnMint => '发行（挑战）';

  @override
  String get btnMintDo => '发行（高强度 PoW）';

  @override
  String get btnTransfer => '转账';

  @override
  String get btnPublishAwaiting => '发布（网络可用）';

  @override
  String get btnChangeThreshold => '更改阈值';

  @override
  String get btnCancelTransfer => '取消';

  @override
  String get btnRegister => '注册钱包';

  @override
  String get btnStartMigration => '启动迁移（新设备）';

  @override
  String get btnApproveMigration => '批准他人的迁移（我是守护人）';

  @override
  String get btnRestart => '重置钱包';

  @override
  String get btnUnderstandRestart => '我明白，重置';

  @override
  String get btnAddSig => '添加签名';

  @override
  String get btnPublishProposal => '发布提议';

  @override
  String get btnSignBiometric => '签名（生物识别）';

  @override
  String get enterPin => '输入 PIN';

  @override
  String get errWrongPin => 'PIN 错误';

  @override
  String get confirmOperation => '确认操作';

  @override
  String get keyUnlockTitle => '解锁密钥';

  @override
  String rejectedWithCode(String code) {
    return '已拒绝：$code';
  }

  @override
  String errorWith(String e) {
    return '错误：$e';
  }

  @override
  String get obTitle => '创建钱包';

  @override
  String get obBioRequired => '钱包需要带生物识别的硬件密钥（BIOMETRIC_STRONG）。未配置指纹无法注册。';

  @override
  String get obBioNotSetup => '指纹未设置。请打开系统设置界面并添加指纹。';

  @override
  String get obBioSetup => '设置指纹';

  @override
  String get obPinIntro =>
      'PIN 保护应用。胁迫 PIN（可选）打开诱饵模式：界面相同，但显示一个可信的小额余额；签名密钥不会被使用。';

  @override
  String get obPinNormal => 'PIN（普通）';

  @override
  String get obWantDuress => '设置胁迫 PIN';

  @override
  String get obDuressPin => '胁迫 PIN';

  @override
  String get obDecoyLegend => '诱饵模式设定（胁迫者将看到的）：';

  @override
  String get obDecoyBalance => '可信余额（不为 0）';

  @override
  String get obDecoyHistory => '历史（每行：日期 — 金额 — 收款人）';

  @override
  String get pinMinDigits => '至少 4 位数字';

  @override
  String get duressPinMinDigits => '胁迫 PIN：至少 4 位数字';

  @override
  String get pinsMatch => '两个 PIN 相同';

  @override
  String get obGuardiansIntro =>
      '指定个人守护人（2/3）。他们可以批准向新设备的迁移。添加新指纹会使密钥失效——没有守护人将失去对钱包的访问。';

  @override
  String get obGuardiansEmpty =>
      '网络中暂时没有其他成员。现在可以不指定守护人，但这一步必须完成：之后名单不可更改（仅创建时）。';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status，账龄 $age 天';
  }

  @override
  String obSelectedCount(int count) {
    return '已选：$count';
  }

  @override
  String get regStatusReady => '准备注册';

  @override
  String get regStatusPow => '认证与 PoW（可能需要一些时间）…';

  @override
  String get regConfirmTitle => '确认注册';

  @override
  String get regStatusDone => '钱包已注册';

  @override
  String pendingCount(int count) {
    return '延迟转账：$count';
  }

  @override
  String get pendingHint => '大额转账在延迟 D 后执行；在 D 内可以取消';

  @override
  String get historyTitle => '历史';

  @override
  String get historyEmpty => '暂无记录';

  @override
  String get effBalance => '有效余额';

  @override
  String get effBalanceDem => '有效余额（已计损耗费）';

  @override
  String lockedReserve(int amount) {
    return '延迟转账预留：$amount';
  }

  @override
  String get flaggedNotice => '钱包已被标记（flagged）——标记公开，不作裁决';

  @override
  String get registeredNeedK => '状态为 registered：激活需要 K 个担保（「担保」标签页）';

  @override
  String get btAwaitingTitle => '已通过蓝牙接收——等待发布';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$amount 来自 $hash';
  }

  @override
  String btStatus(String status) {
    return '状态：$status';
  }

  @override
  String get txRegister => '钱包注册';

  @override
  String get txVouch => '担保';

  @override
  String txMint(int period) {
    return '发行（周期 $period）';
  }

  @override
  String txTransfer(int amount, String hash) {
    return '转账 $amount → $hash';
  }

  @override
  String get mintTitle => '发行';

  @override
  String mintPeriod(int period) {
    return '周期：$period';
  }

  @override
  String mintWindowOpen(int hours) {
    return '挑战窗口已开启（时隙前后 ±$hours 小时）。可以发行。';
  }

  @override
  String get mintWindowOff => '挑战窗口检查已关闭（沙盒）。可以发行。';

  @override
  String mintWindowClosed(int periods) {
    return '挑战窗口已关闭。月内时隙无法提前预知；错过可在累积窗口（$periods 个周期）内补领，资金不会被锁定。';
  }

  @override
  String mintLastClaimed(int period) {
    return '最后领取的周期：$period';
  }

  @override
  String mintClaimPeriod(String period) {
    return '可领取周期：$period（已结束的周期，M4）';
  }

  @override
  String mintClaimAmount(int amount) {
    return '发行金额：$amount';
  }

  @override
  String mintIssuance(int amount) {
    return '每周期发行量：$amount';
  }

  @override
  String get mintConfirmTitle => '确认发行';

  @override
  String get mintPowStatus => '高强度 PoW（Argon2id）…';

  @override
  String get mintNoPeriods => '没有可领取的周期';

  @override
  String get mintPublished => '发行已发布';

  @override
  String get transferTitle => '转账';

  @override
  String get segNetwork => '通过网络';

  @override
  String get segBluetooth => '蓝牙';

  @override
  String get recipientLabel => '收款人地址（hex）';

  @override
  String get recipientHint => '64 个十六进制字符';

  @override
  String get amountLabel => '金额（收款人将恰好收到此数额）';

  @override
  String feeHelper(int fee) {
    return '手续费：$fee（金额之外，销毁）——由发送方支付';
  }

  @override
  String delayedWarn(int hours) {
    return '金额超过阈值 P：转账将被延迟。$hours 小时后执行；延迟期间可以取消（「延迟转账」界面）。';
  }

  @override
  String get btModeHint => '蓝牙离线：收款人将看到「已通过蓝牙接收——等待发布」。在网络确认之前这不是付款（12.4）。';

  @override
  String moratoriumNote(int days) {
    return '宽限期：注册 $days 天后可用转账。';
  }

  @override
  String get transferConfirmTitle => '确认转账';

  @override
  String get btConfirmTitle => '确认蓝牙付款';

  @override
  String get transferSent => '转账已发送';

  @override
  String transferDelayed(int hours) {
    return '转账已延迟：$hours 小时后执行，延迟期间可以取消';
  }

  @override
  String get signNotConfirmed => '签名未确认';

  @override
  String get btNoDevices => '没有已配对的设备';

  @override
  String get btSent => '已通过蓝牙发送。收款人必须等待网络确认——发布之前这不是付款';

  @override
  String btError(String error) {
    return '蓝牙：$error';
  }

  @override
  String get pendingTitle => '延迟转账';

  @override
  String thresholdP(int value) {
    return '阈值 P：$value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return '提高到 $value 将在 $time 后生效';
  }

  @override
  String get newThresholdLabel => '新阈值 P';

  @override
  String newThresholdHelper(int max) {
    return '上限 P_max = $max；提高——48 小时后生效，降低——立即生效';
  }

  @override
  String get pendingEmpty => '没有进行中的延迟转账';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return '$time 后执行（手续费 $fee）';
  }

  @override
  String get cancelConfirmTitle => '确认取消转账';

  @override
  String get canceledMsg => '已取消';

  @override
  String get invalidThreshold => '阈值无效';

  @override
  String get thresholdRaised => '提高将在 48 小时后生效（防胁迫保护）';

  @override
  String get thresholdLowered => '阈值已立即降低';

  @override
  String timeHm(int h, int m) {
    return '$h 小时 $m 分';
  }

  @override
  String get vouchingTitle => '担保';

  @override
  String get segMyStatus => '我的状态';

  @override
  String get segVouch => '担保';

  @override
  String get segMySlots => '我的名额';

  @override
  String get walletActive => '钱包已激活';

  @override
  String vouchProgress(int received, int k) {
    return '担保：$received/$k';
  }

  @override
  String get vouchFindHelpers => '当面找到 K 位担保人：在「担保」标签页向他们展示您地址的二维码（「我的二维码」按钮）。';

  @override
  String get vouchFlagged => 'flagged 标记：无人可以为您担保（被标记钱包的担保已被禁止，决定 №14）';

  @override
  String get vouchConfirmTitle => '确认担保';

  @override
  String get vouchPublished => '担保已发布';

  @override
  String get btWaiting => '正在通过蓝牙等待地址（60 秒）…';

  @override
  String get btAddrReceived => '已通过蓝牙接收地址';

  @override
  String get vouchWarning =>
      '当面担保（二维码或蓝牙）不能证明本人到场。选择由您负责：名额有限，若被担保人出现 evidence，您将失去奖励（不是资金）。';

  @override
  String get vouchOnlyActive => '只有账龄 ≥ 3 个月且完成 ≥ 2 个发行周期的 active 钱包才能担保（7.4）。';

  @override
  String get candidateLabel => '候选人地址（hex）';

  @override
  String get candidateHint => '64 个十六进制字符——来自二维码或蓝牙';

  @override
  String get myQrHint => '我的地址——展示给担保人';

  @override
  String slotsSummary(int occupied, int capacity) {
    return '名额：已占用 $occupied/$capacity';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return '空闲：$free。担保人统计：已担保 $total，激活 $active（公共信誉，7.4）';
  }

  @override
  String get slotsEmpty => '名额空闲';

  @override
  String slotActiveSub(int periods) {
    return '候选人已激活：无 evidence 完成 $periods 个发行周期后名额释放';
  }

  @override
  String slotInactiveSub(int days) {
    return '候选人未激活：$days 天后名额释放';
  }

  @override
  String get registryTitle => '网络名录';

  @override
  String registryRowSub(String status, int age) {
    return '$status • 账龄 $age 天';
  }

  @override
  String chipAge(int age) {
    return '账龄 $age 天';
  }

  @override
  String get chipSeed => '种子圈';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => '公共守护人';

  @override
  String vouchDepthLabel(int depth) {
    return '担保图深度：$depth';
  }

  @override
  String get vouchDepthUnlinked => '担保图深度：与种子圈无关联';

  @override
  String rewardLine(int periods) {
    return '担保人奖励：$periods 个已完成的发行周期';
  }

  @override
  String asVoucher(int count, int free) {
    return '作为担保人：已担保 $count（空闲名额：$free）';
  }

  @override
  String get publicHistory => '公共历史';

  @override
  String get migrationTitle => '迁移';

  @override
  String get migrationIntro =>
      '钱包恢复：守护人需提前指定（创建时）。个人守护人——2/3；含公共守护人——3/5（阈值可组合）。';

  @override
  String migrationTo(String hash) {
    return '迁移至 $hash';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return '$days 天后执行（延迟 $delay 天）；守护人签名：$sigs';
  }

  @override
  String get publicGuardiansTitle => '网络的公共守护人（guardian_offer，11）';

  @override
  String get noGuardians => '尚无人宣布为公共守护人';

  @override
  String guardianOfferRow(int age, int wards) {
    return '账龄 $age 天，被监护人：$wards';
  }

  @override
  String get restartIntro => '如果没有守护人：重置即用新密钥和新的 K 个担保进行普通注册。旧钱包在名录中保持原样且不可访问。';

  @override
  String get cancelMigrationTitle => '确认取消迁移（旧密钥）';

  @override
  String get migrationCanceled => '迁移已取消';

  @override
  String get restartDialogTitle => '资金损失';

  @override
  String get restartDialogBody =>
      '重置不会带来任何超出新钱包注册的东西：旧地址上的资金将丢失，历史不转移。有守护人的迁移会转移账龄、发行历史和担保。';

  @override
  String get restartFlowNote => '重置：删除应用数据并重新完成钱包创建（新密钥、新的 K 个担保）。';

  @override
  String get startMigrationTitle => '启动迁移';

  @override
  String get startMigrationIntro =>
      '您在新设备上。输入旧钱包地址和守护人签名（每位守护人在自己的应用中签署声明——「批准他人的迁移」标签页）。';

  @override
  String get oldAddressLabel => '旧地址（hex）';

  @override
  String get newAddressLabel => '新地址（hex）';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return '已收集签名：$count（个人阈值 $personal/$personalCount；含公共——$public/$publicCount）';
  }

  @override
  String get guardianAddrLabel => '守护人地址（hex）';

  @override
  String get guardianSigLabel => '守护人签名（hex）';

  @override
  String get migrationConfirmTitle => '确认迁移';

  @override
  String migrationStarted(int days) {
    return '迁移已启动：$days 天后执行；旧密钥可在延迟期间取消';
  }

  @override
  String get approveTitle => '迁移批准（守护人）';

  @override
  String get approveIntro => '通过您自己的渠道核实被监护人身份。签名绑定地址对与 nonce；请当面交给被监护人。';

  @override
  String get nonceLabel => 'Nonce（来自被监护人）';

  @override
  String get wardSignTitle => '确认被监护人的迁移（您是守护人）';

  @override
  String get sigToCopy => '签名（交给被监护人）：';

  @override
  String get nonceToCopy => '迁移 nonce（交给守护人）：';

  @override
  String get settingsTitle => '设置';

  @override
  String networkLine(String name) {
    return '网络：$name';
  }

  @override
  String get paramUnit => '单位';

  @override
  String get paramTransfer => '转账';

  @override
  String get paramIssuance => '发行';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '每 $months 个月 $amount';
  }

  @override
  String get paramVouchK => 'K（担保）';

  @override
  String get paramThresholdP => '阈值 P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p（P_max $max）';
  }

  @override
  String get paramDelayD => '延迟 D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours 小时';
  }

  @override
  String get paramMoratorium => '宽限期';

  @override
  String paramMoratoriumValue(String days) {
    return '$days 天';
  }

  @override
  String get paramDemurrage => '损耗费';

  @override
  String get demurrageOn => '已启用';

  @override
  String get demurrageOff => '已停用';

  @override
  String get paramAccrual => '累积窗口';

  @override
  String paramAccrualValue(String periods) {
    return '$periods 个周期';
  }

  @override
  String get paramGuardians => '守护人';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return '个人 $pt/$pc；公共 $pubt/$pubc';
  }

  @override
  String get paramMigration => '迁移';

  @override
  String paramMigrationValue(String days) {
    return '延迟 $days 天';
  }

  @override
  String get paramEpoch => '检查点周期';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds 秒';
  }

  @override
  String get sandboxNote => '沙盒（合成认证）';

  @override
  String get guardianOfferTitle => '宣布为公共守护人';

  @override
  String guardianAlready(int wards) {
    return '您已是公共守护人（被监护人：$wards）';
  }

  @override
  String guardianReq(int days) {
    return '要求：active，账龄 ≥ $days 天';
  }

  @override
  String get guardianConfirmTitle => '确认宣布为公共守护人';

  @override
  String get guardianPublished => '声明已发布';

  @override
  String get nodeRunning => '节点运行中';

  @override
  String get nodeStopped => '节点已停止';

  @override
  String get nodeLevelHint => 'full 级别——充电且连 Wi-Fi 时，否则 light（12.2）';

  @override
  String get lockApp => '锁定应用';

  @override
  String get relockNote => '已锁定。请重新启动应用。';

  @override
  String get languageTitle => '语言';

  @override
  String get languageSystem => '跟随系统';

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
  String get recoveryTitle => '恢复卡';

  @override
  String get recoveryIntro => '请在手机之外保存这些数据：丢失设备时需要它们来恢复钱包（由守护人迁移，第 11 节）。';

  @override
  String get recoveryAddress => '钱包地址';

  @override
  String get recoveryNetwork => '网络';

  @override
  String get recoveryGuardians => '您的守护人';

  @override
  String get recoveryNoGuardians => '未指定守护人：无法恢复，只能重置（普通注册并获得新的担保）。';

  @override
  String get recoveryHowToSave =>
      '保存方法：抄写在纸上或保存在手机之外的安全地点；不要存放在云端。可通过「分享」打印卡片图片。';

  @override
  String get recoveryNeverShare =>
      '切勿向任何人透露迁移数据（nonce 和守护人签名）——只能直接发给您的守护人，且仅在迁移期间使用；迁移完成后请从聊天记录中删除。只能向他人展示地址和本卡的二维码。';

  @override
  String get recoverySavedDone => '已保存 — 继续';

  @override
  String get settingsRecovery => '恢复卡';

  @override
  String get recoveryApkNote => '请提前保存同一网络的安装包（APK）：没有它，在新设备上恢复可能无法进行。';

  @override
  String get recoveryShareBtn => '分享卡片（图片）';

  @override
  String get guardiansTitle => '守护人';

  @override
  String get guardiansIntro => '个人守护人确认迁移（2/3）。可以改选：变更在 48 小时后生效，此前旧名单仍然有效。';

  @override
  String get guardiansEntrySub => '个人守护人：改选，48 小时窗口';

  @override
  String get guardiansCurrent => '当前守护人';

  @override
  String get guardiansNone => '未指定守护人';

  @override
  String get guardiansReelect => '改选';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return '已用守护人变更：$used / $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return '新名单将在 $hours 小时后生效';
  }

  @override
  String get guardiansCancelBtn => '取消变更';

  @override
  String get guardiansStatusCanceled => '已取消';

  @override
  String get guardiansStatusEffective => '已生效';

  @override
  String get guardiansStatusExpired => '已失效：守护人的被监护人名额已满';

  @override
  String get reelectTitle => '改选守护人';

  @override
  String get reelectIntro =>
      '输入新的守护人（最多 3 个，hex，每行一个）。将变更哈希转交给每位新守护人：对方在其应用中签名（“守护人同意”），并把签名的 hex 交还给您。';

  @override
  String get newGuardiansLabel => '新守护人（hex，每行一个）';

  @override
  String get changeHashToCopy => '变更哈希（转交给新守护人）：';

  @override
  String guardiansApprovals(int count, int need) {
    return '已收集同意：$count / $need';
  }

  @override
  String get btnPublishChange => '发布变更';

  @override
  String get changePublished => '变更已发布：48 小时后生效';

  @override
  String get changeConfirmTitle => '确认守护人变更（PIN）';

  @override
  String get cancelChangeConfirmTitle => '确认取消变更（PIN）';

  @override
  String get changeCanceled => '变更已取消';

  @override
  String get consentTitle => '守护人同意';

  @override
  String get consentIntro =>
      '您确认同意成为该被监护人的守护人。输入被监护人地址和变更哈希并签名。将签名的 hex 交还给被监护人。';

  @override
  String get wardAddressLabel => '被监护人地址（hex）';

  @override
  String get changeHashLabel => '变更哈希（hex）';

  @override
  String get consentSignTitle => '确认被监护人的守护人变更（您是守护人）';

  @override
  String get biometricTimeout => '生物识别无响应（超时）——请重试';

  @override
  String hexChunksCount(int n, int total) {
    return '已输入 $n / $total';
  }

  @override
  String hexChunksSigCount(int n) {
    return '已输入字符：$n';
  }

  @override
  String hexLenError(int total) {
    return '应恰好输入 $total 个十六进制字符';
  }

  @override
  String get hexSigEvenError => '签名应为偶数个十六进制字符';

  @override
  String get hexCharsDropped => '已移除无效字符（仅限 0-9、a-f）';
}
