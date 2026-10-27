# Trust Networks Platform — Part 3. Identity and protection

> Unofficial English translation; in case of any discrepancy, the Russian original prevails.

[← Contents](KNOWLEDGE-ENG.md) · Part 3 of 5 · Sections 7–8

---

## 7. Identity, vouching, farm protection

### 7.1 What is achievable ✅

- A network rule, not an identity rule: the wallet is bound to a device, not to a person.
- It is technically impossible to prove "this specific phone" without an external anchor. Known bypasses:
  - `androidId` is unique to the combination "app signing key + user + device": a second Android user, Secure Folder, a work profile, and clones give a separate `androidId` on the same phone; a factory reset changes it;
  - hardware attestation confirms "a real device with protected hardware and an unmodified boot chain", but not the uniqueness of the device;
  - ID attestation (the serial number, IMEI) is unavailable to an ordinary application.
- Therefore "one wallet per phone" is a **cost barrier**, not an absolute ban. The target criterion: creating an extra wallet must cost more than it brings in. The main protection layer is vouching; attestation and PoW complement it.

### 7.2 Implementation

- The wallet creates a key pair in Android Keystore with a hardware-attestation request (`setAttestationChallenge`). The private key is non-exportable.
- `walletFingerprint = SHA256(attestedPublicKey ‖ protocolId)`, where the public key is confirmed by the attestation chain.
- The attestation chain is verified **locally by every node** against the Google root certificates embedded in the APK. What is checked: the hardware level (StrongBox/TEE), `verifiedBootState = Verified`, `deviceLocked = true`, a non-expired patch level.
- The list of revoked attestation certificates is distributed as signed data through the network (not through a server).
- `androidId` is only a weak additional signal (it raises the registration threshold for repeated values), not a ground for rejection.
- One fingerprint = one address; a repeated `register` with the same fingerprint and a different address is rejected. Node index: `fingerprint → address`.
- ✅ By default (`device_policy = strict`) only devices with hardware attestation are allowed (Knox/StrongBox and the like); Play Integrity and Google servers are not used. `relaxed` is acceptable for networks with low stakes; the guarantee level is visible in the manifest.
- ✅ A second wallet is banned at the application level.
- ✅ The "one phone — one wallet" rule applies **within a single network** (per-network; clarified October 27, 2026). The address = `SHA-256(fingerprint ‖ network_id)`, so the same phone yields different addresses in different networks — and in each network it is "one wallet per phone". Wallets of different networks can live on the same phone simultaneously: in v1 each network is its own APK (the constructor: a genesis package + an application), one wallet in each. The networks are isolated, cross-network replay is impossible; a double benefit (two quotas) is achievable only inside a single network.
- 🆕 The `region` field (the country by MCC/MNC) in `register` is not introduced in v1: it gives only crude statistics, and for vulnerable groups it is an extra signal. IMSI/ICCID are never written to the registry.
- Limitations left explicitly: root and emulators bypass `androidId`, but not hardware attestation; several phones = several wallets (this costs money and time); a factory reset = a new wallet unless migration has been performed (Section 11).

### 7.3 Wallet states

| Status       | How reached                            | Capabilities                                                                                                                                                  |
| ------------ | -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `registered` | After `register` (attestation + PoW)   | Receives transfers; issuance unavailable; `transfer` under the moratorium                                                                                      |
| `active`     | K vouches from different wallets       | Issuance, ordinary transfers                                                                                                                                   |
| `expired`    | 3 years in `registered` without activation ✅ | Issuance unavailable; a new `register` with a new key and new vouches is needed; the transfers already received remain at the address and are accessible to the key owner |

The `flagged` mark is a separate attribute (Section 7.5).

### 7.4 Vouching ✅

**The idea.** Sybil protection relies on social ties that cannot be bought in one go.

- A `vouch` transaction is signed by the vouching member and contains the candidate's address and the epoch. Activation requires **K = 3** (a parameter) `vouch` transactions from different wallets.
- Conditions for the vouching member: `active` status, an age ≥ 3 months, at least 2 completed issuance periods, free slots. Verified by every node.
- **Slots (the pool).** A vouching member has a limited number S of simultaneously occupied slots. A slot is freed when the vouched-for member completes 3 issuance periods without `evidence`, or after 90 days if the vouched-for member never became `active`. The pool size by age: 3 slots at 3–12 months, 5 slots above 12 months; 10 slots for the seed circle in the first year.
- The vouching graph is public; the payment recipient sees the sender's age, status, and "depth" and decides themselves whether to accept a payment. Graph analysis (distance to the seed circle, link density, in the spirit of SybilRank) is an auxiliary reputation layer; the result is visible to the recipient, there are no automatic verdicts ✅.
- **There is no deposit** ✅: vouching members must be incentivized, not punished.

**Vouching-member incentives** (only with `issuance_mode = periodic_equal`)

- Of the vouched-for member's first 12 issuances, R = 10% is automatically split equally between their K vouchers (with K = 3 that is about 3.3% each, roughly 33 units per issuance at the base emission).
- The source is the vouched-for member's own issuance, not new emission: the total emission does not grow. In the first year the vouched-for member receives 90%.
- Farm-neutral: if a farm vouches for its own wallets, the share returns to it. Real profit goes only to those vouching for other people.
- The payout stops from the epoch of finalization of an `evidence` against the vouched-for member. This is the only "punishment": the vouching member loses the reward, not units.
- The vouching member's public statistics (how many they vouched for, how many of them are active) is a reputational bonus without money.
- The reward size is a trade-off: the larger it is, the more actively people vouch and the stronger the temptation to hand out vouches to everyone. Vouches can be sold even without rewards; what restrains this is slots and the open graph.

**Network growth speed**

- Maximum growth per slot-turnover period (about a quarter): ×(1 + S/K). With K = 3: S = 2 → ×1.67, S = 3 → ×2, S = 5 → ×2.67 per quarter.
- The theoretical ceiling of the time from 100 seed wallets: up to 10,000 — about 2.3 / 1.7 / 1.2 years, up to 1 million — about 4.5 / 3.3 / 2.3 years (for S = 2 / 3 / 5).
- Real growth is limited above all by people's willingness to vouch. The startup bottleneck is the vouching-member age threshold and the seed-circle size.
- **Honest about the protection:** slots do not slow a farm down relative to an honest network — both grow toward the same ceiling. They protect against the instant creation of thousands of wallets, but not against a slow ring. Therefore slots can be chosen generously. ✅ Confirmed by simulation (Section 7.7): generous slots (5/8/15) give a smaller farm share than narrow ones (2/2/10) at the same honest-growth speed.

**Entry for people without acquaintances in the network**

- A vouch can be arranged in person via QR or Bluetooth, but presence is **not considered proven**: the vouching member answers for their choice with a limited slot and a possible loss of the reward.
- Public community channels are needed where newcomers find vouching members.

**Honest limitations**

- A ring of real-device owners vouching for each other is not forbidden by the rules; slots and the age threshold slow down its growth but do not stop it.

### 7.5 A proven violation: who and how ✅

**Who.** Nobody issues a verdict. The evidence is submitted by any node, and all nodes verify it identically and independently. There are no judges and no votes on guilt.

**A closed list of evidence** (only what is verified mathematically from signatures):

1. **Equivocation:** two transactions signed by the same wallet with the same `wallet` and `seq` (or `prev`), but with different content.
2. **Double issuance within one period** — a special case of equivocation (two `mint` of the same `period`).

Everything else ("looks like a farm", "suspicious activity") is not evidence. A second wallet on the same phone with a different key cannot be proven: vouching and cost protect against it.

**How.** Any node publishes an `evidence` transaction with two embedded transactions. Every node verifies: both signatures are valid, the wallet is the same, the `seq`/`prev` match, the hashes differ, both transactions come after `register`. Verification does not depend on timing or arrival order, so all nodes reach the same conclusion.

**Consequences (the soft scheme, recommended 🆕):**

- the wallet receives a permanent public `flagged` mark; both conflicting branches are not accepted, the branch chosen by finalization remains; issuance is suspended for N months, on a repeat — forever;
- the vouching members stop receiving a share of that wallet's issuance; there are no monetary fines and no loss of slots;
- there is no reward for submitting evidence, otherwise a farm would submit evidence against its own wallets and collect the reward.

**Accidental equivocation.** A real scenario: the app is restored from a backup, the local `seq` rolls back, the wallet signs again. Protection: database backup is disabled (`allowBackup=false`); before signing, the wallet obtains the current `seq` from several peers; without a network response, signing is impossible. The exception is a Bluetooth payment without internet, where a local `seq` is used.

**What the scheme does not catch.** An honestly working farm: its wallets do not double-spend. What works against a farm is the slot pool, the age threshold, the moratorium, the cost of devices, the fee, and the open graph.

### 7.6 Farm protection ✅

Relevant for the profiles where units acquire external value (3, 5, 7).

- **Devices:** hardware attestation; root on such devices burns Knox, so a farm = a farm of expensive devices.
- **The issuance challenge:** reinforced PoW in a random slot within the month with a ±48 h window. The slot `H(address ‖ period ‖ hash(previous epoch's checkpoint))` is unpredictable in advance. The PoW is memory-hard (Argon2id-like, parameters tuned for an average phone): it limits the GPU/ASIC advantage, but does not give full parity.
- A missed challenge: the accrual moves within the `accrual_window`, the funds are not blocked.
- **PoW** is reinforced on `register` and on every issuance; light on `transfer`.
- **The moratorium:** `transfer` from a young wallet becomes available 7–30 days after `register` (a parameter).
- **Reputation without verdicts:** age, issuance history, and incoming transfers from old wallets are visible in the registry; the recipient decides.
- **The security metric:** "the farm's cost per wallet" (device + electricity + time + the price of a vouch: social risk, limited slots, 10% of the first year's issuance, the fee on withdrawal) versus "income per wallet". The goal: income < cost under realistic estimates. The calculation is published together with the network parameters. ⚠️ The simulation (Section 7.7) refined the horizon: the farm's break-even price falls and drops below $0.01 by month 38, i.e. the goal is achievable not forever, but for the first ~3 years; check the metric over a horizon ≥ 3 years; do not disable demurrage and the fee in profiles with transferable value.
- Acknowledged: a guarantee of a human's presence without an external anchor is impossible; the protection is built on the farm's price. The argument "a farm will crash the price" is not considered protection (the commons dilemma).

### 7.7 Farm-economy simulation results ✅

The model: [equality-anti-farm-simulation](https://github.com/rumatakira74/equality-anti-farm-simulation) (Python; averaged over 8 runs, a 60-month horizon). Modeled: the growth of an honest network and of a farm: slots, age thresholds, K, the R reward, the cost of devices, and purchased vouches. Signatures, the network, the registry, PoW, the moratorium, demurrage, and the fee **were not modeled** — the farm economy in the model is overstated rather than understated.

**The base scenario:** K = 3, slots 3/5/10, a seed circle of 100, a farm of 20 devices/month (a cap of 2000 wallets).

1. **The farm does not start without penetration.** If the farm does not buy up vouches from honest members, its ring does not activate a single wallet in 60 months. The farm's mandatory condition is a purchased entry; that is the main cost barrier.
2. **The farm's share is temporary.** The peak is ~40% in month 27; by month 60 the honest network overtakes the farm (12.1%). A growing honest network is itself a protection; a non-growing network is more vulnerable.
3. **K = 3 confirmed.** The farm's share at 60 months: K = 2 → 9.6%, K = 3 → 12.1%, K = 4 → 15.2%, K = 5 → 23.2%, while honest growth at K = 5 is 2.2 times slower (3721 versus 8348 `active`).
4. **Generous slots — confirmed.** Generous slots (5/8/15) give a smaller farm share (11.9%) than narrow ones (2/2/10 — 14.0%) at the same honest-growth speed.
5. **A seed circle ≥ 300.** The farm's share at 60 months: seed 50 → 21.4%, 100 → 12.1%, 300 → 5.4%, 1000 → 5.4% (the effect saturates after ~300). Recommendation: a seed circle of at least 300 wallets.
6. **The farm's scale changes the share, not the economics.** A budget of 5 devices/month → a 3.3% farm share, 100 devices/month → 40.5%, yet the break-even price per wallet by month 60 barely changes ($0.005–0.006): scale changes the share, not the cost of a wallet.
7. ⚠️ **Honest about the payback time.** The farm's break-even price falls monotonically and crosses $0.01 in month 38: at a unit price of $0.01 the base farm is unprofitable for the first ~3 years, then goes into profit. Consequences: demurrage and the burn fee in profiles 5 and 7 are part of the protection, not an option; check the "income < cost" metric over a horizon ≥ 3 years; limit the accumulation horizons (`accrual_window`).

**Assumptions requiring calibration on pilots:** recruit = 0.1 newcomers/person/month; vouch_prob = 0.5 on contact; $200 per device. The numbers are sensitive to these values.

---

## 8. Keys and owner protection

### 8.1 Keys and authentication ✅

- The wallet key is in Android Keystore, export is impossible; the fingerprint image never reaches the registry or the network.
- Key parameters:
  - `setUserAuthenticationRequired(true)` with authentication **for every operation** (CryptoObject), without a time window;
  - with `signing_auth = biometric_strong`, only `BIOMETRIC_STRONG` is allowed (no device PIN/password);
  - `setInvalidatedByBiometricEnrollment(true)`: adding a new fingerprint invalidates the key. This is safer, but it means a loss of access, which is why guardians must be offered at wallet creation.
- In networks with `signing_auth = device_credential` (for example, educational ones for children), signing by device PIN/password is allowed; this is reflected in the manifest.
- App access: a PIN (an ordinary one and, with coercion protection enabled, a duress one). If protection is disabled, fingerprint unlock is allowed.
- A device without configured biometrics: registration is not allowed; the app offers to configure biometrics independently (the system enrollment screen, then re-verification).

### 8.2 Owner coercion: the duress PIN ✅

The threat: the owner is forced to open the app or send units.

- Two PINs: the ordinary one and the duress one. The duress PIN opens a **decoy mode**: the interface is the same, but a plausible small balance and history are shown. Showing zero is not allowed: an active wallet regularly receives issuance, and a zero balance looks suspicious.
- **Fingerprint unlock is disabled when the duress PIN is enabled:** otherwise the coercer presses the owner's finger to the sensor.
- In decoy mode the signing key and biometrics are not used; sending ends with a plausible "awaiting confirmation" or a network error without a real publication; notifications do not show amounts; the real wallet keeps working in the background but is not visible in the interface.
- Both PINs are indistinguishable: the same opening time, interface, and attempt limit.
- **You can hide, but you cannot zero out:** the balance is the network's public state. The key is not deleted; "erase local data" is allowed (the database is restored from the network).
- **What is not protected.** The public registry: anyone you paid to or received from knows the address and sees the balance in any client. The decoy is verifiable: the address in decoy mode will not match the real one on the network. The technique works against quick coercion and robbery, not against a prepared attacker.

### 8.3 Delayed large transfers (the second layer) ✅

- A transfer above the threshold P (a share of the balance or an amount) is executed only after a fixed delay D = 48 h. During D, the owner can cancel it with a biometric signature. The rule is verified by all nodes; the recipient sees "awaiting execution".
- The rule is enabled for all wallets and cannot be disabled, but it applies only in networks with transferable units (`transfer_mode = free`); it does not apply in the sandbox, for `soulbound`, `restricted`, and quotas.
- Small transfers go without a delay. Bluetooth payments above the threshold are subject to the same delay.
- The delay D is fixed and is not changed by the owner. The owner can change P within `P_max`; raising P takes effect after 48 h (otherwise the coercer first raises the threshold), lowering it — immediately.
- ❓ The default value of P and `P_max`.

---

