# Trust Networks Platform — Part 1. Foundations and architecture

> Unofficial English translation; in case of any discrepancy, the Russian original prevails.

[← Contents](KNOWLEDGE-ENG.md) · Part 1 of 5 · Sections 0–3

---

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

