package platform.tnp.android.pin

import javax.crypto.SecretKeyFactory
import javax.crypto.spec.PBEKeySpec

/**
 * Хеш PIN (8.2): PBKDF2-HMAC-SHA256. Keystore-ключ PIN не защищает
 * (его защищает Keystore) — нужна устойчивость к офлайн-перебору
 * словаря по украденному файлу. Выделен для unit-тестов.
 */
object PinHash {
    const val ITERATIONS = 120_000
    const val KEY_BITS = 256

    fun hash(pin: String, salt: ByteArray): ByteArray {
        val spec = PBEKeySpec(pin.toCharArray(), salt, ITERATIONS, KEY_BITS)
        return SecretKeyFactory.getInstance("PBKDF2WithHmacSHA256")
            .generateSecret(spec).encoded
    }

    /** Сравнение за постоянное время (не раскрывает длину/позицию). */
    fun constantTimeEq(a: ByteArray, b: ByteArray): Boolean {
        if (a.size != b.size) return false
        var diff = 0
        for (i in a.indices) diff = diff or (a[i].toInt() xor b[i].toInt())
        return diff == 0
    }
}
