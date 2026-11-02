package platform.tnp.android

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import platform.tnp.android.keystore.KeystoreWallet
import platform.tnp.android.pin.PinAttemptPolicy

/**
 * Тесты лимита попыток PIN (тема UI-защиты владельца, п.1) и окна
 * подписи (A5): политика — чистая логика, значения по обоснованию
 * из отчёта темы; окно 0 — регресс-якорь README 8.1.
 */
class PinAttemptPolicyTest {

    private val p = PinAttemptPolicy()

    // -- рост блокировки: до порога нет, дальше удвоение с капом --

    @Test
    fun lock_grows_after_threshold() {
        assertEquals(0L, p.lockMsAfter(0))
        assertEquals(0L, p.lockMsAfter(4)) // ниже порога — не блокируется
        assertEquals(30_000L, p.lockMsAfter(5)) // 30 с
        assertEquals(60_000L, p.lockMsAfter(6))
        assertEquals(120_000L, p.lockMsAfter(7))
        assertEquals(240_000L, p.lockMsAfter(8))
        assertEquals(480_000L, p.lockMsAfter(9))
        assertEquals(600_000L, p.lockMsAfter(10)) // кап 10 мин
        assertEquals(600_000L, p.lockMsAfter(50)) // кап держится
    }

    // -- сброс счётчика временем: 24 ч без неудач --

    @Test
    fun reset_by_time_window() {
        val now = 1_000_000_000L
        val day = 24L * 60 * 60 * 1000
        assertTrue(p.resetByTime(now - day, now))
        assertFalse(p.resetByTime(now - day + 1, now))
        assertFalse(p.resetByTime(0, now)) // неудач не было — нечего сбрасывать
    }

    // -- A5: окно авторизации ключа — 0 (биометрия на каждую подпись) --

    @Test
    fun auth_window_is_zero() {
        assertEquals(0, KeystoreWallet.AUTH_TIMEOUT_SECONDS)
    }
}
