// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Japanese (`ja`).
class AppLocalizationsJa extends AppLocalizations {
  AppLocalizationsJa([String locale = 'ja']) : super(locale);

  @override
  String get appTitle => 'トラストネットワーク';

  @override
  String get navWallet => 'ウォレット';

  @override
  String get navVouching => '保証';

  @override
  String get navRegistry => '台帳';

  @override
  String get navMigration => 'マイグレーション';

  @override
  String get navSettings => '設定';

  @override
  String get btnNext => '次へ';

  @override
  String get btnCancel => 'キャンセル';

  @override
  String get btnConfirm => '確認';

  @override
  String get btnRefresh => '更新';

  @override
  String get btnSkip => 'スキップ';

  @override
  String get btnLogin => 'ロック解除';

  @override
  String get btnFingerprint => '指紋';

  @override
  String get btnSend => '送信';

  @override
  String get btnSendBt => 'Bluetooth で送信';

  @override
  String get btnVouch => '保証する';

  @override
  String get btnMyQr => '自分の QR';

  @override
  String get btnHideQr => '自分の QR を隠す';

  @override
  String get btnBtReceive => 'BT で受信';

  @override
  String get btnMint => '発行（チャレンジ）';

  @override
  String get btnMintDo => '発行（高強度 PoW）';

  @override
  String get btnTransfer => '送金';

  @override
  String get btnPublishAwaiting => '公開（ネットワーク利用可能）';

  @override
  String get btnChangeThreshold => 'しきい値を変更';

  @override
  String get btnCancelTransfer => 'キャンセル';

  @override
  String get btnRegister => 'ウォレットを登録';

  @override
  String get btnStartMigration => 'マイグレーションを開始（新しいデバイス）';

  @override
  String get btnApproveMigration => '他人のマイグレーションを承認（私はガーディアン）';

  @override
  String get btnRestart => 'ウォレットをリセット';

  @override
  String get btnUnderstandRestart => '理解しました、リセットする';

  @override
  String get btnAddSig => '署名を追加';

  @override
  String get btnPublishProposal => '提案を公開';

  @override
  String get btnSignBiometric => '署名（生体認証）';

  @override
  String get enterPin => 'PIN を入力';

  @override
  String get errWrongPin => 'PIN が違います';

  @override
  String get errNoNode => 'ノードが起動していません';

  @override
  String lockTooManyAttempts(int seconds) {
    return '失敗が多すぎます。$seconds 秒後に再試行してください';
  }

  @override
  String get confirmOperation => '操作を確認';

  @override
  String get keyUnlockTitle => '鍵をロック解除';

  @override
  String rejectedWithCode(String code) {
    return '拒否されました：$code';
  }

  @override
  String errorWith(String e) {
    return 'エラー：$e';
  }

  @override
  String get obTitle => 'ウォレットの作成';

  @override
  String get obBioRequired =>
      'ウォレットには生体認証付きのハードウェア鍵（BIOMETRIC_STRONG）が必要です。指紋が設定されていない場合、登録はできません。';

  @override
  String get obBioNotSetup => '指紋が未設定です。システム設定画面を開き、指紋を追加してください。';

  @override
  String get obBioSetup => '指紋を設定';

  @override
  String get obPinIntro =>
      'PIN はアプリを守ります。脅迫 PIN（任意）を設定するとおとりモードが開きます：同じ画面ですが、もっともらしい少額の残高を表示し、署名鍵は使われません。';

  @override
  String get obPinNormal => 'PIN（通常）';

  @override
  String get obWantDuress => '脅迫 PIN を設定';

  @override
  String get obDuressPin => '脅迫 PIN';

  @override
  String get obDecoyLegend => 'おとりモードの設定（脅迫者が見るもの）：';

  @override
  String get obDecoyBalance => 'もっともらしい残高（0 ではなく）';

  @override
  String get obDecoyHistory => '履歴（行：日付 — 金額 — 受取人）';

  @override
  String get pinMinDigits => '最低 4 桁';

  @override
  String get duressPinMinDigits => '脅迫 PIN：最低 4 桁';

  @override
  String get pinsMatch => '2 つの PIN が一致';

  @override
  String get obGuardiansIntro =>
      '個人ガーディアン（2/3）を指定してください。新しいデバイスへのマイグレーションを承認できるようになります。新しい指紋を追加すると鍵は無効になります——ガーディアンがいないとウォレットへのアクセスを失います。';

  @override
  String get obGuardiansEmpty =>
      'ネットワークにはまだ他のメンバーがいません。今はガーディアンを指定しなくても構いませんが、この提案は必須です：リストは後から変更できません（作成時のみ）。';

  @override
  String guardianRowStatus(String status, int age) {
    return '$status、経過 $age 日';
  }

  @override
  String obSelectedCount(int count) {
    return '選択済み：$count';
  }

  @override
  String get regStatusReady => '登録準備完了';

  @override
  String get regStatusPow => '構成証明と PoW（時間がかかることがあります）…';

  @override
  String get regConfirmTitle => '登録を確認';

  @override
  String get regStatusDone => 'ウォレットを登録しました';

  @override
  String pendingCount(int count) {
    return '遅延送金：$count';
  }

  @override
  String get pendingHint => '大きな送金は遅延 D の後に実行されます。D 内ならキャンセルできます';

  @override
  String get historyTitle => '履歴';

  @override
  String get historyEmpty => 'まだありません';

  @override
  String get effBalance => '有効残高';

  @override
  String get effBalanceDem => '有効残高（デマレージ適用済み）';

  @override
  String lockedReserve(int amount) {
    return '遅延送金のため確保：$amount';
  }

  @override
  String get flaggedNotice => 'ウォレットは flagged マーク付きです——マークは公開、裁定はありません';

  @override
  String get registeredNeedK => '状態 registered：アクティブ化には K 件の保証が必要です（「保証」タブ）';

  @override
  String get btAwaitingTitle => 'Bluetooth で受信——公開待ち';

  @override
  String btAwaitingRow(int amount, String hash) {
    return '$hash から $amount';
  }

  @override
  String btStatus(String status) {
    return '状態：$status';
  }

  @override
  String get txRegister => 'ウォレット登録';

  @override
  String get txVouch => '保証';

  @override
  String txMint(int period) {
    return '発行（周期 $period）';
  }

  @override
  String txTransfer(int amount, String hash) {
    return '送金 $amount → $hash';
  }

  @override
  String get mintTitle => '発行';

  @override
  String mintPeriod(int period) {
    return '周期：$period';
  }

  @override
  String mintWindowOpen(int hours) {
    return 'チャレンジウィンドウが開いています（スロット前後 ±$hours 時間）。発行できます。';
  }

  @override
  String get mintWindowOff => 'チャレンジウィンドウの確認は無効です（サンドボックス）。発行できます。';

  @override
  String mintWindowClosed(int periods) {
    return 'チャレンジウィンドウは閉じています。月内のスロットは事前に予測できません；見送った分は累積ウィンドウ（$periods 周期）内で繰り越され、資金はロックされません。';
  }

  @override
  String mintWindowJitter(int minutes) {
    return 'ウィンドウは開いています；クライアントは $minutes 分後に送信を予定しています（ジッターが負荷のピークを平滑化します）。';
  }

  @override
  String mintLastClaimed(int period) {
    return '最後に請求した周期：$period';
  }

  @override
  String mintClaimPeriod(String period) {
    return '請求可能な周期：$period（完了済みの周期、M4）';
  }

  @override
  String mintClaimAmount(int amount) {
    return '発行額：$amount';
  }

  @override
  String mintIssuance(int amount) {
    return '周期あたりの発行量：$amount';
  }

  @override
  String get mintConfirmTitle => '発行を確認';

  @override
  String get mintPowStatus => '高強度 PoW（Argon2id）…';

  @override
  String get mintNoPeriods => '請求できる周期はありません';

  @override
  String get mintPublished => '発行を公開しました';

  @override
  String get transferTitle => '送金';

  @override
  String get segNetwork => 'ネットワーク経由';

  @override
  String get segBluetooth => 'Bluetooth';

  @override
  String get recipientLabel => '受取人のアドレス（hex）';

  @override
  String get recipientHint => '64 文字の十六進（hex）';

  @override
  String get amountLabel => '金額（受取人は正確にこの額を受け取ります）';

  @override
  String feeHelper(int fee) {
    return '手数料：$fee（金額に加算、バーン）——送信者が支払います';
  }

  @override
  String delayedWarn(int hours) {
    return '金額がしきい値 P を超えています：送金は遅延されます。$hours 時間後に実行；遅延中はキャンセルできます（「遅延送金」画面）。';
  }

  @override
  String get btModeHint =>
      'Bluetooth オフライン：受取人には「Bluetooth で受信——公開待ち」と表示されます。ネットワークが確認するまで支払いではありません（12.4）。';

  @override
  String moratoriumNote(int days) {
    return 'モラトリアム：送金は登録から $days 日後に可能になります。';
  }

  @override
  String get transferConfirmTitle => '送金を確認';

  @override
  String get btConfirmTitle => 'Bluetooth 支払いを確認';

  @override
  String get transferSent => '送金を送信しました';

  @override
  String get transferAwaiting => '送金を送信しました——確認待ち';

  @override
  String transferDelayed(int hours) {
    return '送金は遅延：$hours 時間後に実行、遅延中はキャンセルできます';
  }

  @override
  String get signNotConfirmed => '署名が確認されていません';

  @override
  String get btNoDevices => 'ペアリング済みのデバイスがありません';

  @override
  String get btSent =>
      'Bluetooth で送信しました。受取人はネットワークの確認を待つ必要があります——公開まで支払いではありません';

  @override
  String btError(String error) {
    return 'Bluetooth：$error';
  }

  @override
  String get pendingTitle => '遅延送金';

  @override
  String thresholdP(int value) {
    return 'しきい値 P：$value';
  }

  @override
  String thresholdPendingRaise(int value, String time) {
    return '$value への引き上げは $time 後に有効になります';
  }

  @override
  String get newThresholdLabel => '新しいしきい値 P';

  @override
  String newThresholdHelper(int max) {
    return '上限 P_max = $max；引き上げ——48 時間後、引き下げ——即時';
  }

  @override
  String get pendingEmpty => '進行中の遅延送金はありません';

  @override
  String pendingRow(int amount, String hash) {
    return '$amount → $hash';
  }

  @override
  String pendingRowSub(String time, int fee) {
    return '$time 後に実行（手数料 $fee）';
  }

  @override
  String get cancelConfirmTitle => '送金のキャンセルを確認';

  @override
  String get canceledMsg => 'キャンセルしました';

  @override
  String get invalidThreshold => 'しきい値が無効です';

  @override
  String get thresholdRaised => '引き上げは 48 時間後に有効になります（脅迫対策）';

  @override
  String get thresholdLowered => 'しきい値は即時に引き下げられました';

  @override
  String timeHm(int h, int m) {
    return '$h 時間 $m 分';
  }

  @override
  String get vouchingTitle => '保証';

  @override
  String get segMyStatus => '自分の状態';

  @override
  String get segVouch => '保証する';

  @override
  String get segMySlots => '自分の枠';

  @override
  String get walletActive => 'ウォレットは有効です';

  @override
  String vouchProgress(int received, int k) {
    return '保証：$k 中 $received';
  }

  @override
  String get vouchFindHelpers =>
      '保証人を K 名、対面で見つけてください：「保証する」タブで自分のアドレスの QR コードを見せてください（「自分の QR」ボタン）。';

  @override
  String get vouchFlagged =>
      'flagged マーク：誰もあなたを保証できません（flagged ウォレットからの保証は禁止、決定 №14）';

  @override
  String get vouchConfirmTitle => '保証を確認';

  @override
  String get vouchPublished => '保証を公開しました';

  @override
  String get btWaiting => 'Bluetooth でアドレスを待機中（60 秒）…';

  @override
  String get btAddrReceived => 'Bluetooth でアドレスを受信';

  @override
  String get vouchWarning =>
      '対面の保証（QR または Bluetooth）は本人の在場を証明しません。選択はあなたの責任です：枠は限られており、被保証人について証拠が出された場合は報酬を失います（資金ではありません）。';

  @override
  String get vouchOnlyActive =>
      '保証できるのは、使用期間が 3 か月以上で完了した発行周期が 2 以上の active ウォレットだけです（7.4）。';

  @override
  String get candidateLabel => '候補者のアドレス（hex）';

  @override
  String get candidateHint => '64 文字の十六進（hex）——QR または Bluetooth から';

  @override
  String get myQrHint => '自分のアドレス——保証人に見せてください';

  @override
  String slotsSummary(int occupied, int capacity) {
    return '枠：$capacity 中 $occupied 使用中';
  }

  @override
  String slotsFreeStats(int free, int total, int active) {
    return '空き：$free。保証人の統計：保証 $total、有効 $active（公開の評判、7.4）';
  }

  @override
  String get slotsEmpty => '枠は空いています';

  @override
  String slotActiveSub(int periods) {
    return '候補者は有効：証拠のない発行周期 $periods 回の後に枠が解放されます';
  }

  @override
  String slotInactiveSub(int days) {
    return '候補者は非有効：$days 日後に枠が解放されます';
  }

  @override
  String get registryTitle => 'ネットワーク台帳';

  @override
  String registryRowSub(String status, int age) {
    return '$status • 経過 $age 日';
  }

  @override
  String chipAge(int age) {
    return '経過 $age 日';
  }

  @override
  String get chipSeed => 'シードサークル';

  @override
  String get chipFlagged => 'flagged';

  @override
  String get chipPublicGuardian => '公共ガーディアン';

  @override
  String vouchDepthLabel(int depth) {
    return '保証グラフの深さ：$depth';
  }

  @override
  String get vouchDepthUnlinked => '保証グラフの深さ：シードサークルと無関係';

  @override
  String rewardLine(int periods) {
    return '保証人への報酬：完了した発行周期 $periods 回';
  }

  @override
  String asVoucher(int count, int free) {
    return '保証人として：$count 件（空き枠：$free）';
  }

  @override
  String get publicHistory => '公開履歴';

  @override
  String get migrationTitle => 'マイグレーション';

  @override
  String get migrationIntro =>
      'ウォレットの復旧：ガーディアンは事前（作成時）に指定します。個人——2/3；公共を含む——3/5（しきい値は組み合わせ可）。';

  @override
  String migrationTo(String hash) {
    return '$hash へのマイグレーション';
  }

  @override
  String migrationRowSub(int days, int delay, int sigs) {
    return '$days 日後に実行（遅延 $delay 日）；ガーディアン署名：$sigs';
  }

  @override
  String get publicGuardiansTitle => 'ネットワークの公共ガーディアン（guardian_offer、11）';

  @override
  String get noGuardians => 'まだ公共ガーディアンを名乗った人はいません';

  @override
  String guardianOfferRow(int age, int wards) {
    return '使用日数 $age 日、被後見人：$wards';
  }

  @override
  String get restartIntro =>
      'ガーディアンがいない場合：リセットとは、新しい鍵と新しい K 件の保証による通常の登録です。旧ウォレットは台帳にそのまま残り、アクセスできません。';

  @override
  String get cancelMigrationTitle => 'マイグレーションの取消を確認（旧鍵）';

  @override
  String get migrationCanceled => 'マイグレーションを取り消しました';

  @override
  String get restartDialogTitle => '資金の喪失';

  @override
  String get restartDialogBody =>
      'リセットは新しいウォレットの登録以上のものをもたらしません：旧アドレスの資金は失われ、履歴は移転しません。ガーディアンによるマイグレーションは使用期間、発行履歴、保証を移転します。';

  @override
  String get restartFlowNote =>
      'リセット：アプリのデータを削除し、ウォレット作成をやり直してください（新しい鍵、新しい K 件の保証）。';

  @override
  String get startMigrationTitle => 'マイグレーションの開始';

  @override
  String get startMigrationIntro =>
      'あなたは新しいデバイスにいます。旧ウォレットのアドレスとガーディアン署名を入力してください（各ガーディアンは自分のアプリで宣言に署名します——「他人のマイグレーションを承認」タブ）。';

  @override
  String get oldAddressLabel => '旧アドレス（hex）';

  @override
  String get newAddressLabel => '新アドレス（hex）';

  @override
  String sigsCollected(
    int count,
    int personal,
    int personalCount,
    int public,
    int publicCount,
  ) {
    return '収集した署名：$count（個人しきい値 $personal/$personalCount；公共を含む——$public/$publicCount）';
  }

  @override
  String get guardianAddrLabel => 'ガーディアンのアドレス（hex）';

  @override
  String get guardianSigLabel => 'ガーディアンの署名（hex）';

  @override
  String get migrationConfirmTitle => 'マイグレーションを確認';

  @override
  String migrationStarted(int days) {
    return 'マイグレーションを開始しました：$days 日後に実行；旧鍵は遅延中に取り消せます';
  }

  @override
  String get approveTitle => 'マイグレーションの承認（ガーディアン）';

  @override
  String get approveIntro =>
      '被後見人の本人確認は自分の経路で行ってください。署名はアドレスのペアと nonce を束縛します；被後見人に直接手渡してください。';

  @override
  String get nonceLabel => 'Nonce（被後見人から）';

  @override
  String get wardSignTitle => '被後見人のマイグレーションを確認（あなたがガーディアンです）';

  @override
  String get sigToCopy => '署名（被後見人に手渡す）：';

  @override
  String get nonceToCopy => 'マイグレーション nonce（ガーディアンに手渡す）：';

  @override
  String get settingsTitle => '設定';

  @override
  String networkLine(String name) {
    return 'ネットワーク：$name';
  }

  @override
  String get paramUnit => '単位';

  @override
  String get paramTransfer => '送金';

  @override
  String get paramIssuance => '発行';

  @override
  String paramIssuanceValue(String amount, String months) {
    return '$months か月ごとに $amount';
  }

  @override
  String get paramVouchK => 'K（保証）';

  @override
  String get paramThresholdP => 'しきい値 P';

  @override
  String paramThresholdPValue(String p, String max) {
    return '$p（P_max $max）';
  }

  @override
  String get paramDelayD => '遅延 D';

  @override
  String paramDelayDValue(String hours) {
    return '$hours 時間';
  }

  @override
  String get paramMoratorium => 'モラトリアム';

  @override
  String paramMoratoriumValue(String days) {
    return '$days 日';
  }

  @override
  String get paramDemurrage => 'デマレージ';

  @override
  String get demurrageOn => '有効';

  @override
  String get demurrageOff => '無効';

  @override
  String get paramAccrual => '累積ウィンドウ';

  @override
  String paramAccrualValue(String periods) {
    return '$periods 周期';
  }

  @override
  String get paramGuardians => 'ガーディアン';

  @override
  String paramGuardiansValue(String pt, String pc, String pubt, String pubc) {
    return '個人 $pt/$pc；公共 $pubt/$pubc';
  }

  @override
  String get paramMigration => 'マイグレーション';

  @override
  String paramMigrationValue(String days) {
    return '遅延 $days 日';
  }

  @override
  String get paramEpoch => 'チェックポイント周期';

  @override
  String paramEpochValue(String seconds) {
    return '$seconds 秒';
  }

  @override
  String get sandboxNote => 'サンドボックス（合成構成証明）';

  @override
  String get guardianOfferTitle => '公共ガーディアンへの申告';

  @override
  String guardianAlready(int wards) {
    return 'すでに公共ガーディアンです（被後見人：$wards）';
  }

  @override
  String guardianReq(int days) {
    return '要件：active、使用日数 ≥ $days 日';
  }

  @override
  String get guardianConfirmTitle => '公共ガーディアンになることを確認';

  @override
  String get guardianPublished => '申告を公開しました';

  @override
  String get nodeRunning => 'ノードは実行中';

  @override
  String get nodeStopped => 'ノードは停止中';

  @override
  String get nodeLevelHint => 'full レベル——充電中かつ Wi-Fi 接続時、それ以外は light（12.2）';

  @override
  String get lockApp => 'アプリをロック';

  @override
  String get relockNote => 'ロックしました。アプリを再起動してください。';

  @override
  String get languageTitle => '言語';

  @override
  String get languageSystem => 'システム';

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
  String get recoveryTitle => 'リカバリーカード';

  @override
  String get recoveryIntro =>
      'このデータは携帯電話の外に保存してください：デバイスを紛失したとき、ウォレットを復旧するのに必要です（ガーディアンによるマイグレーション、第 11 節）。';

  @override
  String get recoveryAddress => 'ウォレットアドレス';

  @override
  String get recoveryNetwork => 'ネットワーク';

  @override
  String get recoveryGuardians => 'あなたのガーディアン';

  @override
  String get recoveryNoGuardians =>
      'ガーディアン未指定：復旧は不可能で、リセットだけが残されています（新しい保証による通常の登録）。';

  @override
  String get recoveryHowToSave =>
      '保存方法：紙に写すか、携帯電話の外の安全な場所に保管してください；クラウドには保存しないでください。カードの画像は「共有」から印刷できます。';

  @override
  String get recoveryNeverShare =>
      'マイグレーションデータ（nonce とガーディアン署名）は決して共有しないでください——自分のガーディアンに直接、マイグレーション中だけ渡します；終わったらメッセージから削除してください。他人に見せてよいのはアドレスとこのカードの QR だけです。';

  @override
  String get recoverySavedDone => '保存しました——続ける';

  @override
  String get settingsRecovery => 'リカバリーカード';

  @override
  String get recoveryApkNote =>
      '同じネットワークのインストールファイル（APK）を事前に保存してください：なければ、新しいデバイスでの復旧は不可能になることがあります。';

  @override
  String get recoveryShareBtn => 'カードを共有（画像）';

  @override
  String get guardiansTitle => 'ガーディアン';

  @override
  String get guardiansIntro =>
      '個人ガーディアンはマイグレーションを確認します（2/3）。改選できます：変更は 48 時間後に有効になり、それまでは旧リストが有効です。';

  @override
  String get guardiansEntrySub => '個人ガーディアン：改選、48 時間ウィンドウ';

  @override
  String get guardiansCurrent => '現在のガーディアン';

  @override
  String get guardiansNone => 'ガーディアン未指定';

  @override
  String get guardiansReelect => '改選';

  @override
  String guardiansLimitUsed(int used, int limit) {
    return '使用済みガーディアン変更：$used / $limit';
  }

  @override
  String guardiansPendingSub(int hours) {
    return '新しいリストは $hours 時間後に有効になります';
  }

  @override
  String get guardiansCancelBtn => '変更を取り消す';

  @override
  String get guardiansStatusCanceled => '取り消し済み';

  @override
  String get guardiansStatusEffective => '有効になりました';

  @override
  String get guardiansStatusExpired => '失効：ガーディアンの被後見人上限を超過';

  @override
  String get reelectTitle => 'ガーディアンを改選';

  @override
  String get reelectIntro =>
      '新しいガーディアンを入力してください（最大 3 名、hex、1 行に 1 件）。変更ハッシュを各新ガーディアンに渡してください：相手は自分のアプリで署名し（「ガーディアンの同意」）、署名の hex を返します。';

  @override
  String get newGuardiansLabel => '新しいガーディアン（hex、1 行に 1 件）';

  @override
  String get changeHashToCopy => '変更ハッシュ（新ガーディアンに渡す）：';

  @override
  String guardiansApprovals(int count, int need) {
    return '収集した同意：$count / $need';
  }

  @override
  String get btnPublishChange => '変更を公開';

  @override
  String get changePublished => '変更を公開しました：48 時間後に有効';

  @override
  String get changeConfirmTitle => 'ガーディアン変更を確認（PIN）';

  @override
  String get cancelChangeConfirmTitle => '変更の取消を確認（PIN）';

  @override
  String get changeCanceled => '変更を取り消しました';

  @override
  String get consentTitle => 'ガーディアンの同意';

  @override
  String get consentIntro =>
      'あなたはこの被後見人のガーディアンになることに同意します。被後見人のアドレスと変更ハッシュを入力し、ハッシュに署名してください。署名の hex を被後見人に返します。';

  @override
  String get wardAddressLabel => '被後見人のアドレス（hex）';

  @override
  String get changeHashLabel => '変更ハッシュ（hex）';

  @override
  String get consentSignTitle => '被後見人のガーディアン変更を確認（あなたがガーディアンです）';

  @override
  String get biometricTimeout => '生体認証が応答しません（タイムアウト）——再試行してください';

  @override
  String hexChunksCount(int n, int total) {
    return '$total 中 $n 入力済み';
  }

  @override
  String hexChunksSigCount(int n) {
    return '入力した文字数：$n';
  }

  @override
  String hexLenError(int total) {
    return '正確に $total 文字の十六進（hex）が必要です';
  }

  @override
  String get hexSigEvenError => '署名は偶数個の十六進（hex）文字である必要があります';

  @override
  String get hexCharsDropped => '無効な文字を削除しました（0-9、a-f のみ）';

  @override
  String get aboutNetwork => 'ネットワークについて';

  @override
  String networkSlug(String slug) {
    return 'slug：$slug';
  }

  @override
  String get paramRecurring => '周期単位（recurring）';

  @override
  String get recurringOn => '有効（周期境界で失効）';

  @override
  String get recurringOff => '無効';

  @override
  String get recurringNote =>
      '各単位には周期マークが付いています：周期 P の単位の送金は、P+1 期の境界以降に拒否されます。残高はリセットされません——履歴は公開かつ検証可能なままです。';

  @override
  String get burnedUnitError =>
      '単位は失効しました：P+1 期の境界を過ぎました。送金も引き換えもできませんが、残高はリセットされず——履歴は保持されます（E_BURNED）。';

  @override
  String obNetworkLine(String pair) {
    return 'ネットワーク：$pair';
  }
}
