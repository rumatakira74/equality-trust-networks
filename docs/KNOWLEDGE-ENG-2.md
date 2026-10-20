# Trust Networks Platform — Part 2. Profiles and modules

> Unofficial English translation; in case of any discrepancy, the Russian original prevails.

[← Contents](KNOWLEDGE-ENG.md) · Part 2 of 5 · Sections 4–6

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

