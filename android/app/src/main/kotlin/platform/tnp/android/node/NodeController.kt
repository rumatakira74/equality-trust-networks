package platform.tnp.android.node

import android.content.Context
import android.os.BatteryManager
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import tnp.Genesis
import tnp.Logger
import tnp.Node
import java.util.concurrent.ExecutorService
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicReference

/**
 * Управление узлом (12.2): уровень full/light по условиям
 * «зарядка + Wi-Fi». Полный узел — сервер DHT и полная сверка;
 * лёгкий — DHT client, заголовки чекпоинтов и Merkle-доказательства.
 * Смена уровня — перезапуск узла (состояние — в SQLite/лёгком стейте).
 */
object NodeController {
    private val current = AtomicReference<Node?>(null)
    private val lock = Any()

    var genesis: Genesis? = null
    var ownWallet: ByteArray? = null
    var manualAddrs: String? = null
    var log: Logger? = null

    /**
     * Фоновый однопоточный executor узла (ANR-блокер 8.1, авторизовано
     * владельцем; латентный баг, вскрыт симуляцией 8.1): старт/стоп/смена
     * уровня и ВСЕ cgo-вызовы моста (сверки, PoW, запросы состояния) —
     * вне главного потока Android; вызовы узла сериализованы. Готовность
     * узла UI узнаёт опросом nodeStats (паттерн этапа 7).
     */
    val executor: ExecutorService = Executors.newSingleThreadExecutor { r ->
        Thread(r, "tnp-node-io")
    }

    /** Запуск узла в фоне (главный поток не ждёт bootstrap/сверку). */
    fun ensureStartedAsync(context: Context) {
        executor.execute { ensureStarted(context) }
    }

    /** Остановка в фоне (close() может блокировать); after — после close. */
    fun stopAsync(context: Context, after: () -> Unit = {}) {
        executor.execute {
            stop(context)
            after()
        }
    }

    /** Пересборка узла при смене условий — в фоне. */
    fun reevaluateAsync(context: Context) {
        executor.execute { reevaluate(context) }
    }


    /**
     * Этап 7 (вопрос №38): true — узел без SQLite (в памяти).
     * Причина: modernc/sqlite -> libc.Xlstat -> raw SYS_LSTAT — SIGSYS
     * под app-seccopp (эмуляторы API 35/36 подтверждены); фикс —
     * upstream (fstatat в modernc.org/libc) или замена драйвера.
     */
    @Volatile var inMemoryDb: Boolean = false

    /** Условия полного узла (12.2): зарядка И Wi-Fi. */
    fun shouldBeFull(context: Context): Boolean {
        val bm = context.getSystemService(BatteryManager::class.java) ?: return false
        val charging = bm.isCharging ||
            bm.getIntProperty(BatteryManager.BATTERY_PROPERTY_STATUS) ==
            BatteryManager.BATTERY_STATUS_CHARGING
        val cm = context.getSystemService(ConnectivityManager::class.java) ?: return false
        val caps = cm.getNetworkCapabilities(cm.activeNetwork) ?: return false
        val wifi = caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)
        return charging && wifi
    }

    /** Запуск узла на уровне по условиям (или уже запущен). */
    fun ensureStarted(context: Context) {
        synchronized(lock) {
            if (current.get() != null) return
            val gen = genesis ?: return
            val full = shouldBeFull(context)
            try {
                val node = tnp.Tnp.startNode(
                    gen,
                    if (full) "full" else "light",
                    if (inMemoryDb) "" else context.getDatabasePath(DB_NAME).absolutePath,
                    ownWallet ?: ByteArray(0),
                    manualAddrs ?: "",
                    log,
                )
                current.set(node)
                // диагностика этапа 7: состояние узла сразу после старта
                android.util.Log.d("TnpNode", "запущен: " + node.statsText())
            } catch (e: Throwable) {
                android.util.Log.e("TnpNode", "запуск узла: " + e.message, e)
            }
        }
    }

    /** Пересборка узла при смене условий (зарядка/Wi-Fi). */
    fun reevaluate(context: Context) {
        synchronized(lock) {
            val node = current.get() ?: return
            val wantFull = shouldBeFull(context)
            val isFull = node.statsText().contains("\"level\":\"full\"")
            if (wantFull != isFull) {
                node.close()
                current.set(null)
                ensureStarted(context)
            }
        }
    }

    fun node(): Node? = current.get()

    fun stop(context: Context) {
        synchronized(lock) {
            current.get()?.close()
            current.set(null)
        }
    }

    const val DB_NAME = "tnp-node.db"
}
