# Trust Networks Platform — Part 4. Registry, units, and network

> Unofficial English translation; in case of any discrepancy, the Russian original prevails.

[← Contents](KNOWLEDGE-ENG.md) · Part 4 of 5 · Sections 9–12

---

## 9. The registry

### 9.1 A chain per wallet + voting + checkpoints ✅

**Structure.** Each wallet has its own chain. A transaction contains `wallet`, `seq` (strictly +1), `prev` (the hash of the previous one), `type`, `payload`, `timestamp_claim`, a signature.

**Why this way.**

- "One wallet — one balance" maps naturally onto a chain: a double spend = two transactions with one `seq`/`prev`; this is trivially detected and proven.
- The issuance period and the order of operations are determined by `seq`, not by the position in a shared graph.
- Confirmations do not depend on other users' traffic.

**Conflicts.** Nodes resolve a conflict by sampled voting (Snowball/Avalanche-like): repeated polling of a random sample of peers, the weights by wallet age and attestation. The branch chosen by finalization wins; a wallet caught in equivocation receives the `flagged` mark.

**Finality.** Periodic checkpoints (epochs): a hash of the state root (a sparse Merkle tree of balances and `seq`) signed with a threshold signature. A checkpoint gives a deterministic before/after order, safe snapshots for light nodes, and time for the network.

**Checkpoint modes** (`checkpoint_mode`):

- `decentralized` — the signature of a sample of the epoch's validators;
- `members_quorum` — the signature of ≥ 2/3 of the members (for small networks).

**The transitional cold-start mode.** In a small network, voting is unreliable. The first checkpoints are signed by a fixed set of seed-circle keys (N-of-M) with a hard shutdown schedule (`seed_sunset`). This is a temporary compromise against the decentralization goal; it must be described publicly.

**Requires verification before implementation.** A simulation (at least on hundreds of nodes) of conflicts, equivocation, and loss of connectivity.

### 9.2 Time and epochs ✅

- Time is network time, not device time; NTP/GPS — calibration and diagnostics only.
- The source of truth is the checkpoint epochs, not the median of timestamps (with low traffic it can be shifted).
- The calendar is UTC; "the 1st of the month" = the first epoch that started after 00:00 UTC on the 1st by consensus time.
- A claimed timestamp is accepted with a deviation from network time of no more than ±30 minutes; it does not affect the issuance period.
- Network time cannot shift between checkpoints by more than a set limit.

### 9.3 Transaction types

- Core: `register`, `vouch`, `mint`, `transfer`, `migrate`, `evidence`, `guardian_offer`.
- Modules: `pool_create`, `pool_donate`, `pool_close`, `pool_claim`, `redeem`/`receipt`, `contribution_claim`/`contribution_confirm` (Section 6).
- The balance = the sum of valid history, recomputed by every node. The issuance rules are verified by every node, not only by the app.
- A light PoW on every transaction + a small fee (Section 10.2).

---

## 10. Units, issuance, stock

The values in this section are the default "Currency" profile; the other profiles override the unit, the issuance mode, transfer, and stock (Sections 4–5).

### 10.1 `periodic_equal` issuance ✅

- Base issuance of 1000 units, on the 1st of each month, a single date for everyone.
- The first `mint` = `period − 1` (no retro-accruals). The `period` of a new issuance is strictly greater than the previous one; `amount = base_emission × periods_elapsed`.
- Issuance is available only to a wallet in `active` status.
- **Accrual for missed months is limited by a window** (`accrual_window`, 6 months by default): unlimited accrual creates "sleeping" accruals and an incentive for a delayed attack. The units are not blocked: beyond the window, the accrual simply burns.
- The emission is constant: 1000 × the number of `active` wallets per month; a decreasing emission is not used.
- An offline duplicate conflict: the first one by finalization order wins; the second is voided.
- Freshly issued funds are not transferable immediately (part of the moratorium, Section 7.6): this cuts off "issue → dump".

### 10.2 The fee ✅

- Every `transfer` pays a fee that is **burned**: `fee = max(fee_min, rate × amount)`; the sender pays it on top of the amount, the recipient receives exactly `amount`.
- `fee_min` is fixed and at the same time protects against spam.
- `rate` is adaptive: at the epoch boundary it is recomputed deterministically from the checkpoint data. If the supply growth over the last K epochs is above the target, `rate` rises by a step, otherwise it falls; the bounds `[rate_min, rate_max]` are set by the protocol. All nodes obtain the same value.
- **What the fee solves and what it does not.** It burns only *movement*, but not *holding*. A rough equilibrium estimate: the supply per wallet ≈ `1000 / (rate × turnover)`, where turnover is the share of the supply transferred per month (at `rate` = 1% and a turnover of 100% per month — about 100,000 per wallet; at 20% — about 500,000). If units are hoarded and not spent, the stock is weak.

### 10.3 Demurrage ✅

- The only mechanism that penalizes precisely holding. A reserve lever in the base profile (disabled); enabled by default in profiles 5 and 7. The values are to be tuned by simulation.
- Parameters: `demurrage_rate` (a fraction per month) and `demurrage_floor` (a tax-free remainder).
- There are no deduction transactions: the effective balance is computed deterministically at the wallet's next operation by the number of elapsed epochs n: `balance_eff = floor + (balance − floor) × (1 − rate)^n` with `balance > floor`; the deducted amount is burned.

### 10.4 Economics ⏳

The parameters of the base economy (the bounds and the step of the adaptive fee, the target supply growth, the "farm versus income" model, the demurrage rate) are to be tuned by simulation, deferred.

---

## 11. Wallet recovery

- **Personal guardians: 2 of 3.** The guardians are appointed in advance (at wallet creation), not at the moment of loss. Transaction type `migrate`.
- **Migration:** a 14-day delay, during which the old key (if alive) can cancel it; the new wallet passes the same attestation as at `register`; age, issuance history, `period`, vouches, **and the balance are transferred** (otherwise migration bypasses the moratorium and the limits); the old fingerprint is blacklisted forever (otherwise the "migrate → register again" cycle yields unlimited units); a guardian cannot be a guardian for more than K wallets.
- **If there are no guardians — two paths ✅:**
  - **Public guardians:** an `active` wallet aged ≥ 12 months declares itself a guardian (`guardian_offer`). The user chooses them in advance; with public guardians participating, the threshold is 3 of 5 (any mix of personal and public ones). The reward is a fixed sum from new emission for each successful `migrate` of a ward (the size ❓, a guideline of no more than 10% of monthly issuance; no more often than once per 12 months per ward; a limit of G_max = 20 wards per guardian; paid only to the signers and only after the delay without a cancellation). Applicable in networks with emission.
    - Risk: a reward from emission is a subsidy to farms "migrating" their own wallets; the frequency limit, the delay, and the migration PoW must make this unprofitable; verify the size by simulation.
    - Risk: collusion of public guardians against an owner who has lost the device. Protection: the 3-of-5 threshold, the 14-day delay, public signatures. ❓ Whether to require at least one personal guardian.
  - **A restart by the user's decision:** an ordinary `register` with a new key and new K vouches. The old wallet remains in the registry untouched and inaccessible; the funds on it are lost. A restart gives nothing beyond registering a new wallet.

### Known limitations of v1

- **Changing the personal guardian list is not supported** (decision D4): the guardians are appointed at `register` (wallet creation) and remain unchanged for the wallet's entire lifetime; a way to update the list afterwards is absent from the Document and does not fit into the closed list of transaction types (№6).
- **Migration cancellation is the old key's right within the 14-day window** — a deliberate owner's decision, kept as is. Known risk: a thief who knows the victim's PIN can cancel recovery attempts within the window, retaining access to the stolen wallet. Mitigations (without changing the rule): public guardians (public signatures, a 3-of-5 threshold) and a restart by the user's decision.

---

## 12. Network and client

### 12.1 The client and the core ✅

- **Flutter & Dart** — the interface and platform APIs; Android only, phones only.
- **Kotlin** — Keystore, biometrics, signing, the foreground service (the platform layer, it is unavoidable).
- **The core in Go** — the network stack, the verification of transactions and attestation chains, PoW verification, checkpoints, storage (SQLite). The core verifies signatures but does not own keys.
  - Why Go: a mature `go-libp2p` (QUIC/TCP transport, Noise, a Kademlia DHT, GossipSub, hole punching, circuit relay); the standard library and `x/crypto` have ECDSA P-256, X.509 parsing, and Argon2id.
  - Integration: `gomobile bind` is recommended (an AAR; the Kotlin service starts the core, Flutter communicates via platform channels, exchanging serialized messages, for example protobuf); the alternative is `c-shared` and `dart:ffi`.
  - The fallback: the core in Kotlin (`jvm-libp2p`) — one native language instead of two, but the networking part is weaker.
- The same core code runs as a headless simulator (Section 6.5).
- ❓ **The wallet signature.** The hardware Keystore reliably supports ECDSA P-256; Ed25519 — not everywhere; most likely a P-256 signature. Confirm on the target devices.
- ❓ **Resources.** The build size, memory, and energy (the Go runtime, the DHT, persistent connections). For the light mode — the DHT in client mode; measurements on the prototype.
- ❓ **Android attestation.** The attestation extension is parsed via `encoding/asn1`; a ready-made library for our needs may not exist.

### 12.2 Node tiers

| Tier   | What it does                                                              | When                                  |
| ------ | ------------------------------------------------------------------------- | -------------------------------------- |
| Full   | Stores and verifies the registry, relays, participates in voting           | Charging + Wi-Fi, a foreground service |
| Light  | Stores its own data and checkpoint headers, verifies Merkle proofs         | The phone's ordinary mode               |

A phone behind mobile NAT (CGNAT) does not accept incoming connections, which is why hole punching and relaying through volunteer full nodes are needed (in the spirit of libp2p circuit relay). These are not the project's servers, but not "pure P2P" either: we state this honestly.

### 12.3 Bootstrap ✅

- The first-peer search cascade: 1) a cache of past connections (SQLite); 2) Telegram/WhatsApp channels with a fresh signed seed list; 3) an address list embedded in the APK (an outdated fallback); 4) NSD/mDNS in the current Wi-Fi network; 5) a Bluetooth scan; 6) a QR "add a node" from a person nearby; 7) manual entry.
- The seed lists are signed with an N-of-M project multisignature; the channels are best-effort (WhatsApp has practically no programmatic access, Telegram works through third-party infrastructure), not a mandatory source. Peer exchange and list transfer through neighboring nodes have been added.
- Eclipse protection: a node mixes sources and does not trust a single peer until confirmation from N unrelated peers.
- After the first contact — a DHT (Kademlia) + gossip.
- Network instances: the nodes tell peers apart by `network_id`. In `invite_only` networks, the first node addresses and the network key are passed in the invitation (a QR or a link); the channels and the embedded list are not used.
- "No servers" = no central service infrastructure; a static seed list in the APK is data, not a server.
- The fragile period: the cold start of each network (the first ~10 nodes) — the seed circle's first nodes are constantly online.

### 12.4 Bluetooth and "offline" ✅

- Offline payments are rejected: Bluetooth is only a channel for the direct delivery of a signed transaction to the recipient, who publishes it to the network when internet appears.
- Until the transaction is published and confirmed by the network, it **is not a payment**: the sender can give the same sum to two recipients. Therefore: the interface status is "received via Bluetooth — awaiting publication"; the recipient must wait for network confirmation; on publication, the node checks `seq`/`prev` to quickly detect equivocation.

### 12.5 Storage ✅

- Drift/SQLite for the chains and indexes.
- Snapshots = state commitments: the root of the Merkle state tree in a checkpoint signed with a threshold signature (in the transitional period — by the seed circle).
- A light node stores its own chain, its incoming transfers, and checkpoint headers; a balance proof is obtained on request.
- The storage volume is not a constraint at the start: the target devices have gigabytes, the database is megabytes. A guideline of ~250 bytes per transaction; at 1 million wallets, the issuances alone give on the order of 3 GB per year. Revisit the history-pruning policy when approaching such a scale (checkpoints allow doing this without a protocol change).

---

