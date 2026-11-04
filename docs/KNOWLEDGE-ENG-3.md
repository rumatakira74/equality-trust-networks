# Trust Networks Platform — Part 3. Identity and protection

> Unofficial English translation; in case of any discrepancy, the Russian original prevails.

[← Contents](KNOWLEDGE-ENG.md) · Part 3 of 5 · Sections 7–8

---

## 7. Identity, vouching, farm protection

### 7.1 What is achievable ✅

- A network rule, not an identity rule: the wallet is bound to a device, not to a person.
- It is technically impossible to prove "this specific phone" without an external anchor. Known bypasses:
  - `androidId` is unique to the combination "app signing key + user + device": a second Android user, Secure Folder, a work profile, and clones give a separate `androidId` on the same phone; a factory reset changes it;
  - hardware attestation confirms "a real device with protected hardware and an unmodified boot chain", but not the uniqueness of the device: after leaf binding (v9.12) the registered key must match the key of the attested leaf, yet one device can produce several attested keys — several wallets on a single device remain possible even after the fix;
  - ID attestation (the serial number, IMEI) is unavailable to an ordinary application.
- Therefore "one wallet per phone" is a **cost barrier**, not an absolute ban. The target criterion: creating an extra wallet must cost more than it brings in. The main protection layer is vouching; attestation and PoW complement it.

### 7.2 Implementation

- The wallet creates a key pair in Android Keystore with a hardware-attestation request (`setAttestationChallenge`). The private key is non-exportable.
- `walletFingerprint = SHA256(attestedPublicKey ‖ protocolId)`, where the public key is confirmed by the attestation chain.
- The attestation chain is verified **locally by every node** against the Google root certificates embedded in the APK. What is checked: the hardware level (StrongBox/TEE), `verifiedBootState = Verified`, `deviceLocked = true`, a non-expired patch level.
- ✅ Both Google roots are embedded (v9.12): the current RSA root and the new ECDSA P-384 root (Remote Key Provisioning); a chain is valid if it reaches any of them. Synthetic roots are allowed only in the sandbox (`sandbox = true`): a non-sandbox network with a synthetic root does not start (genesis validation).
- ✅ Leaf binding (v9.12): the SPKI of the leaf certificate equals the registered `attestedPublicKey` — the chain was issued for exactly this key, reusing someone else's chain is impossible (`E_ATTEST_KEY_MISMATCH`). The leaf's validity period is checked at the registration's `timestamp_claim` (leaves are short-lived; replay gives the same verdict as the live check). Honestly: attestation confirms a real device with protected hardware, but not the uniqueness of the device, and no measurement on live devices has been performed — the DISCLAIMER remains.
- The list of revoked attestation certificates is distributed as signed data through the network (not through a server).
- `androidId` is only a weak additional signal (it raises the registration threshold for repeated values), not a ground for rejection.
- One fingerprint = one address within a network: `address = SHA-256(fingerprint ‖ network_id)` — a single derivation point from the final `network_id` (v9.11). Addresses are not stored in the manifest or genesis: the seed circle and providers are defined by fingerprints (Sections 3.3, 6.2), so the self-certification `network_id = H(manifest ‖ genesis)` is not circular. A repeated `register` with the same fingerprint is rejected. Node index: `fingerprint → address`.
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

Everything else ("looks like a farm", "suspicious activity") is not evidence. A second wallet on the same phone with a different key cannot be proven: vouching and cost protect against it. The cost barrier, therefore, rests on the assumption "one device — one wallet", which the protocol does not verify; the robustness of the economic conclusion is verified by the wallets_per_device sensitivity grid (7.7).

**How.** Any node publishes an `evidence` transaction with two embedded transactions. Every node verifies: both signatures are valid, the wallet is the same, the `seq`/`prev` match, the hashes differ, both transactions come after `register`. Verification does not depend on timing or arrival order, so all nodes reach the same conclusion.

**Consequences (the soft scheme) ✅:**

- the wallet receives a permanent public `flagged` mark; both conflicting branches are not accepted, the branch chosen by finalization remains; issuance is suspended for 3 months (No. 12), on a repeat — forever;
- the vouching members stop receiving a share of that wallet's issuance; there are no monetary fines and no loss of slots;
- a `flagged` wallet cannot vouch for new members (No. 14);
- there is no reward for submitting evidence, otherwise a farm would submit evidence against its own wallets and collect the reward.

**Accidental equivocation.** A real scenario: the app is restored from a backup, the local `seq` rolls back, the wallet signs again. Protection: database backup is disabled (`allowBackup=false`); before signing, the wallet obtains the current `seq` from several peers; without a network response, signing is impossible. The exception is a Bluetooth payment without internet, where a local `seq` is used.

**What the scheme does not catch.** An honestly working farm: its wallets do not double-spend. What works against a farm is the slot pool, the age threshold, the moratorium, the cost of devices, the fee, and the open graph.

### 7.6 Farm protection ✅

Relevant for the profiles where units acquire external value (3, 5, 7).

- **Devices:** hardware attestation up to the embedded Google roots with leaf binding to the registered key (v9.12): software farms and emulators are cut off, root on such devices burns Knox. But one device can produce several attested keys, so "a farm = a farm of expensive devices" does not hold automatically: several wallets per device remain possible, and the barrier is held by vouching and cost (7.1).
- **The issuance challenge:** reinforced PoW in a random slot within the period with a ±48 h window (configured by field 51 `mint_challenge_window_hours`). The slot `period_start + H(address ‖ period) mod period_seconds` is publicly computable in advance — the app can show it to the user; including `hash(previous epoch's checkpoint)` in the formula is an open question (PENDING for the owner; the implementation is `H(address ‖ period)`). ✅ v9.13 (the time-fix stage): the window is checked against the tx's own `timestamp_claim` (like all time-dependent rules, 9.2); the slot is scaled by the period length `period_seconds` (field 61, 0 — the 30-day convention). ✅ v9.14 (the mint-fix topic, the owner's decision of November 3, 2026): the slot anchor is the CURRENT issuance period (the txNow period), not the claimed one: period P can only be claimed after P ends (10.1), so the claimed period's slot window intersected the allowed submission time only when the slot fell in the last 48 h of the period — reachability 1/15 per period, six consecutive misses produced an absorbing state (simulation A9: honest wallets −91.2% of income over 60 months). The current period's slot lies inside it: the intersection of the ±48 h window with the current period is always ≥ 48 h — every wallet has a reachable window in every period, and a missed period is compensated within the `accrual_window` (10.1). The reachability of the seed circle's first claimable periods with short `period_seconds` (quotas from period 1 near genesis, blocker 2 of the 12-testnet run) is set by field 51: the template ±48 h window does not cover them, a test network sets a larger window. The PoW is memory-hard (Argon2id-like, parameters tuned for an average phone): it limits the GPU/ASIC advantage, but does not give full parity.
- A missed challenge: the accrual moves within the `accrual_window`, the funds are not blocked.
- **PoW** is reinforced on `register` and on every issuance; light on `transfer`.
- **The moratorium:** `transfer` from a young wallet becomes available 7–30 days after `register` (a parameter).
- **Reputation without verdicts:** age, issuance history, and incoming transfers from old wallets are visible in the registry; the recipient decides.
- **The security metric:** "the farm's cost per wallet" (device + electricity + time + the price of a vouch: social risk, limited slots, 10% of the first year's issuance, the fee on withdrawal) versus "income per wallet". The goal: income < cost under realistic estimates. The calculation is published together with the network parameters. ⚠️ The price of a vouch is not accounted for in the simulation (7.7) (S-05 of the model audit): infiltration in the model is free and without a probability of honest refusal — with the vouch price included, the metric is more conservative, but calibration on pilots is needed. ⚠️ The simulation refined the horizon: the farm's break-even price falls and drops below $0.01 already in month 39 at the $120 canon per device (in month 53 at $200, in month 33 at wallets_per_device = 4 — the grid 7.7): the goal is achievable not forever, but for the first ~3–4.5 years depending on the grid row; check the metric over a horizon ≥ 5 years (with a margin to the grid); do not disable demurrage and the fee in profiles with transferable value.
- Acknowledged: a guarantee of a human's presence without an external anchor is impossible; the protection is built on the farm's price. The argument "a farm will crash the price" is not considered protection (the commons dilemma).

### 7.7 Farm-economy simulation results ✅

The model: [equality-anti-farm-simulation](https://github.com/rumatakira74/equality-anti-farm-simulation) (Python; averaged over 8 runs, a 60-month horizon; the canonical runs — the numpy path in a single environment with a SHA-256 manifest, S-08 of the model audit). Modeled: the growth of an honest network and of a farm: slots, age thresholds, K, the R reward, the cost of devices, and purchased vouches; the A9 recalculation (under the platform's actual parameters) added the PoW cost at the actual targets and the mint challenge window mechanics. Signatures, the network, the registry, the moratorium, demurrage, and the fee **were not modeled** — this understates the farm's economy. At the same time, the key assumption "one device — one wallet" (7.2, A1) is not verified by the protocol (7.5): if several attested keys are created on a device, the farm's economy in the model is **understated by multiples** (S-01 of the model audit). The owner's decision on S-01: the canonical device cost is $120 (conservative for the platform's conclusion: even with a cheap device the farm does not break even — the conclusion is robust); robustness to the "several wallets per device" bypass is verified by the sensitivity grid device_cost {120, 200} × wallets_per_device {1, 2, 4} (item 8).

**The base scenario:** K = 3, slots 3/5/10, a seed circle of 100, a farm of 20 devices/month (a cap of 2000 wallets), $120 per device — the canonical value; the published figures are the canonical numpy-path runs at $120.

1. **The farm does not start without penetration.** If the farm does not buy up vouches from honest members, its ring does not activate a single wallet in 60 months. The farm's mandatory condition is a purchased entry; that is the main cost barrier.
2. **The farm's share is temporary — at recruit ≥ 0.1.** The peak is 38.6% in month 28; by month 60 the honest network overtakes the farm (12.0%). A growing honest network is itself a protection; a non-growing network is more vulnerable. Caveat (S-06 of the model audit): at recruit = 0.05 the farm holds 47–54% at month 60 and does not retreat; churn and "dead" active wallets are not accounted for in the model.
3. **K = 3 chosen as a compromise.** K = 2 gives a peak of 42% (m20) and 9.6% at m60, K = 3 — 40% (m27) and 12.1%, K = 4 — 36% (m35) and 15.3%, K = 5 — 32% (m49) and 23.8% with honest growth 2.3 times slower (S-03 of the model audit: "confirmed" by the m60 share is wrong — K = 2 is better by it; a higher K lowers the peak but crushes honest growth).
4. **Slots — a "peak versus growth" compromise.** The farm's share at month 60 is nearly the same for any slots (the farm is budget-capped — 1,140 wallets; the difference in shares is created only by honest growth: narrow slots slow it by 15–17%). Relative to the dangerous moment — the peak — it is the other way around (S-02, 60 seeds): narrow slots (2/2/10) lower the farm's peak from 40.2% (m27) to 31.5% (m38) and delay it by 11 months; generous ones (5/8/15) raise the peak to 43.9% (m21) and bring it 6 months closer; by month 24 the farm's share with generous slots is twice as high as with narrow ones (42.5% versus 20.3%).
5. **A seed circle ≥ 300.** The farm's share at 60 months: seed 50 → 22.1%, 100 → 12.0%, 300 → 5.4%, 1000 → 5.4% (the canonical numpy-path runs at $120; they match A9 after the single RNG path was pinned). The 300/1000 match is an artifact of the model's ceiling on the number of honest wallets (20,000); the effect does not saturate: the farm's peak falls from 40% (seed 100) to 18% (300) and 6% (1000) (S-04 of the model audit; the month-60 figure is unfit for comparing seed circles). Recommendation: a seed circle of at least 300 wallets.
6. **The farm's scale changes the share, not the economics.** A budget of 5 devices/month → a 3.3% farm share, 100 devices/month → 40.4%, yet the break-even price per wallet by month 60 barely changes ($0.005–0.006 at the $120 canon; $0.0085 at $200 — the grid): scale changes the share, not the cost of a wallet.
7. ⚠️ **Honest about the payback time.** The farm's break-even price falls monotonically and crosses $0.01 in month 39 at the $120 canon (in month 53 at $200 — the grid): at a unit price of $0.01 the base farm is unprofitable for the first ~3.5 years, then goes into profit. The break-even price at 60 months: $0.0053 at $120 (the canon), $0.0085 at $200, $0.0125 at $300 (at $300 the crossing is beyond the 60-month horizon). Consequences: demurrage and the burn fee in profiles 5 and 7 are part of the protection, not an option; check the "income < cost" metric over a horizon ≥ 5 years; limit the accumulation horizons (`accrual_window`).
8. ⚠️ **The estimates assume 1 wallet per device** (after the A1 binding, 7.2) — the assumption is not verified by the protocol (7.5). The bypass — 2–4 wallets per device (Android user profiles, a cost of $2 per wallet; a factory reset, $25) — raises the farm share peak from 38.6% to 51.7–59.3% and lowers the break-even price at 60 months to $0.0016–0.0037 at the $120 canon ($0.0022–0.0044 at $200): A1 ("device ≈ wallet") holds only under a strict 1:1 wallet/device ratio. The sensitivity grid (without the bypass costs — it isolates the two parameters, conservatively cheapening the farm's wallet):

   | device_cost, $ | wallets per device | farm share peak, % (month) | break-even price m60, $/unit | $0.01 crossing, month |
   | --- | --- | --- | --- | --- |
   | 120 (canon) | 1 | 38.6 (28) | 0.0053 | 39 |
   | 120 | 2 | 51.7 (32) | 0.0028 | 34.5 |
   | 120 | 4 | 59.3 (35) | 0.0015 | 33 |
   | 200 | 1 | 38.6 (28) | 0.0085 | 53.5 |
   | 200 | 2 | 51.7 (32) | 0.0043 | 40 |
   | 200 | 4 | 59.3 (35) | 0.0022 | 35 |

   The conclusion is robust across the whole grid: the farm breaks even at a price of $0.01 in every row (the crossing is before month 60), and the $120 canon is the farm-generous boundary: even with a cheap device and a strict 1:1, the conclusion "does not break even for the first ~3.5 years" holds.
9. **PoW is a rate limiter, not a cost.** At the actual targets (hard 2^56, light 2^40; Argon2id 32 MiB × 1 iteration × p=1) the issuance PoW costs $2.9·10⁻⁶/month and 17.4 s of pure time per issuance — many orders of magnitude below the device cost. The hardware advantage over an average phone: GPU ≈182×, HBM-ASIC ≈506× — there is no full parity, but PoW is not a cost barrier either.
10. **The mint challenge window — after the v9.14 fix.** The slot anchor defect (reachability 1/15 per period, absorption after 6 consecutive misses) took 91.2% of an honest member's income over 60 months, while an automated farm was unaffected (the slot is publicly computable). The fix in main: the slot anchor is the current issuance period, and the ±48 h window always intersects the current period by ≥ 48 h — every wallet has a reachable window in every period; the absorbing state is eliminated (7.6).
11. **The "income < cost" metric** holds due to the device cost and purchased vouches — not due to PoW (item 9) and not due to the mint window (item 10).

**Assumptions requiring calibration on pilots:** recruit = 0.1 newcomers/person/month; vouch_prob = 0.5 on contact; $120 per device (the canon; $200 and wallets_per_device {1, 2, 4} — in the sensitivity grid, the `--a9` scenario). The numbers are sensitive to these values.

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
- In decoy mode the signing key is not used; the same biometric prompt is shown (plain, without the key — skipping the prompt would give the decoy away); sending ends with a plausible "awaiting confirmation" or a network error without a real publication; notifications do not show amounts; the real wallet keeps working in the background but is not visible in the interface.
- Both PINs are indistinguishable: the same opening time, interface, and attempt limit (5 consecutive failures lock input for 30 s, doubling up to 10 min; reset by a successful entry or 24 h without failures; key destruction on failures is not introduced).
- **You can hide, but you cannot zero out:** the balance is the network's public state. The key is not deleted; "erase local data" is allowed (the database is restored from the network).
- **What is not protected.** The public registry: anyone you paid to or received from knows the address and sees the balance in any client. The decoy is verifiable: the address in decoy mode will not match the real one on the network. The technique works against quick coercion and robbery, not against a prepared attacker.

### 8.3 Delayed large transfers (the second layer) ✅

- A transfer above the threshold P (a share of the balance or an amount) is executed only after a fixed delay D = 48 h. During D, the owner can cancel it with a biometric signature. The rule is verified by all nodes; the recipient sees "awaiting execution".
- The rule is enabled for all wallets and cannot be disabled, but it applies only in networks with transferable units (`transfer_mode = free`); it does not apply in the sandbox, for `soulbound`, `restricted`, and quotas.
- Small transfers go without a delay. Bluetooth payments above the threshold are subject to the same delay.
- The delay D is fixed and is not changed by the owner. The owner can change P within `P_max`; raising P takes effect after 48 h (otherwise the coercer first raises the threshold), lowering it — immediately.
- ❓ The default value of P and `P_max`.

---

