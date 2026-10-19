package platform.tnp.android

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.runner.RunWith
import platform.tnp.android.net.BluetoothSideChannel
import platform.tnp.android.net.BtFrame
import tnp.Tx

/**
 * Instrumented-тесты Bluetooth-канала (12.4) и замеры моста (12.1,
 * вопрос №9): накладные расходы вызова моста и время PoW на устройстве.
 *
 * BT-передача между двумя устройствами — чек-лист (нужен второй телефон);
 * здесь — проверка разрешений/доступности и кадра.
 */
@RunWith(AndroidJUnit4::class)
class BluetoothAndBridgeInstrumentedTest {

    private val context get() = InstrumentationRegistry.getInstrumentation().targetContext

    @Test
    fun bluetooth_frameAndAvailability() {
        // кадр (unit-покрыт; здесь — на устройстве)
        val frame = BtFrame.encode(ByteArray(64))
        assertTrue(BtFrame.decode(frame)!!.size == 64)
        // доступность BT: на эмуляторе обычно нет — чек-лист для устройства
        val perms = BluetoothSideChannel.hasPermissions(context)
        assumeTrue("BT-разрешения не выданы (чек-лист)", perms)
    }

    /**
     * Замеры моста (12.1, №9): время лёгкого PoW (таргет 2^24 попыток),
     * время Argon2id (32 МиБ × 1 итерация — таргет выпуска), накладные
     * расходы вызова моста (build+encode+decode без PoW), подпись/проверка.
     * Результат — в файл для артефактов reports/stage6/.
     */
    @Test
    fun bridge_benchmark() {
        val gen = tnp.Tnp.sandboxGenesis(
            "tnp-bench", 0, 3600,
            0x0000_0001_0000_0000L, // таргет ~2^32/2^32: быстро, разминка
            0x0010_0000_0000_0000L, 8, 1, 1,
        )
        val pair = tnp.Tnp.softKey()!!
        val addr = tnp.Tnp.addressFromPublicKey(pair.pub(), gen.networkID())

        // (1) накладные расходы вызова моста: сборка + кодирование + разбор
        val t0 = System.nanoTime()
        val N = 200
        repeat(N) {
            val tx = tnp.Tnp.newTransfer(
                gen.networkID(), addr, 1, ByteArray(32), addr, 100,
                gen.protocolVersion(), 1234567,
            )
            val enc = tx.encode()
            tnp.Tnp.decodeTx(enc)
        }
        val perCallMs = (System.nanoTime() - t0) / 1_000_000.0 / N

        // (2) лёгкий PoW: таргет 2^24 пространства (замер вопроса №9)
        val tx1 = tnp.Tnp.newTransfer(
            gen.networkID(), addr, 1, ByteArray(32), addr, 100,
            gen.protocolVersion(), 1234567,
        )
        val t1 = System.nanoTime()
        tx1.solveLight(0x0000_0100_0000_0000L) // ~2^24 попыток
        val powLightMs = (System.nanoTime() - t1) / 1_000_000.0

        // (3) mint-PoW: Argon2id 32 МиБ × 1, таргет 2^56 (~64 итераций,
        // предложение №9; 2^52 на эмуляторе занимает десятки минут)
        val tx2 = tnp.Tnp.newMint(
            gen.networkID(), addr, 1, ByteArray(32), 1, 1000,
            gen.protocolVersion(), 1234567,
        )
        val t2 = System.nanoTime()
        tx2.solveHard(0x0040_0000_0000_0000L, 32768, 1, 1)
        val argonMs = (System.nanoTime() - t2) / 1_000_000.0

        // (4) подпись + проверка (мягкий ключ; на устройстве — Keystore)
        val sig = tnp.Tnp.signWithKey(pair.privDER(), tx1.signingHash())
        val t3 = System.nanoTime()
        tx1.setSignature(sig)
        val ok = tx1.verifySignature(pair.pub())
        val verifyMs = (System.nanoTime() - t3) / 1_000_000.0
        assertTrue(ok)

        // артефакт для отчёта
        val out = context.getExternalFilesDir(null) ?: context.filesDir
        java.io.File(out, "bridge-benchmark.txt").writeText(
            """
            |bridge_call_ms=$perCallMs
            |pow_light_2^24_ms=$powLightMs
            |argon2id_32MiB_ms=$argonMs
            |verify_ms=$verifyMs
            |model=${android.os.Build.MODEL}
            |abi=${android.os.Build.SUPPORTED_ABIS.joinToString(",")}
            |api=${android.os.Build.VERSION.SDK_INT}
            """.trimMargin(),
        )
        assertTrue(perCallMs >= 0)
    }
}
