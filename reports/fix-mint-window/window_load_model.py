#!/usr/bin/env python3
"""Нагрузка на узлы от окна mint: до/после фикса-темы fix-mint-window.

Считает почасовой приток mint-tx за один период (T = 30 сут, W = ±48 ч)
для трёх режимов подачи клиента:

  - «сразу при открытии окна» (худший случай, без джиттера);
  - «в случайный момент окна» (равномерный джиттер, модель аудита);
  - «в детерминированно-равномерный момент окна» (реализованный джиттер
    моста: submit_at = lo + H("tnp-mint-jitter:" ‖ addr ‖ period) mod
    (hi−lo), lo/hi — достижимые границы окна).

Семантики окна:
  - наследие (mint_slot_mode = 0): слот заявляемого периода; заявка
    возможна только после его конца (M4), поэтому окно достижимо лишь
    при u >= T−W и открывается на границе периода;
  - вариант A (mint_slot_mode = 1): слот периода выпуска; окно
    [u−W, u+W] усечено периодом — достижимо у каждого кошелька.

Верификация mint = Argon2id (32 MiB, ~25 мс на хосте, замеры этапа 12).
Только stdlib; `python3 window_load_model.py`.
"""
import hashlib
import random
import struct

T, W = 30 * 86400, 48 * 3600
H = T // 3600                      # часов в периоде = 720
VERIFY_MS = 25.0                   # мс на верификацию одного mint
G = 1_760_000_000


def h3(addr, period):
    return int.from_bytes(hashlib.sha256(addr + struct.pack(">Q", period)).digest()[:8], "big")


def u_of(addr, period):
    return h3(addr, period) % T


def jitter_hash(addr, period, span):
    """Реализованный джиттер (mobile/query.go: mintJitterOffset)."""
    if span == 0:
        return 0
    h = hashlib.sha256(b"tnp-mint-jitter:" + addr + struct.pack(">Q", period)).digest()
    return int.from_bytes(h[:8], "big") % span


def legacy_window(u):
    """Наследие: окно доступно только при u >= T-W; открывается на границе
    периода q+1 (час 0 следующего периода), длина u+W-T."""
    if u < T - W:
        return None
    return 0, (u + W - T)


def variant_a_window(u):
    """Вариант A: слот внутри периода выпуска; окно [u-W, u+W] усечено периодом."""
    lo, hi = max(0, u - W), min(T, u + W)
    return lo, hi - lo


def hist(n, rule, mode, seed=3):
    rnd = random.Random(seed)
    buckets = [0] * H
    for _ in range(n):
        addr = rnd.randbytes(32)
        u = u_of(addr, 5)
        w = rule(u)
        if not w:
            continue
        start, length = w
        if mode == "none":
            t = start
        elif mode == "uniform":
            t = start + rnd.random() * length
        elif mode == "hash":
            t = start + jitter_hash(addr, 5, length)
        else:
            raise ValueError(mode)
        buckets[min(H - 1, int(t // 3600))] += 1
    return buckets


def report(name, b, n):
    total = sum(b)
    peak = max(b)
    avg = total / H
    print(f"{name:56} mint/период={total/n*100:6.2f}% кошельков  "
          f"пик/час={peak/n*100:6.3f}% N  пик/среднее={peak/avg if avg else 0:5.1f}x")
    return total / n, peak / n, (peak / avg if avg else 0)


if __name__ == "__main__":
    n = 100_000
    print(f"N={n} кошельков, T={T//86400} сут, W={W//3600} ч, часов в периоде {H}\n")
    rows = []
    rows.append(("наследие (mode 0), клиент без джиттера",
                 report("наследие (mode 0), клиент без джиттера", hist(n, legacy_window, "none"), n)))
    rows.append(("наследие (mode 0), клиент с джиттером",
                 report("наследие (mode 0), клиент с джиттером", hist(n, legacy_window, "uniform"), n)))
    rows.append(("вариант A (mode 1), клиент без джиттера",
                 report("вариант A (mode 1), клиент без джиттера", hist(n, variant_a_window, "none"), n)))
    rows.append(("вариант A (mode 1), джиттер равномерный (аудит)",
                 report("вариант A (mode 1), джиттер равномерный (аудит)",
                        hist(n, variant_a_window, "uniform"), n)))
    rows.append(("вариант A (mode 1), джиттер реализованный (hash)",
                 report("вариант A (mode 1), джиттер реализованный (hash)",
                        hist(n, variant_a_window, "hash"), n)))

    # доля кошельков, чьё окно обрезано границей периода (источник пика)
    rnd = random.Random(3)
    truncated = sum(1 for _ in range(n) if u_of(rnd.randbytes(32), 5) < W)
    print(f"\nокно обрезано границей периода (u < W): {truncated/n*100:.2f}% кошельков — "
          f"без джиттера они подают в первый час периода разом")

    print()
    for N in (10**5, 10**6, 10**7):
        cur = N * W / T
        a = N
        print(f"N={N:>9}: верификаций за период — наследие {cur:>12,.0f}, "
              f"вариант A {a:>12,.0f}; CPU-секунд/период на узел: "
              f"{cur*VERIFY_MS/1000:>10,.0f} -> {a*VERIFY_MS/1000:>10,.0f}")
