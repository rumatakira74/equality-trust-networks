---
name: "trust-networks-platform"
description: "Trust Networks Platform — a constructor of isolated networks without servers or an administrator, running on participants' phones: one person ≈ one wallet (vouching plus hardware attestation), equal periodic issuance, publicly verifiable history; seven application scenarios"
---

# Trust Networks Platform

Russian original: [README.md](README.md).

> **Unofficial English translation.** In case of any discrepancy, the Russian
> original ([README.md](README.md)) prevails.

Document version: 9.3. Working title — Equality Project.  
Document date: October 18, 2026.  
Author: Kirill Lapchinsky  
Contacts: rumatakira74@gmail.com, Telegram @rumatakira  
© 2026 Kirill Lapchinsky. Terms of use are defined by the license (Section 16).

**DOI:** [10.5281/zenodo.23351012](https://doi.org/10.5281/zenodo.23351012)

The document has been reworked: rejected decisions were moved to a short appendix at the end.

Version 9.2: added Section 7.7 — results of the farming-economy simulation ([equality-simulation](https://github.com/rumatakira74/equality-simulation)); K = 3, slots 3/5/10, R = 10%, and a seed circle ≥ 300 confirmed quantitatively.

Version 9.3: working title — Equality Project; v1.0.0 published on Zenodo (DOI [10.5281/zenodo.23458917](https://zenodo.org/records/23458917)).

## 0. How to read this document

| Mark | Meaning                          |
| ---- | -------------------------------- |
| ✅    | Decision confirmed by the user    |
| 🆕    | Proposal, awaiting confirmation  |
| ❓    | Open question                    |
| ⏳    | Deliberately deferred            |

Numeric parameters are draft values and subject to simulation.

---

## 1. Product

### 1.1 What it is

A platform where a community (a school, a cooperative, a volunteer group, a diaspora, a foundation, an event organizer) launches **its own isolated trust network** on participants' phones: no servers, no administrator, and no operating costs.

Each network provides three things:

1. **One person ≈ one wallet** — vouching by existing members plus hardware attestation of the device.
2. **Equal periodic issuance** — the network's "unit" is issued to all active members equally.
3. **Publicly verifiable history** — a member's reputation: age, the vouching graph, contributions.

What the "unit" is (money, hours, a quota, points, a voucher) and what can be done with it is defined by the **network profile**.

### 1.2 This is not a cryptocurrency ✅

- The platform does not issue or sell tokens, does not create an exchange market, and does not reward mining. Units exist only inside a specific network and can be non-transferable.
- The value is in the application (the seven scenarios below), not in the unit's price.
- The words "wallet", "checkpoint", `mint` are the technical vocabulary of a log of signed records, not a sign of a financial instrument.

### 1.3 Principles ✅

- **No administrator and no central servers.** The only exception is the seed circle at network launch; it self-liquidates by a rule from genesis (Section 13).
- **Zero operating costs.** A network is participants' phones; simulations and sandboxes are free.
- **Maximum decentralization** where it is achievable; temporary crutches are documented and time-limited.
- **A public registry within the network; anonymity is not a goal.**
- **Android smartphones with a hardware keystore only** (by default). Anyone without a suitable smartphone is outside the system; workarounds are not designed.
- **Honesty instead of promises:** each network publishes a manifest from which the guarantee level is computed (Section 3.3).
- **The source code is open, the license is non-commercial only** ✅: commercial use and commercial derivative works — under a purchased license (Section 16).

### 1.4 Seven scenarios ✅

1. Education and an institutions laboratory (including simulations).
2. Proof of personhood as a service for other applications.
3. Equal quotas for a shared resource.
4. Volunteer and street programs.
5. An internal currency or a time bank (equal monthly issuance and vouching instead of an administrator keeping accounts).
6. Volunteer contribution accounting.
7. A mutual aid fund.

---

## 2. Analogues and differences 🆕

State as of October 2026 from open sources (the list is at the end of the section); verify before using in materials.

### 2.1 Closest analogues

**Web-of-trust currencies: Circles.**

- Each participant, based on trust, receives a freely issued personal currency; protection from fakes is built on a web of trust.
- Circles 2.0 launched on May 21, 2025; issuance is one unit per hour; tokens are implemented as ERC-1155.
- The protocol runs on smart contracts of a public chain (Gnosis) and needs external web services (for example, a transaction-fee relayer). The developers acknowledged that changing trust limits after launch is hard.
- The Berlin pilot has been running since October 2021.

**Proof of personhood (PoP).**

- World — iris scanning with a special device; BrightID — a social graph; Idena — regular synchronous ceremonies with puzzles; Proof of Humanity — video verification.
- In January 2026 the Passport aggregator removed the BrightID and Idena stamps from the score calculation. This does not prove an absence of demand, but it shows the instability of the integrations market.
- The overall conclusion of reviews: BrightID is the most private but the hardest to bootstrap.

**Software for local currencies and time banks.**

- Cyclos, Community Forge (Drupal), Community Weaver, hOurworld, Community Exchange System.
- All centralized: data in an ordinary database on hosting, an administrator required. The authors remind that software does not replace community governance and support. A known scaling barrier is exchange between separate groups.

### 2.2 How the platform differs

- **No blockchain and no servers:** the network lives on phones. Circles depends on a chain and a relayer; local-currency software on hosting and an administrator.
- **Hardware attestation of the device** as an additional anti-sybil layer (together with vouching). This layer is not visible in the reviewed analogues.
- **A platform of profiles** for seven scenarios instead of one product.
- **Owner protection from coercion:** a duress PIN and delayed large transfers.
- **Networks without an administrator** with immutable-by-default parameters, non-retroactive changes, and the right to fork.
- **Zero operating costs:** free simulations and sandboxes as the entry to the product.

### 2.3 Risks and conclusions

- Circles is close in idea and has been working for years. The difference must be stated explicitly ("without a chain and without servers, on phones, with profiles"), otherwise the project will be considered a repeat.
- Social trust graphs are hard to bootstrap (a cold start). Run pilots in already existing communities.
- The "proof of personhood as a service" market is unstable so far; do not build the whole business model on it.
- A platform with profiles is also a dilution risk: each profile must have at least one real pilot.

### 2.4 Sources

- https://hackernoon.com/circles-a-global-experiment-to-reinvent-money
- https://arxiv.org/html/2504.02714v1
- https://hackmd.io/3RaPT3vqSsayKUuqtuIRvw
- https://passport.human.tech/blog/proof-of-personhood-explained-how-it-works-who-s-building-it-and-why-it-matters-now
- https://support.passport.human.tech/stamps/how-to-verify/brightid-stamp
- https://support.passport.human.tech/stamps/how-to-verify/idena-stamp
- https://www.drupal.org/project/mutual_credit
- https://wiki.p2pfoundation.net/Cyclos
- https://blog.p2pfoundation.net/an-update-on-the-interoperability-of-complementary-currency-software-systems/

---

## 3. Platform architecture

### 3.1 Network instances ✅

- Not one network but many isolated ones. Each has its own `network_id`, genesis, seed circle, and profile. There is no exchange between networks (bridges are outside v1).
- All scenarios rely on the same three things (Section 1.1). The difference is in **what counts as the unit, where it comes from, whether it can be transferred, and what is done with it**. That is exactly what is moved into the profile.
- A network is created by a constructor: an organization picks a profile template, configures parameters, and receives a genesis package and an application.

### 3.2 The three layers

| Layer       | What is included                                                                                                                                                                 | How it changes                                                                                                                                           |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Core**    | Wallet chains, signatures, attestation (per the network policy), statuses, `vouch`, `evidence` and the `flagged` mark, equivocation, checkpoints, epochs, the public registry, verification by every node | Only by protocol version (Section 13), the same for all networks                                                                                          |
| **Profile** | Network parameters: the unit, issuance, transferability, stock, entry, device policy, etc. (Section 4)                                                                            | Set in genesis; immutable by default, with `params_mutable = true` — only within the genesis bounds by member signaling (Section 13)                      |
| **Modules** | Typed operations: attestation, redemption, the pool, contributions, the sandbox (Section 6)                                                                                    | Enabled by the profile                                                                                                                                    |

**Core invariants** (the profile cannot cancel them):

- signature verification and the double-spend ban;
- wallet activation only through `vouch` (the seed circle is defined in genesis and time-limited);
- the registry is public within the network;
- the manifest and the guarantee-level mark, the manifest hash is written into genesis;
- delayed large transfers (Section 8.3) are mandatory in all networks with transferable units, except the sandbox;
- parameter changes are non-retroactive: they do not revoke `active` status, do not change past balances and history, and apply only to future operations;
- the right to fork: any member can continue the network under the old rules with the same history.

### 3.3 The network manifest and the guarantee level ✅

- The **manifest** is public, its hash is written into genesis, and after launch it is immutable (except via the procedure in Section 13): `network_id`, all profile parameters, `sandbox`, `device_policy`, `checkpoint_mode`, the seed circle.
- The **guarantee level is computed from the manifest, not declared.** High — `strict` + `decentralized` + `open_vouch`; medium — `relaxed` or `members_quorum`; educational — `sandbox`.
- An attestation (Section 6) carries the manifest hash, so the consumer sees from which network and with which guarantees a "person" arrives. A school class must not pass itself off as global identity proof.
- The partner itself chooses which networks to trust: this continues the "reputation without verdicts" principle.

---

## 4. Profile parameters

| Parameter                              | Options                                                                                                                                                                                       | Default ("Currency")         |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `unit`                                | `currency`, `hours`, `quota(resource_id)`, `points`, `voucher(program_id)`, `heartbeat` (an activity counter without value)                                                                     | `currency`                     |
| `transfer_mode`                       | `free`; `restricted` (only member → resource provider, or per a list from the network parameters); `soulbound` (non-transferable)                                                               | `free`                         |
| `issuance_mode`                       | `periodic_equal`; `quota_reset` (a quota per period, unused amounts burn); `on_contribution`; `pool_claim` (equal shares from a pool); `heartbeat`; `none`                                       | `periodic_equal`               |
| `issuance_amount` / `issuance_period` | any                                                                                                                                                                                            | 1000 / month                   |
| `accrual_window`                      | 0…N months                                                                                                                                                                                     | 6 months (0 for `quota_reset`) |
| `fee_mode`                            | `burn`; `pool` (into the network's common fund); `none`                                                                                                                                         | `burn`, an adaptive rate       |
| `demurrage_rate` / `demurrage_floor`  | a fraction per month / a tax-free remainder                                                                                                                                                     | disabled                       |
| `entry_mode`                          | `open_vouch`; `founders_seed` (a seed circle from genesis, then vouching only); `invite_only_vouch` (entry by an invitation key from genesis, then vouching)                                     | `open_vouch`                   |
| `seed_sunset`                         | the number of `active` wallets or a term after which the seed circle loses its special rights                                                                                                    | ❓                              |
| K, slot pool S, vouching-member age   | numbers                                                                                                                                                                                        | 3; 3/5/10; 3 months            |
| `voucher_reward_R`                    | a fraction, only with `periodic_equal`                                                                                                                                                          | 10% of the first 12 issuances  |
| `device_policy`                       | `strict` (hardware attestation); `relaxed` (devices without it are allowed; only for networks with low stakes)                                                                                   | `strict`                       |
| `signing_auth`                        | `biometric_strong`; `device_credential` (the device's PIN/password)                                                                                                                              | `biometric_strong`             |
| `network_access`                      | `public`; `invite_only`                                                                                                                                                                        | `public`                       |
| `checkpoint_mode`                     | `decentralized`; `members_quorum` (the checkpoint is signed by ≥ 2/3 of the members)                                                                                                             | `decentralized`                |
| `sandbox` / `time_scale`              | `false`/`true`; `real`/`accelerated(n)`                                                                                                                                                         | `false`, `real`                |
| `params_mutable`                      | `false`; `true` (mutable parameters only within the genesis bounds)                                                                                                                              | `false`                        |
| `mutable_bounds`                      | for each mutable parameter `[min, max]`; protective parameters are not included here                                                                                                            | —                              |
| `modules`                             | the list of modules from Section 6                                                                                                                                                              | empty                          |

---

## 5. Profiles for the seven scenarios ✅

| №   | Profile                            | Unit / issuance                                                              | Transfer                                     | Stock                                                                        | Entry / K                                    | Devices / signing                                            | Modules                                                |
| --- | ---------------------------------- | ---------------------------------------------------------------------------- | -------------------------------------------- | ---------------------------------------------------------------------------- | --------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------- |
| 1   | Laboratory (education)             | `points` or a play `currency`; `periodic_equal`, accelerated epochs           | `free`                                       | Parameters are part of the study scenario, fixed at network creation          | `founders_seed`, K = 1–2                      | `relaxed`; `device_credential`                               | `sandbox` (+ `pool`, `contribution` for cases)          |
| 2   | Proof of personhood                | `heartbeat`: a monthly "pulse" with reinforced PoW, accumulates "tenure"      | `soulbound`                                  | none                                                                         | `open_vouch` or `invite_only_vouch`, K = 3–5  | `strict`; `biometric_strong`                                 | `attest`                                                |
| 3   | Equal resource quotas              | `quota(resource_id)`; `quota_reset`, burns at the end of the period          | `restricted` (member → provider)            | none                                                                         | `invite_only_vouch`, K = 3                    | `strict` (a double quota = sybil profit)                     | `redeem`                                                |
| 4   | Volunteer / street program         | `voucher(program_id)`; `pool_claim` — equal shares from the program budget    | `restricted` (member → partner point)        | none                                                                         | `invite_only_vouch`, K = 2–3                  | `strict`; members only with a smartphone                     | `pool`, `redeem`                                        |
| 5   | Internal currency / time bank      | `hours` or `currency`; `periodic_equal` (small amounts)                      | `free`                                       | Demurrage enabled (❓ rate), a minimal or `pool` fee                          | `open_vouch`, K = 2–3                         | `strict`; `relaxed` acceptable for closed communities       | `pool` (the common fund), optionally `contribution`     |
| 6   | Volunteer contribution accounting | `points`; `on_contribution`, paid from the program pool                       | `soulbound` or `restricted` → `redeem`       | none                                                                         | `invite_only_vouch` or `founders_seed`, K = 2–3 | `strict` or `relaxed` by the network's decision               | `contribution` (`peer`), `redeem`, `pool`                |
| 7   | Mutual aid fund                    | `currency`; `periodic_equal`                                                 | `free`                                       | Demurrage + a burn fee                                                       | `open_vouch`, K = 3                           | `strict`; `biometric_strong`                                 | `pool` (collections), `attest` (presenting "tenure" to foundations) |

**Notes on the profiles**

- **Profile 7 and external foundations.** A balance does not grant a right to a foundation payout: the unit is a vehicle of trust and a measure of distribution, not a saving. The foundation itself reads the public registry and announces its own criteria; the protocol knows nothing about foundations and contains no grant marks or special payouts (otherwise the registry turns into a public list of a vulnerable group). Collections should be judged by the history and the distribution of incoming transfers (many transfers from old unrelated wallets ≠ a few from a fresh ring), not by the total. The public registry is a showcase of solidarity: a donor sees that help reaches a living person.
- **Profile 7 and redistribution.** Value is in the flow, not in accumulation. The fee burns on movement and does not penalize holding; holding is penalized by demurrage, which is why it is enabled by default in profiles 5 and 7 (Section 10.3).
- **Profile 4.** Members can only be owners of suitable smartphones (Section 15).
- **Profile 2.** The "pulse" is monthly proof that the wallet is alive (reinforced PoW, Section 7.6); the units are not transferred and have no value.

---

## 6. Modules (typed operations, not contracts) ✅

Operations are fixed by the protocol and verified by every node. There is no arbitrary code: the platform stays without smart contracts.

### 6.1 `attest` — attestation (proof of personhood)

- The wallet signs a statement: `network_id`, address, status, age, vouching-graph depth, the `flagged` mark, the manifest hash, the checkpoint epoch, the verifier's `challenge`, an expiration date. The signature follows the network's `signing_auth`.
- A Merkle proof of the wallet's state from a checkpoint is attached to the statement. The verifier does not need to be a node: it verifies the wallet signature, the proof, and the checkpoint signature.
- The verifier library (Go; JVM and JS builds — ❓) is part of the product; it needs a lightweight network client that syncs with any node; the project's servers are not needed.
- **The verifier decides** what counts as sufficient (the age threshold, depth, guarantee level): the protocol issues no verdicts.
- "One person — one action" in a third-party application is ensured by the identifier `H(address ‖ app_id)`. It is publicly linked to the address, which is why **secret voting is not supported in v1** (❓ ZK membership proofs — later).
- Reuse protection: the `challenge` and the expiration date.

### 6.2 `redeem` — redemption

- A transfer of a unit to a provider wallet with a `resource_id` and an optional time slot. The provider publishes a signed `receipt`.
- The provider is an ordinary wallet whose address is listed in the network parameters; it has no special governance rights. For physical access it must wait for network confirmation (Section 12.4).
- A quota (`quota_reset`) does not accumulate and burns at the end of the period.

### 6.3 `pool` — the pool (collections, a program budget, a common fund)

- A typed account with fixed rules: `pool_create` {recipient, purpose, deadline, release mode `release_on_deadline` | `release_on_goal` | `release_continuous`, refund `refund_if_unmet` | `none`}, `pool_donate`, `pool_close`, `pool_claim` (equal shares to `active` wallets).
- Progress is public. Judging collections by the history of incoming transfers stays with external readers of the registry.

### 6.4 `contribution` — contribution accounting

- `contribution_claim` {description hash, size (hours/points), period}; `contribution_confirm` from confirmers.
- Confirmation is `peer` only: N independent confirmations from different `active` wallets not linked to the claimant by vouching, with a per-person confirmation limit per period. There is no coordinator role.
- Payout: **from the program pool** (recommended, no inflation) or limited `on_contribution` issuance with a cap per period and per person.
- The collusion risk ("we confirm each other") is mitigated by limits, the cap, the graph, and publicity, but is not fully eliminated.

### 6.5 `sandbox` — the sandbox (an institutions laboratory, simulations)

- A network mode: accelerated epochs (`time_scale`); parameters are fixed in genesis and do not change. A "reset" and "another experiment" mean launching a new network, and it costs nothing.
- **Synthetic wallets** (the `synthetic` mark) can be launched by any member on their own device or computer per a scenario. They have no special rights: they pass `register` and vouching like everyone else.
- **Zero cost.** The simulator is a headless mode of the same core (Go): hundreds and thousands of virtual nodes on one computer, reproducible scenarios (an RNG seed), accelerated time. For a live classroom demo, the network runs over local Wi-Fi (NSD/mDNS discovery, Section 12.3) without internet or servers.
- The registry is exported for analysis: turnover, supply, and inequality are computed from the public registry by any client.
- A ban on the link to external value: no funds and no attestations for external consumers, the `sandbox` mark in the manifest.

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
- The payout stops from the epoch of finalization of an `evidence` against the vouched-for member. This is the only "punishment": the vouching member loses the reward, not money.
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

The model: [equality-simulation](https://github.com/rumatakira74/equality-simulation) (Python; averaged over 8 runs, a 60-month horizon). Modeled: the growth of an honest network and of a farm: slots, age thresholds, K, the R reward, the cost of devices, and purchased vouches. Signatures, the network, the registry, PoW, the moratorium, demurrage, and the fee **were not modeled** — the farm economy in the model is overstated rather than understated.

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

The threat: the owner is forced to open the app or send money.

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
- **Accrual for missed months is limited by a window** (`accrual_window`, 6 months by default): unlimited accrual creates "sleeping" accruals and an incentive for a delayed attack. The money is not blocked: beyond the window, the accrual simply burns.
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
- **Migration:** a 14-day delay, during which the old key (if alive) can cancel it; the new wallet passes the same attestation as at `register`; age, issuance history, `period`, and vouches **are transferred** (otherwise migration bypasses the moratorium and the limits); the old fingerprint is blacklisted forever (otherwise the "migrate → register again" cycle yields unlimited units); a guardian cannot be a guardian for more than K wallets.
- **If there are no guardians — two paths ✅:**
  - **Public guardians:** an `active` wallet aged ≥ 12 months declares itself a guardian (`guardian_offer`). The user chooses them in advance; with public guardians participating, the threshold is 3 of 5 (any mix of personal and public ones). The reward is a fixed sum from new emission for each successful `migrate` of a ward (the size ❓, a guideline of no more than 10% of monthly issuance; no more often than once per 12 months per ward; a limit of G_max = 20 wards per guardian; paid only to the signers and only after the delay without a cancellation). Applicable in networks with emission.
    - Risk: a reward from emission is a subsidy to farms "migrating" their own wallets; the frequency limit, the delay, and the migration PoW must make this unprofitable; verify the size by simulation.
    - Risk: collusion of public guardians against an owner who has lost the device. Protection: the 3-of-5 threshold, the 14-day delay, public signatures. ❓ Whether to require at least one personal guardian.
  - **A restart by the user's decision:** an ordinary `register` with a new key and new K vouches. The old wallet remains in the registry untouched and inaccessible; the funds on it are lost. A restart gives nothing beyond registering a new wallet.

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

## 13. Network governance: no administrator ✅

- **There are no governance roles:** no operator, no instructor, no coordinator, no charter keys. A resource provider or a pool recipient is an ordinary wallet listed in the network parameters; they cannot change the network.
- **The seed circle** (`founders_seed`) is defined in genesis and limited by time or by the number of `active` wallets (`seed_sunset`): after the threshold, the founders lose their special rights as vouching members and checkpoint signers. This is the only exception to "no administrator", and it self-liquidates by a rule from genesis.
- **Parameters are immutable by default** (`params_mutable = false`). If the rules have stopped fitting, a new network is launched (it is free), and the members move voluntarily.
- If a network enables `params_mutable = true`:
  - for every mutable parameter, genesis sets the bounds `[min, max]` (for example, the demurrage rate from 0 to 3%);
  - **the protective parameters never change:** K and the vouching rules, the delay D = 48 h, `transfer_mode`, the emission size and mode, `device_policy`, `signing_auth`, `checkpoint_mode`, `params_mutable`;
  - the procedure: a proposal by any `active` wallet → a public signing window (❓ the duration) → a threshold of ≥ 2/3 of the `active` wallets (❓) → a public activation delay of at least 30 days → activation at a designated epoch.
- **Non-retroactivity** (a core invariant): the majority can neither "switch off" a minority nor confiscate its balance.
- **The right to fork** (an invariant): the registry is public, so any member can continue the network under the old rules with the same history up to the activation epoch. A fork package (a new genesis from the checkpoint of epoch E) is assembled with one app command (❓ the implementation). The delay of at least 30 days is needed so that everyone has time to see the decision and choose.
- **In the sandbox**, `params_mutable` is always `false`.
- **The residual risk:** a colluding majority can push the mutable parameters toward the tolerance bounds and cause a community split, but cannot take away funds and status.
- **Protocol and software versions.** Every transaction and checkpoint has a `protocolVersion` field. The core rules change by protocol version: a proposal → signaling by the nodes (a weighted vote threshold) → activation at a designated epoch; the nodes with the old version disconnect from the new chain at activation. The APK release is signed with an N-of-M multisignature; reproducible builds and published hashes.
- ⏳ The project keys (the signing of releases and seed lists): who holds them and how in the long term, the transfer to the community.
- ❓ An emergency stop on a critical bug: in networks without an administrator, a kill-switch contradicts the principle; an explicitly documented procedure (a multisignature, a time limit) or a refusal is needed.

---

## 14. Threat model

| Threat                               | Vector                                                       | Main defense                                                                                                                                                     | Residual risk                                                                                  |
| ------------------------------------ | ------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Sybil / a wallet farm                | Many devices, emulators, cloned profiles                    | Vouching, hardware attestation, memory-hard PoW, the moratorium, the accrual window, the fee                                                                           | A farm of real devices with mutual vouches at a high external unit value                       |
| Vouching-member collusion            | A ring of real devices vouches for each other               | The slot pool, the age threshold, the moratorium, graph analysis                                                                                                       | Slow but possible ring growth                                                                  |
| A double spend                       | Two transfers from one state                                 | Chain numbering, sampled voting, the recipient waiting for finality                                                                                                    | The window until finalization                                                                  |
| A double issuance                    | Re-claiming a period                                         | Verification by every node via `period` in the chain                                                                                                                   | —                                                                                              |
| An eclipse attack                    | Isolating a node among controlled peers                      | Mixing the sources, confirmation from N unrelated peers                                                                                                                 | A new node with a single source                                                                |
| Time shifting                        | Manipulating "network time"                                   | Epochs by finalized checkpoints                                                                                                                                        | A small network at the start                                                                    |
| Phone theft                          | Using an unlocked device                                      | Signing only after per-operation authentication (per `signing_auth`)                                                                                                  | Owner coercion                                                                                  |
| Owner coercion                       | The owner is forced to open the app or send money            | The duress PIN with decoy mode, delayed large transfers                                                                                                                | A prepared attacker checking the registry                                                      |
| Device loss                          | The key is non-exportable                                     | Social recovery with a delay, public guardians, a restart                                                                                                             | Guardian collusion                                                                             |
| Project key compromise               | The signing of seed lists and releases                       | An N-of-M multisignature, key rotation                                                                                                                                  | Social engineering                                                                             |
| Spam                                 | Mass transactions                                            | A light PoW + a burn fee                                                                                                                                                | —                                                                                              |
| Contribution-confirmation collusion | Members mutually confirm non-existent contributions          | Confirmation limits, a cap per period, a payout from a pool, the vouching graph                                                                                        | Collusion in small groups                                                                       |
| Attestation replay                   | Passing off someone else's attestation as one's own          | The verifier's `challenge`, the expiration, the signature per `signing_auth`                                                                                            | A voluntary handover of the device                                                              |
| Parameter capture by a majority      | A colluding majority changes the rules                       | Parameters immutable by default; mutable only within the genesis bounds; the protective ones immutable; non-retroactivity; a delay ≥ 30 days; the right to fork        | A community split, but no loss of funds                                                        |

---

## 15. Applicability bounds and risks ✅

- **No smartphone — outside the system.** The protocol requires a hardware-secured Android phone. Workarounds (curators, paper receipts) are not designed. The consequence: profiles 3, 4, and 6 are available only to owners of suitable phones; the product is positioned accordingly.
- **The proof of personhood is soft.** It does not replace KYC and biometrics; it is not recommended where the stakes are high.
- **Secret voting** is impossible in v1 (Section 6.1).
- **Minors** (education): personal data is not written to the registry; compliance with the requirements for the protection of children's data is the responsibility of the organization and the lawyer (❓); children need `device_credential`.
- **Small networks.** Sampled voting is unreliable for a network of dozens of people. They use `members_quorum`; if some of the phones are offline, finalization slows down; the network size boundary — ❓.
- **Zero costs do not cancel connectivity.** The relays and the nodes with a public address are volunteer. In a closed network where everyone is behind mobile NAT in different carriers, at least one node with a reachable address is needed (a home router, IPv6); in a classroom and at an event, local Wi-Fi suffices.
- **The cold start of the trust graph.** Every network needs a sufficient seed circle and a living community; pilots — only in existing communities.
- **Contribution confirmation** is the weakest point of scenario 6 (collusion).
- **Publicity.** The network registry is visible to its members; anonymity is not a goal; vulnerable groups must account for this before the launch.

---

## 16. Commercial model ✅

### 16.1 Licensing ✅

- The project is distributed as **open source under a non-commercial-use-only license**: the code can be read, run, modified, and the changes distributed **for non-commercial purposes only**.
- **Any commercial use and the creation of commercial derivative works require purchasing a commercial license.**
- The license covers all the platform's software: the core, the app, the network constructor, the verifier, the simulator.
- Such a license does not meet the OSI definition of "open source" (it restricts commercial use). In public materials, it is more accurate to write "the source code is open" (source-available), so as not to mislead.
- The platform has no operating costs (no servers and no hosting), so the only revenue source from the software is the commercial license. A network-usage fee is not introduced; the protocol contains no payments to the developer (otherwise an embedded rent and a question of trust).
- The network fork right (Section 13) and the software license are independent: forking a network on the same software is allowed within the license (non-commercial — freely, commercial — under the license).

### 16.2 Who pays and who does not 🆕

- **Free (non-commercial use):** education and research, personal use, simulations and sandboxes, volunteer and charity groups (the bounds — Section 16.3).
- **A paid license:**
  - commercial products embedding the verification of attestations;
  - commercial derivative works (an app under a client's brand, and the like);
  - the commercial operation of networks (for example, a paid network for clients).
- Separately from the license, services can be provided: training, pilot support, custom network setup (❓).

### 16.3 What to decide before publishing the code ❓

1. **The definition of "non-commercial use".** Border cases: schools and universities (including paid ones), organizations with paid staff, cooperatives and HOAs (membership fees), foundations that collect and distribute money (profile 7), paid courses about the platform.
2. **The basis of the license text:** a ready-made non-commercial license template (for example, PolyForm Noncommercial) or our own text; also consider a model with a transition to a free license after a set term (BSL).
3. **A contributor agreement (CLA):** without it, third-party contributions cannot be included in the commercial license.
4. **The licenses of dependencies** (for example, `go-libp2p`): check the compatibility with the chosen license.
5. **The verifier and the attestation SDK:** one license with the rest of the code, or a separate, freer one. If every third-party application must buy a license, this slows down the adoption of proof of personhood.
6. **The trademark and the name:** to be protected separately from the code license.
7. **The price and form of the commercial license:** one-time, a subscription, by the number of networks or members.
8. **A legal review** of the license and the bounds of "non-commercial" before the first publication.
9. **Enforcement.** Violations in a public network without an administrator cannot be technically stopped; the reliance is on legal measures, the trademark, reputation, updates, and support for the licensees. Technical restrictions (DRM, a license check in the app) contradict the platform's principles and are not planned.
10. **Patents.**
    - ✅ The user's decision: open code without a patent clause invites patent claims against the project and the licensees; in dual licensing, the patent license must be included only in the paid branch, otherwise the commercial license has nothing to sell.
    - **Whether there are own patents or applications** and whether they are planned. If there are none, there is nothing to sell in the commercial license except the right of commercial use; protection from third-party patent claims remains needed in any case.
    - **The scope of the patent license in the free branch.** A patent grant limited to non-commercial use is recommended: otherwise schools and volunteers remain under the threat of a suit from the licensor itself. Commercial use, including the patent license, only under the paid license. Apache-2.0 as a model does not fit: it gives a patent grant for any use, including commercial, and devalues the paid branch. The ready-made non-commercial templates (for example, PolyForm Noncommercial) contain patent provisions: check the text.
    - **Third-party patent claims.** Contributions are accepted with a patent grant (in the CLA) and with a termination of rights on a suit against the project. A search of existing patents on the key mechanisms (device attestation, vouching, delayed transfers) — with a patent attorney.
    - **Timing.** Publishing the code or the protocol description before filing can destroy the invention's patentability where absolute novelty is required; a grace period exists not everywhere and is limited. If patents are needed, file before the first publication (including before the publication of this document). If not, publish the description as defensive (prior art).

### 16.4 The legal complexity of the profiles

- The profiles with a transferable currency (5, 7) are noticeably more complex than the profiles with non-transferable or resource units (1, 2, 3, 6). Profile 7 additionally touches fundraising. A lawyer's consultation — before the pilots of 5 and 7 (❓).
- **The preliminary pilot order (❓):** 1 → 2 → 3 → 6 → 5 and 4 → 7 — from the least to the most regulated. Every profile needs at least one real pilot.

---

## 17. Parameters (draft values, subject to simulation)

| Parameter                                                            | Value                                                                                                       | Status                             |
| --------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| Base issuance                                                         | 1000 / month                                                                                                 | ✅                                  |
| Missed-issuance accrual window                                        | 6 months                                                                                                    | ✅                                  |
| `registered` lifetime without activation                              | 3 years                                                                                                     | ✅                                  |
| `transfer` moratorium                                                 | 7–30 days                                                                                                   | ✅ (a range)                        |
| Issuance-challenge window                                             | ±48 h                                                                                                       | ✅                                  |
| Timestamp tolerance                                                   | ±30 min                                                                                                     | ✅                                  |
| Migration delay                                                       | 14 days                                                                                                     | 🆕                                  |
| Personal guardians                                                    | 2 of 3                                                                                                      | ✅                                  |
| Public guardians: threshold / age / G_max                             | 3 of 5 / ≥ 12 months / 20                                                                                    | 🆕                                  |
| Public guardian reward                                                | ❓ (a guideline ≤ 10% of monthly issuance, no more often than once per 12 months per ward)                    | 🆕                                  |
| Vouches for activation K                                              | 3                                                                                                           | ✅ (simulation, Section 7.7)        |
| Slot pool S                                                           | 3 (3–12 months), 5 (above 12 months), 10 for the seed circle in the first year                               | ✅ (simulation, Section 7.7)        |
| Vouching-member age                                                   | ≥ 3 months                                                                                                  | 🆕                                  |
| Seed-circle size                                                       | ≥ 300 wallets                                                                                               | ✅ (simulation)                     |
| Voucher reward R                                                      | 10% of the vouched-for member's first 12 issuances                                                           | ✅ (in the model; farm-neutral, Section 7.7) |
| Delayed-transfer delay D                                              | 48 h, fixed                                                                                                 | ✅                                  |
| Delayed-transfer threshold P                                          | ❓ (the default and `P_max`)                                                                                 | 🆕                                  |
| Fee: `fee_min`, `[rate_min, rate_max]`, target supply growth          | ❓ (a `rate` guideline of 0.5–2%)                                                                            | ⏳                                  |
| Demurrage: rate / floor                                               | off by default; ❓ for profiles 5 and 7                                                                     | ⏳                                  |
| PoW for `transfer`                                                    | light                                                                                                       | ✅                                  |
| PoW for `register` and issuance                                       | memory-hard, reinforced                                                                                     | ✅                                  |
| Checkpoint-epoch duration                                             | ❓                                                                                                           | 🆕                                  |
| Parameter-change activation delay                                     | ≥ 30 days                                                                                                   | 🆕                                  |
| Parameter-change signature threshold / window                         | ≥ 2/3 `active` / ❓                                                                                          | 🆕                                  |

---

## 18. Open questions

1. ✅ Product name — **Equality Project** (confirmed October 18, 2026; a tag on the Zenodo v1.0.0 deposit). Brand — ❓.
2. Simulation: the farm economy is closed (Section 7.7 — K, S, R, the seed circle confirmed). Remaining: the registry (conflicts, equivocation, loss of connectivity), the fee, demurrage, the moratorium, and calibrating recruit/vouch_prob on pilots.
3. `seed_sunset`: the condition and term for shutting down the seed circle; the signer set in the transitional mode.
4. The default value of P and `P_max` (Section 8.3).
5. The consequences of a proven equivocation: confirm the soft scheme (Section 7.5).
6. Public guardians: the reward size, whether a personal guardian is mandatory (Section 11).
7. The Go core: the integration method, resources, the P-256 signature, attestation parsing (Section 12.1).
8. Parameter changes: the signature threshold and window, the fork-package implementation (Section 13).
9. Secret voting and ZK membership proofs (Section 6.1).
10. Minors in educational networks: data and legal requirements.
11. Contribution confirmation: resistance to collusion, the cap sizes.
12. Small networks: the size boundary for `members_quorum`, the behavior when some of the members are offline.
13. Legal questions on the profiles (especially 5 and 7), the distribution of the app (Google Play, and the like).
14. Licensing: the definition of "non-commercial", the text basis, the CLA, the patent strategy, the verifier license, the commercial-license price (Section 16.3).
15. The attestation verifier: JVM and JS builds.
16. The emergency stop in networks without an administrator (Section 13).
17. ⏳ Economics, the project keys.

---

## 19. Roadmap

1. **The simulator and the specification.** A headless core, models of the registry, vouching, the fee, demurrage, and farm income; it costs nothing and is needed for choosing all the parameters. ✅ The farm-economy model is ready and confirmed K = 3, slots 3/5/10, R = 10%, a seed circle ≥ 300 (Section 7.7); the rest — the registry, the fee, demurrage.
2. **Licensing.** The non-commercial and commercial license texts, the CLA, the trademark, the patent strategy, a legal review (Section 16.3). Before the first publication of the code and the protocol description.
3. **A core prototype (Go, with an Android build check).** Transactions (including `vouch`), signatures, attestation verification, statuses, the fee, the wallet chain, local storage, without networking.
4. **The network.** Peer discovery, DHT/gossip, relaying, the bootstrap cascade, `invite_only`; an eclipse test.
5. **A closed test network** on real Android devices: issuance, challenges, migration, Bluetooth transfer.
6. **An audit** of the cryptography, consensus, and attestation by an independent party.
7. **The platform.** Profiles and modules, the network configurator (the genesis package and the manifest), the attestation verifier, the templates of the seven profiles.
8. **Pilots by profile** in the order of Section 16.4: laboratory → proof of personhood → quotas → contributions → internal currency and programs → mutual aid fund. Every pilot is a separate network instance.

---

## Appendix A. Rejected decisions (briefly)

So as not to return to what has been discussed; the section can be deleted.

- **An IOTA-style DAG** instead of a chain per wallet — no total order, slow confirmations in a small network.
- **A decreasing emission** — handled by the fee and demurrage.
- **A vouching-member deposit** and **a reward to vouching members from new emission** — the vouching members are incentivized with a share of the vouched-for member's issuance.
- **An operator, a charter, an instructor, a coordinator** — all networks without an administrator.
- **Paper receipts and curators** for people without a smartphone — outside the system.
- **An SMS challenge, a video call to a "random user", a QR as presence proof** — vulnerable to sybils, require servers.
- **A reward for submitting violation evidence** — a farm would submit it against itself.
- **A parameter-change threshold of 90% and above** (a minority veto) and **the clustering of votes by graph** (fragile).
- **The core in Zig** — no libp2p, the language is before version 1.0.
- **The core in Kotlin** — kept as a fallback.
- **Anonymity and privacy as a goal** (Tor, stealth addresses, hiding the graph) — not the project's goal.
- **Mint without a challenge and without an accrual window**, unlimited accrual of missed issuances.

---

© 2026 Kirill Lapchinsky · rumatakira74@gmail.com · Telegram @rumatakira · October 18, 2026
