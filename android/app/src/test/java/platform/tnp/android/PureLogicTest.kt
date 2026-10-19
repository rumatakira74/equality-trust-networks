package platform.tnp.android

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertThrows
import org.junit.Test
import platform.tnp.android.keystore.ChainEncoding
import platform.tnp.android.net.BtFrame
import platform.tnp.android.pin.PinHash

/**
 * Unit-тесты приложения (JVM): кодирование цепочки аттестации (формат
 * core/attest), кадры Bluetooth-канала (12.4), PIN-хеш (8.2).
 */
class PureLogicTest {

    // -- цепочка аттестации: 4-байтовый BE-префикс длины, лист первым --

    @Test
    fun chainEncoding_roundtrip() {
        val certs = listOf(
            ByteArray(10) { it.toByte() },
            ByteArray(300) { (it * 3).toByte() }, // длина > 255: префикс длинный
        )
        val blob = ChainEncoding.encode(certs)
        assertEquals(blob.size, 4 + 10 + 4 + 300)
        val decoded = ChainEncoding.decode(blob)
        assertEquals(certs.size, decoded.size)
        certs.zip(decoded).forEach { (a, b) -> assertArrayEquals(a, b) }
    }

    @Test
    fun chainEncoding_garbage_rejected() {
        assertThrows(IllegalArgumentException::class.java) {
            ChainEncoding.decode(byteArrayOf(0, 0, 0, 99))
        }
        assertThrows(IllegalArgumentException::class.java) {
            ChainEncoding.decode(byteArrayOf(0, 0, 1, 0) + ByteArray(255))
        }
    }

    // -- Bluetooth-кадр (12.4): длина BE + нагрузка --

    @Test
    fun btFrame_roundtrip() {
        val payload = ByteArray(123) { (it * 7).toByte() }
        val frame = BtFrame.encode(payload)
        assertEquals(127, frame.size)
        assertArrayEquals(payload, BtFrame.decode(frame))
    }

    @Test
    fun btFrame_incomplete_and_oversize() {
        assertNull(BtFrame.decode(ByteArray(3)))
        val big = BtFrame.encode(ByteArray(BtFrame.MAX + 1))
        assertNull(BtFrame.decode(big))
    }

    // -- PIN (8.2): хеш детерминирован, соль меняет хеш, сравнение
    //    за постоянное время --

    @Test
    fun pinHash_deterministic_and_salted() {
        val salt = ByteArray(16) { it.toByte() }
        val a = PinHash.hash("1234", salt)
        val b = PinHash.hash("1234", salt)
        assertArrayEquals(a, b)
        val c = PinHash.hash("1234", ByteArray(16) { (it + 1).toByte() })
        assertEquals(32, a.size)
        org.junit.Assert.assertFalse(a.contentEquals(c))
    }

    @Test
    fun pinHash_constantTime() {
        val salt = ByteArray(16)
        val h = PinHash.hash("0000", salt)
        org.junit.Assert.assertTrue(PinHash.constantTimeEq(h, h.copyOf()))
        org.junit.Assert.assertFalse(PinHash.constantTimeEq(h, PinHash.hash("0001", salt)))
        org.junit.Assert.assertFalse(PinHash.constantTimeEq(h, ByteArray(31)))
    }
}
