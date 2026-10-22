// Модели данных приложения (этап 7). Все значения приходят из
// платформенного слоя (мост mobile/ + net/query.go); приложение не
// вычисляет и не хардкодит параметры сети.
// ignore_for_file: constant_identifier_names

/// Параметры сети из манифеста genesis (раздел 4 KNOWLEDGE.md).
class ManifestParams {
  final String networkId;
  final String displayName;
  final String unit;
  final String transferMode;
  final String issuanceMode;
  final int issuanceAmount;
  final int issuancePeriodMonths;
  final int accrualWindowPeriods;
  final String feeMode;
  final int feeMin;
  final int transferMoratoriumDays;
  final int transferDelayHours; // D = 48 ч, фиксированная (8.3)
  final int delayedThresholdDefault; // P
  final int delayedThresholdMax; // P_max
  final int vouchK;
  final int voucherMinAgeDays;
  final int guardianPersonalThreshold; // 2 из 3 личных (11)
  final int guardianPersonalCount;
  final int guardianPublicThreshold; // 3 из 5 публичных (11)
  final int guardianPublicCount;
  final int guardianMinAgeDays;
  final int migrateDelayDays; // 14 дней (11)
  final int mintChallengeWindowHours; // ±48 ч (7.6)
  final int slotFreeMintPeriods; // 3 выпуска без evidence (7.4)
  final int slotFreeDays; // 90 дней без активации (7.4)
  final int demurrageRateMicros;
  final int demurrageFloor;
  final int epochSeconds;
  final bool sandbox;

  ManifestParams.fromJson(Map<String, dynamic> j)
      : networkId = j['network_id'] as String? ?? '',
        displayName = j['display_name'] as String? ?? '',
        unit = j['unit'] as String? ?? 'currency',
        transferMode = j['transfer_mode'] as String? ?? 'free',
        issuanceMode = j['issuance_mode'] as String? ?? 'periodic_equal',
        issuanceAmount = _i(j['issuance_amount']),
        issuancePeriodMonths = _i(j['issuance_period_months'], 1),
        accrualWindowPeriods = _i(j['accrual_window_periods']),
        feeMode = j['fee_mode'] as String? ?? 'burn',
        feeMin = _i(j['fee_min']),
        transferMoratoriumDays = _i(j['transfer_moratorium_days']),
        transferDelayHours = _i(j['transfer_delay_hours'], 48),
        delayedThresholdDefault = _i(j['delayed_threshold_default']),
        delayedThresholdMax = _i(j['delayed_threshold_max']),
        vouchK = _i(j['vouch_k'], 3),
        voucherMinAgeDays = _i(j['voucher_min_age_days']),
        guardianPersonalThreshold = _i(j['guardian_personal_threshold'], 2),
        guardianPersonalCount = _i(j['guardian_personal_count'], 3),
        guardianPublicThreshold = _i(j['guardian_public_threshold'], 3),
        guardianPublicCount = _i(j['guardian_public_count'], 5),
        guardianMinAgeDays = _i(j['guardian_min_age_days']),
        migrateDelayDays = _i(j['migrate_delay_days'], 14),
        mintChallengeWindowHours = _i(j['mint_challenge_window_hours'], 48),
        slotFreeMintPeriods = _i(j['slot_free_mint_periods'], 3),
        slotFreeDays = _i(j['slot_free_days'], 90),
        demurrageRateMicros = _i(j['demurrage_rate_micros']),
        demurrageFloor = _i(j['demurrage_floor']),
        epochSeconds = _i(j['epoch_seconds']),
        sandbox = j['sandbox'] as bool? ?? false;

  static int _i(dynamic v, [int dflt = 0]) =>
      v == null ? dflt : (v is num ? v.toInt() : int.tryParse('$v') ?? dflt);

  bool get demurrageEnabled => demurrageRateMicros > 0;
  bool get transfersDelayed =>
      transferMode == 'free' && !sandbox; // 8.3: только передаваемые
}

/// Снапшот кошелька (net/query.go -> WalletInfoJSON).
class WalletInfo {
  final bool exists;
  final String address;
  final String status; // REGISTERED / ACTIVE / EXPIRED
  final bool flagged; // метка показывается, вердиктов нет (7.5)
  final int ageDays;
  final int balance; // эффективный остаток (демередж материализован, 10.3)
  final int liquid;
  final int locked; // pending-резерв отложенных (8.3)
  final int seq;
  final int lastClaimedPeriod;
  final int thresholdP;
  final int pendingThreshold;
  final int pendingThresholdNotBefore;
  final int vouchesReceived;
  final int completedMintPeriods;
  final int vouchDepth; // расстояние до посевного круга (-1 нет)
  final List<String> personalGuardians;
  final List<String> activators;
  final bool publicGuardian;
  final int wards;
  final int guardianChanges; // вступивших в силу смен (лимит 150, 11)
  final int mintSuspendedUntilPeriod;
  final bool seed;
  final bool terminal;

  WalletInfo.fromJson(Map<String, dynamic> j)
      : exists = j['exists'] as bool? ?? false,
        address = j['address'] as String? ?? '',
        status = j['status'] as String? ?? '',
        flagged = j['flagged'] as bool? ?? false,
        ageDays = _i(j['age_days']),
        balance = _i(j['balance']),
        liquid = _i(j['liquid']),
        locked = _i(j['locked']),
        seq = _i(j['seq']),
        lastClaimedPeriod = _i(j['last_period']),
        thresholdP = _i(j['threshold_p']),
        pendingThreshold = _i(j['pending_threshold']),
        pendingThresholdNotBefore = _i(j['pending_threshold_not_before']),
        vouchesReceived = _i(j['vouches_received']),
        completedMintPeriods = _i(j['completed_mint_periods']),
        vouchDepth = _i(j['vouch_depth'], -1),
        personalGuardians = (j['personal_guardians'] as List? ?? [])
            .map((e) => '$e')
            .toList(),
        activators = (j['activators'] as List? ?? []).map((e) => '$e').toList(),
        publicGuardian = j['public_guardian'] as bool? ?? false,
        wards = _i(j['wards']),
        guardianChanges = _i(j['guardian_changes']),
        mintSuspendedUntilPeriod = _i(j['mint_suspended_until_period']),
        seed = j['seed'] as bool? ?? false,
        terminal = j['terminal'] as bool? ?? false;

  bool get active => status == 'ACTIVE';
  bool get registered => status == 'REGISTERED';
  static int _i(dynamic v, [int dflt = 0]) =>
      v == null ? dflt : (v is num ? v.toInt() : int.tryParse('$v') ?? dflt);
}

/// Запись истории цепочки (HistoryJSON).
class HistoryEntry {
  final int seq;
  final String type;
  final String txid;
  final int timestamp;
  final String recipient;
  final int amount;
  final int period;
  final String candidate;
  final String note;

  HistoryEntry.fromJson(Map<String, dynamic> j)
      : seq = _i(j['seq']),
        type = j['type'] as String? ?? '',
        txid = j['txid'] as String? ?? '',
        timestamp = _i(j['timestamp']),
        recipient = j['recipient'] as String? ?? '',
        amount = _i(j['amount']),
        period = _i(j['period']),
        candidate = j['candidate'] as String? ?? '',
        note = j['note'] as String? ?? '';

  static int _i(dynamic v) => v is num ? v.toInt() : 0;

}

/// Отложенный перевод (8.3).
class PendingTransfer {
  final int seq;
  final String recipient;
  final int amount;
  final int fee;
  final int deadline;
  final int remainingSec;
  final bool canceled;
  final bool executed;

  PendingTransfer.fromJson(Map<String, dynamic> j)
      : seq = _i(j['seq']),
        recipient = j['recipient'] as String? ?? '',
        amount = _i(j['amount']),
        fee = _i(j['fee']),
        deadline = _i(j['deadline']),
        remainingSec = _i(j['remaining_sec']),
        canceled = j['canceled'] as bool? ?? false,
        executed = j['executed'] as bool? ?? false;

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Незавершённая миграция (11).
class PendingMigration {  final String newAddress;
  final String oldAddress;
  final int proposeSeq;
  final int notBefore;
  final int remainingSec;
  final int nonce;
  final bool canceled;
  final bool executed;
  final List<String> approvals;

  PendingMigration.fromJson(Map<String, dynamic> j)
      : newAddress = j['new_address'] as String? ?? '',
        oldAddress = j['old_address'] as String? ?? '',
        proposeSeq = _i(j['propose_seq']),
        notBefore = _i(j['execute_not_before']),
        remainingSec = _i(j['remaining_sec']),
        nonce = _i(j['nonce']),
        canceled = j['canceled'] as bool? ?? false,
        executed = j['executed'] as bool? ?? false,
        approvals = (j['approvals'] as List? ?? [])
            .map((a) => ((a as Map)['guardian'] ?? '') as String)
            .toList();

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Смена гардианов (11, решение владельца 21.10.2026): окно 48 ч,
/// отмена в окне, вступление или истечение по лимиту №7.
class PendingGuardianUpdate {
  final int proposeSeq;
  final List<String> newGuardians;
  final int effectiveAt;
  final int remainingSec;
  final int nonce;
  final bool canceled;
  final bool effective;
  final bool expired;

  PendingGuardianUpdate.fromJson(Map<String, dynamic> j)
      : proposeSeq = _i(j['propose_seq']),
        newGuardians = (j['new_guardians'] as List? ?? [])
            .map((e) => '$e')
            .toList(),
        effectiveAt = _i(j['effective_at']),
        remainingSec = _i(j['remaining_sec']),
        nonce = _i(j['nonce']),
        canceled = j['canceled'] as bool? ?? false,
        effective = j['effective'] as bool? ?? false,
        expired = j['expired'] as bool? ?? false;

  bool get active => !canceled && !effective && !expired;

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Окно челленджа выпуска (7.6).
class MintWindow {
  final bool open;
  final int windowHours;
  final int accrualWindowPeriods;
  final int issuanceAmount;
  final int currentPeriod;
  final int claimPeriod;
  final int claimAmount;
  final int lastClaimedPeriod;

  MintWindow.fromJson(Map<String, dynamic> j)
      : open = j['open'] as bool? ?? false,
        windowHours = _i(j['window_hours']),
        accrualWindowPeriods = _i(j['accrual_window_periods']),
        issuanceAmount = _i(j['issuance_amount']),
        currentPeriod = _i(j['current_period']),
        claimPeriod = _i(j['claim_period']),
        claimAmount = _i(j['claim_amount']),
        lastClaimedPeriod = _i(j['last_claimed_period']);

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Слот поручителя (7.4).
class VouchSlot {
  final String candidate;
  final int since;
  final bool candidateActive;

  VouchSlot.fromJson(Map<String, dynamic> j)
      : candidate = j['candidate'] as String? ?? '',
        since = _i(j['since']),
        candidateActive = j['candidate_active'] as bool? ?? false;

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Слоты поручителя (7.4).
class SlotsInfo {
  final bool exists;
  final int capacity;
  final List<VouchSlot> occupied;
  int get free => capacity - occupied.length;

  SlotsInfo.fromJson(Map<String, dynamic> j)
      : exists = j['exists'] as bool? ?? false,
        capacity = _i(j['capacity']),
        occupied = (j['occupied'] as List? ?? [])
            .map((e) => VouchSlot.fromJson((e as Map).cast<String, dynamic>()))
            .toList();

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Участник реестра (9.1).
class RegistryRow {
  final String address;
  final String status;
  final int ageDays;
  final bool flagged;

  RegistryRow.fromJson(Map<String, dynamic> j)
      : address = j['address'] as String? ?? '',
        status = j['status'] as String? ?? '',
        ageDays = _i(j['age_days']),
        flagged = j['flagged'] as bool? ?? false;

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Публичный гардиан (11).
class GuardianOffer {
  final String address;
  final int ageDays;
  final int wards;
  final String status;
  final bool flagged;

  GuardianOffer.fromJson(Map<String, dynamic> j)
      : address = j['address'] as String? ?? '',
        ageDays = _i(j['age_days']),
        wards = _i(j['wards']),
        status = j['status'] as String? ?? '',
        flagged = j['flagged'] as bool? ?? false;

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Офлайн-платёж, ожидающий публикации (12.4).
class AwaitingTx {
  final String txid;
  final String type;
  final String recipient;
  final int amount;
  final String status;

  AwaitingTx.fromJson(Map<String, dynamic> j)
      : txid = j['txid'] as String? ?? '',
        type = j['type'] as String? ?? '',
        recipient = j['recipient'] as String? ?? '',
        amount = _i(j['amount']),
        status = j['status'] as String? ?? '';

  static int _i(dynamic v) => v is num ? v.toInt() : 0;
}

/// Результат публикации транзакции.
class SubmitResult {
  final String outcome;
  final String code;
  final bool fake; // приманка (8.2): приложение не различает
  SubmitResult(this.outcome, this.code, {this.fake = false});

  bool get accepted => outcome == 'accepted' || outcome == 'accept';
}

extension ShortHash on String {
  /// Короткое представление адреса для UI.
  String shortHash() =>
      length <= 12 ? this : '${substring(0, 6)}…${substring(length - 6)}';
}
