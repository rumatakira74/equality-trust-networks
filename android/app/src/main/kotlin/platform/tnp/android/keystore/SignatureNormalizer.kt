package platform.tnp.android.keystore

import java.math.BigInteger

/**
 * Нормализация ECDSA-подписи под требования ядра (этап 9: каноничный
 * DER + low-s). Keystore (NONEwithECDSA) возвращает s как есть — при
 * s > n/2 ядро отвергает подпись с E_SIG (~50% register-подписей,
 * находка №1 прогона stage11-prep). Здесь: разбор DER, замена
 * s на n − s при s > n/2, каноничная (минимальная) перекодировка.
 */
object SignatureNormalizer {

    /** Порядок P-256 (secp256r1). */
    private val N = BigInteger(
        "FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16,
    )
    private val HALF_N = N.shiftRight(1)

    /** low-s нормализация DER-подписи (r, s). */
    fun normalizeLowS(der: ByteArray): ByteArray {
        val (r, s) = parse(der)
        val lowS = if (s > HALF_N) N.subtract(s) else s
        return encode(r, lowS)
    }

    /** Разбор DER: SEQUENCE { INTEGER r, INTEGER s }. */
    private fun parse(der: ByteArray): Pair<BigInteger, BigInteger> {
        var off = 0
        fun readByte(): Int {
            require(off < der.size) { "обрезанная подпись" }
            return der[off++].toInt() and 0xff
        }

        fun readLength(): Int {
            val first = readByte()
            if (first and 0x80 == 0) return first
            val n = first and 0x7f
            require(n in 1..4) { "некорректная длина DER" }
            var len = 0
            repeat(n) { len = (len shl 8) or readByte() }
            return len
        }

        fun readInteger(): BigInteger {
            require(readByte() == 0x02) { "ожидается INTEGER" }
            val len = readLength()
            require(len > 0 && off + len <= der.size) { "обрезанный INTEGER" }
            val value = BigInteger(1, der.copyOfRange(off, off + len))
            off += len
            return value
        }

        require(readByte() == 0x30) { "ожидается SEQUENCE" }
        val seqLen = readLength()
        require(seqLen == der.size - off) { "некорректная длина SEQUENCE" }
        val r = readInteger()
        val s = readInteger()
        require(off == der.size) { "мусор в конце подписи" }
        return r to s
    }

    /** Каноничная DER-кодировка (минимальная форма INTEGER). */
    private fun encode(r: BigInteger, s: BigInteger): ByteArray {
        fun intBytes(v: BigInteger): ByteArray {
            var b = v.toByteArray() // может иметь ведущий 0
            var i = 0
            while (i < b.size - 1 && b[i] == 0.toByte()) i++
            b = b.copyOfRange(i, b.size)
            if (b[0].toInt() and 0x80 != 0) b = byteArrayOf(0) + b // знак
            return byteArrayOf(0x02, b.size.toByte()) + b
        }

        val body = intBytes(r) + intBytes(s)
        // длины > 127 у ECDSA-подписи P-256 не бывает (r, s ≤ 33 байта),
        // но кодируем корректно в общем виде
        val lenBytes = if (body.size < 128) {
            byteArrayOf(body.size.toByte())
        } else {
            var n = body.size
            val stack = mutableListOf<Byte>()
            while (n > 0) { stack.add(0, (n and 0xff).toByte()); n = n shr 8 }
            byteArrayOf((0x80 or stack.size).toByte()) + stack.toByteArray()
        }
        return byteArrayOf(0x30) + lenBytes + body
    }
}
