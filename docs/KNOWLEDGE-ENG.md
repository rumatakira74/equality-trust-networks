---
name: "equality-trust-networks"
description: "Trust Networks Platform — a constructor of isolated networks without servers or an administrator, running on participants' phones: one person ≈ one wallet (vouching plus hardware attestation), equal periodic issuance, publicly verifiable history; eight application scenarios"
---

# Trust Networks Platform

> **Unofficial English translation.** In case of any discrepancy, the Russian
> original (`KNOWLEDGE.md`) prevails.

Document version: 9.11. Working title — Equality Project.  
Document date: November 1, 2026.  
Author: Kirill Lapchinsky  
Contacts: rumatakira74@gmail.com, Telegram @rumatakira  
© 2026 Kirill Lapchinsky. Terms of use are defined by the license (Section 16).

**DOI:** [10.5281/zenodo.23474316](https://doi.org/10.5281/zenodo.23474316)

Documentation: [equality-user-docs](https://github.com/rumatakira74/equality-user-docs) — for participants; [equality-programming-docs](https://github.com/rumatakira74/equality-programming-docs) — for developers.

The document has been reworked: rejected decisions were moved to a short appendix at the end.

Version 9.2: added Section 7.7 — results of the farming-economy simulation ([equality-anti-farm-simulation](https://github.com/rumatakira74/equality-anti-farm-simulation)); K = 3, slots 3/5/10, R = 10%, and a seed circle ≥ 300 confirmed quantitatively.

Version 9.3: working title — Equality Project; v1.0.0 published on Zenodo (DOI [10.5281/zenodo.23474358](https://zenodo.org/records/23474358)).

Version 9.8 (owner's decisions, October 21, 2026): guardian re-election (Section 11) — the new `update_guardians` transaction type: personal guardians can be re-elected after registration (the new guardians' consent is a signature of the change hash, passed manually, like a migration approve), a 48-hour effective window, cancellation by explicit PIN entry only, a limit of 150 effective changes per wallet over its lifetime (the former D4 restriction "appointed forever" is repealed: a network may be started without a seed circle, and public guardians are unattainable during the first year); `migrate` also transfers the **balance** (a formalization of the stage-3 implementation); "known v1 limitations" — migration cancellation remains the old key's right within the 14-day window (the theft-with-known-PIN risk is documented honestly, not fixed). The English translation of versions 9.4–9.7 is pending; the Russian original prevails.

Version 9.9 (the owner's decisions of October 22, 2026, following the block-3 simulation): the adaptive-fee target (10.2) is **15% per window** (was 6%); the mechanics are unchanged (K = 6, a 0.05 pp step, the bounds [0.5; 2]%, fee_min = 0.01, burn); the demurrage floor of profile template 5 is **30** (was 3000; 3 monthly issuances), profile 7 is unchanged (3000); the 14-day moratorium is confirmed unchanged.

Version 9.10 (the owner's decisions of October 27, 2026): profile 8 "Subscription resource" — `recurring` units (Section 4: each unit carries a period mark `period`; transferring a unit of period P is rejected after the boundary of period P+1; not a balance reset — the history stays verifiable; default `recurring = off`); network identification: a self-certifying `network_id = H(manifest ‖ genesis)` plus a human-readable `slug` (a label, not identity); the reserved constant `network_id = "global"` for the global network (there is no empty identifier — cross-network replay protection works in the global network too); a new transfer mode `peer-redeem` (member → member and member → provider) for the flow of recurring quotas between members; genesis validation of combinations: `recurring = on` only with a non-global `network_id`, `unit = quota(resource_id)`, and `transfer_mode ∈ {restricted, peer-redeem}`; APK file names: `equality-{global|slug}-{std|rec}-v{version}.apk` (ergonomics and license transparency, not identity); `recurring = on` is the technical marker of commercial use (Section 16.2).

Version 9.11 (the owner's decisions of November 1, 2026, mini-stages 12.2–12.3): the manifest consists of public keys — the seed circle (3.3) and providers (6.2) are defined in the manifest and genesis by fingerprints (public keys; fingerprint = SHA-256(attestedPublicKey ‖ protocolId), Section 7.2), not addresses; addresses are not stored in the manifest and are derived from the final network_id: `address = SHA-256(fingerprint ‖ network_id)` — a single derivation point (Section 7.2), so the self-certification `network_id = H(manifest ‖ genesis)` is not circular; wire-genesis: SeedMember has a single public_key field (the address field is removed; v9.10 genesis binaries are incompatible); a provider need not be a seed-circle member; at network assembly duplicate fingerprints and empty keys are rejected; the configurator: applicationId = `io.equality.<sanitized-slug>.x<hash8>`, the global network — `io.equality.global.x<hash8>` (hash8 is the first 8 hex of SHA-256(network_id), as in the `Eq-<hash8>` label; the `x` prefix is Android's requirement on the first character of a package segment; a configurator artifact, not a protocol one).

## Table of contents

The specification is split into five parts at numbered-section boundaries; section numbering is continuous. This page is the title, the version history, and the table of contents.

- **[Part 1. Foundations and architecture](KNOWLEDGE-ENG-1.md)** — Sections 0–3: how to read this document; product; analogues and differences; platform architecture.
- **[Part 2. Profiles and modules](KNOWLEDGE-ENG-2.md)** — Sections 4–6: profile parameters; profiles for the eight scenarios; modules.
- **[Part 3. Identity and protection](KNOWLEDGE-ENG-3.md)** — Sections 7–8: identity, vouching, farm protection; keys and owner protection.
- **[Part 4. Registry, units, and network](KNOWLEDGE-ENG-4.md)** — Sections 9–12: the registry; units, issuance, stock; wallet recovery; network and client.
- **[Part 5. Governance, model, and roadmap](KNOWLEDGE-ENG-5.md)** — Sections 13–19 and Appendix A: network governance; threat model; applicability bounds; commercial model; parameters; open questions; roadmap; rejected decisions.
