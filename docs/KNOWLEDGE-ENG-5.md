# Trust Networks Platform — Part 5. Governance, model, and roadmap

> Unofficial English translation; in case of any discrepancy, the Russian original prevails.

[← Contents](KNOWLEDGE-ENG.md) · Part 5 of 5 · Sections 13–19 and Appendix A

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
| Owner coercion                       | The owner is forced to open the app or send units            | The duress PIN with decoy mode, delayed large transfers                                                                                                                | A prepared attacker checking the registry                                                      |
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
| Demurrage: rate / floor                                               | 1% per month; floor: profile 7 — 3000, profile 5 — 30 (3 monthly issuances); mandatory for profiles 5 and 7                                     | ✅                                  |
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

---

## Theme files (service)

- [Naming and artifact-storage rules](naming-conventions.md) — the development plan is always `development-plan`; in the clouds, everything goes into the `mistral` folder, not the root.
