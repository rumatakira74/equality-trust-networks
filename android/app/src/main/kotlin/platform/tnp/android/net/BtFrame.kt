package platform.tnp.android.net

import java.io.ByteArrayOutputStream

/**
 * Кадр Bluetooth-канала (12.4): 4 байта BE длины + полезная нагрузка
 * (wire-кодирование транзакции). Выделен для unit-тестов.
 */
object BtFrame {
    const val MAX = 1_048_576

    fun encode(payload: ByteArray): ByteArray {
        val out = ByteArrayOutputStream(payload.size + 4)
        out.write(intToBytes(payload.size))
        out.write(payload)
        return out.toByteArray()
    }

    /** Разбор из буфера; null — кадр неполный или слишком большой. */
    fun decode(buf: ByteArray): ByteArray? {
        if (buf.size < 4) return null
        val n = ((buf[0].toInt() and 0xff) shl 24) or
            ((buf[1].toInt() and 0xff) shl 16) or
            ((buf[2].toInt() and 0xff) shl 8) or
            (buf[3].toInt() and 0xff)
        if (n < 0 || n > MAX || buf.size < 4 + n) return null
        return buf.copyOfRange(4, 4 + n)
    }

    private fun intToBytes(n: Int): ByteArray = byteArrayOf(
        (n ushr 24).toByte(), (n ushr 16).toByte(),
        (n ushr 8).toByte(), n.toByte(),
    )
}
