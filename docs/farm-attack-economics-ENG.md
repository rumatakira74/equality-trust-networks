# Farm-Attack Economics — Interpreting the Simulation Results

An addition to section 7.7 of KNOWLEDGE.md (v9.2). Basis: `equality-simulation` (https://github.com/rumatakira74/equality-simulation), base scenario: K = 3, slots 3/5/10, seed circle 100, farm 20 devices/mo., 8 runs × 60 months.
Date: October 18, 2026.

---

## 1. Results in plain words

The platform's main fear was tested: a person with money buys phones, creates a thousand fake "people" and collects payments on par with real participants. What the model showed:

1. **The farm cannot bootstrap itself.** A ring of fakes cannot activate under the rules — the farm must buy vouching from real participants. Entry is paid, and that is the main barrier.
2. **The farm is a temporary problem if the network grows.** At the worst moment (year two) the farm reaches ~40% of the network, but honest members dilute it: by year five — 12%. A non-growing network is more vulnerable; a growing one heals itself.
3. **The rules are confirmed.** Three vouchers, generous slots, a seed circle of 300+ people — these settings give the farm the minimum and honest growth the maximum.
4. **Inconvenient honesty.** At a unit price of $0.01 the farm turns a profit only after roughly 3 years. That is why demurrage and fees must not be switched off: they are part of the defense, not an option.
5. **Reliability check.** The model was run independently twice — with different random generators (numpy in the original run, pure Python in the reproduction). The picture matched: the same peaks, the same percentages, the same conclusions. The result is not a fluke.

## 2. Why "3 years in the red" is itself the barrier

Objection: nobody will run at a loss for years maintaining a farm. That is true — and it strengthens the conclusion rather than weakening it. The attack economics from the base run:

- **The maximum pit is ~$36k in month 23.** The farm only turns a profit in month 38, having invested ~$90k by then.
- **Capital is frozen for years.** The money cannot be touched for 3+ years. With any reasonable discount rate, payback drifts beyond 5 years; the alternative of "putting the same money into bonds" is more profitable.
- **A bet on someone else's success.** The farm earns only if the unit price rises to $0.01 and stays there. A network with a public ledger, where a ring of ~1140 mutually vouching wallets is visible to everyone, carries the risk that the network will not exist at all in 3 years. Any noticeable probability of that outcome zeroes the expected profit.
- **Its own dump would crash the price.** The model assumes a fixed price of $0.01, but in reality the farm needs to sell ~29 million units — its own dump is what would crash the price. Real revenue is far below the estimate.
- **Demurrage and fees are not included in the model.** With them, the payback point moves even further away, possibly beyond the horizon entirely.

**Bottom line.** The farm is not "a profitable venture with a long start", but a speculative bet: tens of thousands of dollars, three years of losses, a publicly visible scheme, and a payoff only if someone else's project takes off. The phrasing of section 7.7 — "the farm breaks even in month 38" — reads as "you must hold tens of thousands of dollars in the red for 3 years with the ring fully visible", which is precisely the intended "income < cost" barrier.

---

© 2026 Kirill Lapchinsky (Лапчинский Кирилл) · rumatakira74@gmail.com · Telegram @rumatakira
