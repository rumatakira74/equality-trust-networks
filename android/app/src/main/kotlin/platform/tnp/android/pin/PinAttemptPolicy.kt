package platform.tnp.android.pin

import android.content.Context
import android.content.SharedPreferences

/**
 * Лимит попыток PIN (тема UI-защиты владельца, п.1; 8.2 — «ограничение
 * числа попыток» одинаково для обычного и тревожного PIN).
 *
 * Политика (значения — константы с обоснованием, см. отчёт темы):
 *   - 5 неудачных попыток подряд — блокировка ввода на 30 с;
 *   - каждая следующая неудача удваивает блокировку: 30/60/120/240/480 с,
 *     кап 10 минут;
 *   - успешный вход (обычный ИЛИ тревожный — неотличимо) сбрасывает счётчик;
 *   - 24 ч с последней неудачи — счётчик сбрасывается временем;
 *   - уничтожение ключа НЕ вводится: опечатка не должна стоить кошелька.
 *
 * Обоснование порога: 4-значный PIN — 10^4 комбинаций; после ~9 неудач
 * блокировка достигает капа (10 мин), полный перебор — недели непрерывного
 * ввода при живом экране; каждая попытка дополнительно стоит PBKDF2
 * 120k итераций. Честный владелец с опечатками восстанавливается за 30 с.
 *
 * Хранилище: SharedPreferences (счётчик — не секрет; угроза «стереть
 * данные» стирает и PIN-файл — это заявленный путь «скрыть можно,
 * обнулить нельзя», README 8.2).
 */
class PinAttemptPolicy(
    private val maxFailures: Int = MAX_FAILURES,
    private val baseLockMs: Long = BASE_LOCK_MS,
    private val maxLockMs: Long = MAX_LOCK_MS,
    private val resetWindowMs: Long = RESET_WINDOW_MS,
) {
    /** Блокировка после n-й неудачи (0 — не блокируется). */
    fun lockMsAfter(failures: Int): Long {
        if (failures < maxFailures) return 0
        val shift = failures - maxFailures
        // удвоение без переполнения: кап проверяется на каждом шаге
        var lock = baseLockMs
        repeat(shift) {
            if (lock >= maxLockMs) return maxLockMs
            lock *= 2
        }
        return minOf(lock, maxLockMs)
    }

    /** Сброс счётчика временем: 24 ч без неудач. */
    fun resetByTime(lastFailureAt: Long, now: Long): Boolean =
        lastFailureAt > 0 && now - lastFailureAt >= resetWindowMs

    companion object {
        const val MAX_FAILURES = 5
        const val BASE_LOCK_MS = 30_000L
        const val MAX_LOCK_MS = 600_000L
        const val RESET_WINDOW_MS = 24L * 60 * 60 * 1000
    }
}

/**
 * Состояние счётчика попыток (перспективное хранилище — SharedPreferences;
 * логика — PinAttemptPolicy). Часы — System.currentTimeMillis (стена;
 * сдвиг часов назад удлиняет блокировку не сильнее окна сброса).
 */
class PinAttemptState(context: Context, private val policy: PinAttemptPolicy = PinAttemptPolicy()) {
    private val prefs: SharedPreferences =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun failures(): Int = prefs.getInt(KEY_FAILURES, 0)

    fun lastFailureAt(): Long = prefs.getLong(KEY_LAST_FAILURE, 0)

    /** Остаток блокировки, мс (0 — ввод не заблокирован). */
    fun lockedRemaining(now: Long = System.currentTimeMillis()): Long {
        val until = prefs.getLong(KEY_LOCK_UNTIL, 0)
        if (until <= now) {
            if (policy.resetByTime(lastFailureAt(), now)) reset()
            return 0
        }
        return until - now
    }

    /**
     * Неудачная попытка: инкремент счётчика, при достижении порога —
     * блокировка. Возвращает остаток блокировки после попытки (0 —
     * порог ещё не достигнут).
     */
    fun registerFailure(now: Long = System.currentTimeMillis()): Long {
        // окно сброса: старые неудачи не копятся вечно
        if (policy.resetByTime(lastFailureAt(), now)) {
            prefs.edit().putInt(KEY_FAILURES, 0).apply()
        }
        val n = failures() + 1
        val lock = policy.lockMsAfter(n)
        prefs.edit()
            .putInt(KEY_FAILURES, n)
            .putLong(KEY_LAST_FAILURE, now)
            .putLong(KEY_LOCK_UNTIL, if (lock > 0) now + lock else 0)
            .apply()
        return if (lock > 0) lock else 0
    }

    /** Сброс: успешный вход или окно времени. */
    fun reset() {
        prefs.edit().clear().apply()
    }

    companion object {
        const val PREFS = "tnp_pin_attempts"
        const val KEY_FAILURES = "failures"
        const val KEY_LAST_FAILURE = "last_failure_at"
        const val KEY_LOCK_UNTIL = "lock_until"
    }
}
