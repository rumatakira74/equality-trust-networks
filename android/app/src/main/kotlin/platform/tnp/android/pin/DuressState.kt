package platform.tnp.android.pin

import android.content.Context
import android.content.SharedPreferences
import org.json.JSONArray
import tnp.Tnp

/**
 * Режим-приманка (8.2): вход по тревожному PIN. Правдоподобный остаток
 * и история — УРОВЕНЬ ПРИЛОЖЕНИЯ, не ядра: реальное состояние реестра
 * не искажается, ключ подписи при приманке не используется, сеть не
 * опрашивается. Отпечаток при активном тревожном отключён.
 *
 * Приманка настраивается вместе с PIN: владелец задаёт правдоподобную
 * «легенду» (остаток + история), которую увидит принуждающий.
 */
class DuressState(context: Context) {
    private val prefs: SharedPreferences =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    /** Тревожный режим активен (сессия). */
    @Volatile
    var active: Boolean = false
        private set

    /** Остаток легенды (приманка). */
    fun decoyBalance(): Long = prefs.getLong(KEY_BALANCE, 0)

    /** История легенды (приманка): строки «дата — сумма — получатель». */
    fun decoyHistory(): List<String> {
        val arr = prefs.getString(KEY_HISTORY, "[]")
        val out = mutableListOf<String>()
        val ja = JSONArray(arr ?: "[]")
        for (i in 0 until ja.length()) out.add(ja.getString(i))
        return out
    }

    /**
     * Подменной адрес легенды (8.2, F5): генерируется ОДИН раз (при
     * создании легенды) и хранится в tnp_duress — стабилен между сессиями
     * и рестартами, пока жива легенда. Адрес деривируется обычным путём от
     * случайного ключа (mobile.GenerateDecoyAddress) — валидного формата,
     * не деривируется от реального ключа/фингерпринта, ≠ реальному.
     * real — адрес настоящего кошелька (для проверки неравенства; может
     * быть null до создания кошелька).
     */
    fun ensureDecoyAddress(networkID: ByteArray, real: ByteArray?): ByteArray {
        prefs.getString(KEY_DECOY_ADDRESS, null)?.let { h ->
            unhexOrNull(h)?.let { b ->
                if (b.size == 32) return b
            }
        }
        val addr = Tnp.generateDecoyAddress(networkID, real ?: ByteArray(0))
            ?: throw IllegalStateException("подменной адрес не сгенерирован")
        prefs.edit().putString(KEY_DECOY_ADDRESS, hex(addr)).apply()
        return addr
    }

    /** Подменной адрес легенды (генерируется при отсутствии). */
    fun decoyAddress(networkID: ByteArray, real: ByteArray? = null): ByteArray =
        ensureDecoyAddress(networkID, real)

    /** Настройка легенды (при настройке тревожного PIN). */
    fun setLegend(balance: Long, history: List<String>) {
        prefs.edit()
            .putLong(KEY_BALANCE, balance)
            .putString(KEY_HISTORY, JSONArray(history).toString())
            .apply()
    }

    /** Вход по тревожному PIN: приманка активна до перезапуска. */
    fun enter() {
        active = true
    }

    /** Выход из приманки (после разблокировки обычным PIN). */
    fun exit() {
        active = false
    }

    /** Отпечаток отключён при активном тревожном режиме (8.2). */
    fun biometricAllowed(): Boolean = !active

    companion object {
        const val PREFS = "tnp_duress"
        const val KEY_BALANCE = "decoy_balance"
        const val KEY_HISTORY = "decoy_history"
        const val KEY_DECOY_ADDRESS = "decoy_address"

        private fun hex(b: ByteArray): String =
            b.joinToString("") { "%02x".format(it) }

        private fun unhexOrNull(s: String): ByteArray? = try {
            ByteArray(s.length / 2) { i ->
                ((Character.digit(s[i * 2], 16) shl 4) +
                    Character.digit(s[i * 2 + 1], 16)).toByte()
            }
        } catch (_: Exception) {
            null
        }
    }
}
