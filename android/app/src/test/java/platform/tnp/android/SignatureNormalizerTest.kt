package platform.tnp.android

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test
import platform.tnp.android.keystore.SignatureNormalizer
import java.math.BigInteger

/**
 * Юнит-тесты low-s нормализации ECDSA-подписи (находка №1 прогона
 * stage11-prep): высокая s заменяется на n − s, DER остаётся
 * каноничным (минимальная форма INTEGER).
 */
class SignatureNormalizerTest {

    private val N = BigInteger(
        "FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16,
    )
    private val HALF_N = N.shiftRight(1)

    /** DER-кодировка подписи (r, s) — как выдаёт Keystore. */
    private fun der(r: BigInteger, s: BigInteger): ByteArray {
        fun int(v: BigInteger): ByteArray {
            var b = v.toByteArray()
            var i = 0
            while (i < b.size - 1 && b[i] == 0.toByte()) i++
            b = b.copyOfRange(i, b.size)
            if (b[0].toInt() and 0x80 != 0) b = byteArrayOf(0) + b
            return byteArrayOf(0x02, b.size.toByte()) + b
        }

        val body = int(r) + int(s)
        return byteArrayOf(0x30, body.size.toByte()) + body
    }

    private fun rOf(der: ByteArray): Pair<BigInteger, BigInteger> {
        // SEQUENCE(1+1) INTEGER: 0x02 len ...
        var off = 2
        fun readInt(): BigInteger {
            assert(der[off].toInt() == 0x02)
            val len = der[off + 1].toInt()
            val v = BigInteger(1, der.copyOfRange(off + 2, off + 2 + len))
            off += 2 + len
            return v
        }
        return readInt() to readInt()
    }

    @Test
    fun highS_normalized_to_n_minus_s() {
        val r = BigInteger("8940132151695652516321075123003017910646236074709678658785734232312", 10)
        val highS = HALF_N.add(BigInteger.ONE) // строго выше n/2
        val out = SignatureNormalizer.normalizeLowS(der(r, highS))
        val (r2, s2) = rOf(out)
        assertEquals(r, r2) // r не меняется
        assertEquals(N.subtract(highS), s2) // s → n − s
        assertTrue(s2 <= HALF_N)
    }

    @Test
    fun lowS_unchanged() {
        val r = BigInteger("42", 10)
        val lowS = HALF_N // граница: остаётся как есть
        val out = SignatureNormalizer.normalizeLowS(der(r, lowS))
        assertArrayEquals(der(r, lowS), out) // байт-в-байт
    }

    @Test
    fun canonical_der_minimal_integers() {
        // r и s с ведущим нулём в старшем байте (положительные, но
        // старший бит установлен) — перекодировка минимальна
        val r = BigInteger("FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551", 16)
            .subtract(BigInteger.ONE) // почти n: 32 байта, старший бит 0xFF
        val s = HALF_N.subtract(BigInteger.ONE)
        val out = SignatureNormalizer.normalizeLowS(der(r, s))
        val (r2, s2) = rOf(out)
        assertEquals(r, r2)
        assertEquals(s, s2)
        // INTEGER-кодировка минимальна: нет лишних ведущих нулей
        assertEquals(0x30, out[0].toInt())
        assertEquals(out.size - 2, out[1].toInt())
    }

    @Test
    fun malformed_der_rejected() {
        assertThrows(IllegalArgumentException::class.java) {
            SignatureNormalizer.normalizeLowS(byteArrayOf(0x30, 0x02, 0x02, 0x01))
        }
        assertThrows(IllegalArgumentException::class.java) {
            SignatureNormalizer.normalizeLowS(ByteArray(0))
        }
        assertThrows(IllegalArgumentException::class.java) {
            // SEQUENCE с длиной больше тела
            SignatureNormalizer.normalizeLowS(
                byteArrayOf(0x30, 0x0A, 0x02, 0x01, 0x01, 0x02, 0x01, 0x02),
            )
        }
    }
}
