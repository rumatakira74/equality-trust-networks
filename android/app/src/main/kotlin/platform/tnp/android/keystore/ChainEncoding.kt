package platform.tnp.android.keystore

/**
 * Кодирование цепочки аттестации в формате ядра (core/attest):
 * DER-сертификаты, лист первым, 4-байтовый big-endian префикс длины
 * на каждый. Корни НЕ передаются — зашиты в APK (7.2).
 */
object ChainEncoding {

    fun encode(certs: List<ByteArray>): ByteArray {
        var out = ByteArray(0)
        for (der in certs) {
            out += prefix(der.size) + der
        }
        return out
    }

    /** Разбор блоба (для тестов и отображения). */
    fun decode(blob: ByteArray): List<ByteArray> {
        val out = mutableListOf<ByteArray>()
        var off = 0
        while (off + 4 <= blob.size) {
            val n = ((blob[off].toInt() and 0xff) shl 24) or
                ((blob[off + 1].toInt() and 0xff) shl 16) or
                ((blob[off + 2].toInt() and 0xff) shl 8) or
                (blob[off + 3].toInt() and 0xff)
            off += 4
            require(n >= 0 && off + n <= blob.size) { "обрезанная цепочка" }
            out.add(blob.copyOfRange(off, off + n))
            off += n
        }
        require(off == blob.size) { "мусор в конце цепочки" }
        return out
    }

    private fun prefix(n: Int): ByteArray = byteArrayOf(
        (n ushr 24).toByte(), (n ushr 16).toByte(),
        (n ushr 8).toByte(), n.toByte(),
    )
}
