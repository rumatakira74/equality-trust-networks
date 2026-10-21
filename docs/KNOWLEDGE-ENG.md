---
name: "equality-trust-networks"
description: "Trust Networks Platform — a constructor of isolated networks without servers or an administrator, running on participants' phones: one person ≈ one wallet (vouching plus hardware attestation), equal periodic issuance, publicly verifiable history; seven application scenarios"
---

# Trust Networks Platform

> **Unofficial English translation.** In case of any discrepancy, the Russian
> original (`KNOWLEDGE.md`) prevails.

Document version: 9.8. Working title — Equality Project.  
Document date: October 21, 2026.  
Author: Kirill Lapchinsky  
Contacts: rumatakira74@gmail.com, Telegram @rumatakira  
© 2026 Kirill Lapchinsky. Terms of use are defined by the license (Section 16).

**DOI:** [10.5281/zenodo.23474357](https://doi.org/10.5281/zenodo.23474357)

The document has been reworked: rejected decisions were moved to a short appendix at the end.

Version 9.2: added Section 7.7 — results of the farming-economy simulation ([equality-anti-farm-simulation](https://github.com/rumatakira74/equality-anti-farm-simulation)); K = 3, slots 3/5/10, R = 10%, and a seed circle ≥ 300 confirmed quantitatively.

Version 9.3: working title — Equality Project; v1.0.0 published on Zenodo (DOI [10.5281/zenodo.23474358](https://zenodo.org/records/23474358)).

Version 9.8 (owner's decisions, October 21, 2026): guardian re-election (Section 11) — the new `update_guardians` transaction type: personal guardians can be re-elected after registration (the new guardians' consent is a signature of the change hash, passed manually, like a migration approve), a 48-hour effective window, cancellation by explicit PIN entry only, a limit of 15 effective changes per wallet over its lifetime (the former D4 restriction "appointed forever" is repealed: a network may be started without a seed circle, and public guardians are unattainable during the first year); `migrate` also transfers the **balance** (a formalization of the stage-3 implementation); "known v1 limitations" — migration cancellation remains the old key's right within the 14-day window (the theft-with-known-PIN risk is documented honestly, not fixed). The English translation of versions 9.4–9.7 is pending; the Russian original prevails.

## Table of contents

The specification is split into five parts at numbered-section boundaries; section numbering is continuous. This page is the title, the version history, and the table of contents.

- **[Part 1. Foundations and architecture](KNOWLEDGE-ENG-1.md)** — Sections 0–3: how to read this document; product; analogues and differences; platform architecture.
- **[Part 2. Profiles and modules](KNOWLEDGE-ENG-2.md)** — Sections 4–6: profile parameters; profiles for the seven scenarios; modules.
- **[Part 3. Identity and protection](KNOWLEDGE-ENG-3.md)** — Sections 7–8: identity, vouching, farm protection; keys and owner protection.
- **[Part 4. Registry, units, and network](KNOWLEDGE-ENG-4.md)** — Sections 9–12: the registry; units, issuance, stock; wallet recovery; network and client.
- **[Part 5. Governance, model, and roadmap](KNOWLEDGE-ENG-5.md)** — Sections 13–19 and Appendix A: network governance; threat model; applicability bounds; commercial model; parameters; open questions; roadmap; rejected decisions.
