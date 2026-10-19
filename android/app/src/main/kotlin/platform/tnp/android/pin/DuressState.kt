package platform.tnp.android.pin

import android.content.Context
import android.content.SharedPreferences
import org.json.JSONArray

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
    }
}
