package platform.tnp.android.flutterbridge

import android.Manifest
import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.content.Context
import android.os.Build
import androidx.fragment.app.FragmentActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import org.json.JSONArray
import org.json.JSONObject
import platform.tnp.android.keystore.BiometricGate
import platform.tnp.android.keystore.KeystoreWallet
import platform.tnp.android.net.BluetoothSideChannel
import platform.tnp.android.node.NodeController
import platform.tnp.android.node.NodeService
import platform.tnp.android.pin.DuressState
import platform.tnp.android.pin.PinCheck
import platform.tnp.android.pin.PinManager
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicLong

/**
 * MethodChannel-мост Flutter -> платформенный слой (этап 7; 12.1:
 * Flutter — интерфейс, Kotlin — Keystore/биометрия/BT/узел, Go — ядро).
 *
 * Канал: platform.tnp/app. Обмен — JSON-строки и простые типы;
 * транзакции живут на Kotlin-стороне под идентификаторами (сборка ->
 * подпись биометрией -> публикация); ключ кошелька не покидает Keystore.
 *
 * Режим-приманка (8.2): при активном тревожном входе мост возвращает
 * правдоподобные данные легенды, подпись НЕ выполняется (ключ и
 * биометрия не используются), публикация не происходит; ответы
 * неотличимы от обычных.
 */
object FlutterBridge {
    private const val CHANNEL = "platform.tnp/app"

    // песочница эмулятора: те же константы, что в cmd/tnp-sandbox-peer
    // (идентичный genesis -> одинаковый manifest_hash)
    private const val SANDBOX_NETWORK_ID = "tnp-sandbox-0001"
    private const val SANDBOX_GENESIS_TIME = 1_760_000_000L

    private lateinit var appContext: Context
    private lateinit var activity: FragmentActivity
    private lateinit var pin: PinManager
    private lateinit var duress: DuressState
    private lateinit var wallet: KeystoreWallet

    // собранные транзакции: id -> Tx (подпись и публикация по id)
    private val pendingTx = ConcurrentHashMap<String, tnp.Tx>()
    private val txCounter = AtomicLong(0)

    // песочница эмулятора: genesis с малыми PoW (№9 — финализация на
    // живом телефоне); параметры сети приложение читает из манифеста
    private var genesis: tnp.Genesis? = null

    fun register(activity: FragmentActivity, engine: FlutterEngine) {
        this.activity = activity
        appContext = activity.applicationContext
        pin = PinManager(appContext)
        duress = DuressState(appContext)
        wallet = KeystoreWallet(appContext)
        ensureGenesis()
        MethodChannel(engine.dartExecutor.binaryMessenger, CHANNEL)
            .setMethodCallHandler { call, result -> handle(call, result) }
    }

    /**
     * Genesis песочницы + адрес своего кошелька для узла.
     *
     * Genesis-время ФИКСИРОВАНО: приложение и host-узел песочницы
     * (cmd/tnp-sandbox-peer, эмулятор подключается через 10.0.2.2)
     * обязаны строить идентичный genesis (manifest_hash совпадает).
     * Пир host-узла — только для песочницы эмулятора; реальные сети —
     * bootstrap-каскад (12.3).
     */
    private fun ensureGenesis() {
        if (genesis != null) return
        val gen = tnp.Tnp.sandboxGenesis(
            SANDBOX_NETWORK_ID, SANDBOX_GENESIS_TIME, 3600,
            0x0010_0000_0000_0000L, 0x0010_0000_0000_0000L, 8, 1, 1,
        )
        genesis = gen
        NodeController.genesis = gen
        NodeController.ownWallet = address()
        // полный multiaddr с /p2p/<id>: без ID компонента адрес молча
        // отбрасывается AddrInfoFromP2pAddr (bootstrap не подключается);
        // ID стабилен — детерминированный ключ узла песочницы
        NodeController.manualAddrs =
            "/ip4/10.0.2.2/udp/4001/quic-v1/p2p/QmNPJienGU6g9WBVbwaX26kBUr7ePWPHDNsaLMv9VVXdtP"
        NodeController.log = tnp.Logger { line ->
            android.util.Log.d("TnpNode", line)
        }
        // SQLite узла на Android отключён (в памяти): modernc/sqlite
        // вызывает raw SYS_LSTAT (libc.Xlstat), app-seccomp эмуляторов
        // API 35/36 даёт SIGSYS (вопрос №38, отчёт этапа 7); проверка
        // на живом телефоне — чек-лист
        NodeController.inMemoryDb = true
    }

    // ------------------------------------------------------------------
    // Вспомогательные
    // ------------------------------------------------------------------

    private fun address(): ByteArray? {
        val pub = wallet.publicKey() ?: return null
        return tnp.Tnp.addressFromPublicKey(pub, genesis!!.networkID())
    }

    private fun hex(b: ByteArray?): String =
        b?.joinToString("") { "%02x".format(it) } ?: ""

    private fun unhex(s: String): ByteArray =
        ByteArray(s.length / 2) { i ->
            ((Character.digit(s[i * 2], 16) shl 4) + Character.digit(s[i * 2 + 1], 16)).toByte()
        }

    private fun node(): tnp.Node? = NodeController.node()

    private fun store(tx: tnp.Tx): String {
        val id = "tx${txCounter.incrementAndGet()}"
        pendingTx[id] = tx
        return id
    }

    /** Легенда приманки (8.2): правдоподобный остаток и история. */
    private fun decoyWalletJson(): String = JSONObject().apply {
        put("exists", true)
        put("address", "")
        put("status", "ACTIVE")
        put("flagged", false)
        put("decoy", true)
        put("balance", duress.decoyBalance())
        put("liquid", duress.decoyBalance())
        put("locked", 0)
        put("age_days", 400)
        put("vouch_depth", 1)
        put("vouches_received", 3)
        put("personal_guardians", JSONArray())
        put("activators", JSONArray())
        put("threshold_p", 3000)
        put("pending_threshold", 0)
        put("pending_threshold_not_before", 0)
        put("last_period", 2)
        put("completed_mint_periods", 2)
        put("public_guardian", false)
        put("wards", 0)
        put("mint_suspended_until_period", 0)
        put("seq", 4)
        put("tip", "")
        put("registered_at", 0)
        put("activated_at", 0)
        put("epoch", 0)
        put("synthetic", false)
        put("seed", false)
        put("terminal", false)
    }.toString()

    private fun decoyHistoryJson(): String {
        val arr = JSONArray()
        duress.decoyHistory().forEach { line ->
            // легенда: строки «дата — сумма — получатель» (8.2)
            val parts = line.split(" — ")
            val e = JSONObject()
            e.put("seq", 0)
            e.put("type", "TRANSFER")
            e.put("txid", "")
            e.put("timestamp", 0)
            if (parts.size >= 2) {
                e.put("amount", parts[1].toLongOrNull() ?: 0)
                e.put("recipient", if (parts.size > 2) parts[2] else "")
            }
            e.put("period", 0)
            e.put("candidate", "")
            e.put("note", "")
            arr.put(e)
        }
        return arr.toString()
    }

    // ------------------------------------------------------------------
    // Обработчик
    // ------------------------------------------------------------------

    private fun handle(call: MethodCall, result: MethodChannel.Result) {
        android.util.Log.d("TnpBridge", "-> " + call.method)
        try {
            when (call.method) {
                // -- манифест и устройство --
                "manifest" -> result.success(genesis!!.manifestJSON())
                "biometricsAvailable" ->
                    result.success(BiometricGate.canUseStrongBiometrics(appContext))
                "openBiometricsEnroll" -> {
                    activity.startActivity(BiometricGate.enrollIntent())
                    result.success(null)
                }
                "androidId" -> result.success(
                    android.provider.Settings.Secure.getString(
                        activity.contentResolver,
                        android.provider.Settings.Secure.ANDROID_ID,
                    ) ?: "",
                )

                // -- PIN и приманка (8.1, 8.2) --
                "pinIsSet" -> result.success(pin.isSet())
                "setupPin" -> {
                    val normal = call.argument<String>("normal")!!
                    val duressPin = call.argument<String?>("duress") ?: ""
                    val decoyBalance = call.argument<Any>("decoyBalance")?.let { num ->
                        (num as? Number)?.toLong() ?: num.toString().toLong()
                    } ?: 0L
                    val decoyHistory = call.argument<List<String>>("decoyHistory") ?: emptyList()
                    pin.setup(normal, duressPin)
                    duress.setLegend(decoyBalance, decoyHistory)
                    result.success(true)
                }
                "verifyPin" -> {
                    val entered = call.argument<String>("pin")!!
                    // один и тот же код и ритм для обоих PIN (8.2)
                    when (pin.verify(entered)) {
                        PinCheck.NORMAL -> {
                            duress.exit()
                            result.success("normal")
                        }
                        PinCheck.DURESS -> {
                            duress.enter()
                            result.success("normal") // вход неотличим
                        }
                        PinCheck.NONE -> result.success("none")
                    }
                }
                "duressActive" -> result.success(duress.active)
                "exitDuress" -> {
                    duress.exit()
                    result.success(null)
                }
                "biometricLoginAllowed" -> result.success(duress.biometricAllowed())

                // -- кошелёк --
                "walletAddress" -> result.success(hex(address()))
                "walletRegistered" -> result.success(wallet.isRegistered())

                // -- сборка транзакций --
                "buildRegister" -> {
                    val guardians = call.argument<List<String>>("guardians") ?: emptyList()
                    val res = buildRegister(guardians)
                    if (res == null) result.error("BUILD", "register не собрался", null)
                    else result.success(mapOf("id" to res))
                }
                "buildTransfer" -> {
                    val recipient = unhex(call.argument<String>("recipient")!!)
                    val amount = (call.argument<Any>("amount") as Number).toLong()
                    val res = buildTransfer(recipient, amount)
                    if (res == null) result.error("BUILD", "перевод не собрался", null)
                    else result.success(res)
                }
                "buildMint" -> {
                    val period = (call.argument<Any>("period") as Number).toLong()
                    val amount = (call.argument<Any>("amount") as Number).toLong()
                    val res = buildMint(period, amount)
                    if (res == null) result.error("BUILD", "выпуск не собрался", null)
                    else result.success(mapOf("id" to res))
                }
                "buildVouch" -> {
                    val candidate = unhex(call.argument<String>("candidate")!!)
                    val addr = address()
                        ?: run {
                            result.error("NOWALLET", "кошелёк не создан", null)
                            return
                        }
                    val tp = tip(addr)
                        ?: run {
                            result.error("NOTIP", "нет пиров или seq", null)
                            return
                        }
                    val tx = tnp.Tnp.newVouch(
                        genesis!!.networkID(), addr, tp.first + 1, tp.second,
                        candidate, epochNow(), genesis!!.protocolVersion(),
                        System.currentTimeMillis() / 1000,
                    )
                    tx.solveLight(genesis!!.powTargetLight())
                    result.success(mapOf("id" to store(tx)))
                }
                "buildCancel" -> {
                    val pendingSeq = (call.argument<Any>("pendingSeq") as Number).toLong()
                    val addr = address()
                        ?: run {
                            result.error("NOWALLET", "кошелёк не создан", null)
                            return
                        }
                    val tp = tip(addr)
                        ?: run {
                            result.error("NOTIP", "нет пиров или seq", null)
                            return
                        }
                    val tx = tnp.Tnp.newTransferCancel(
                        genesis!!.networkID(), addr, tp.first + 1, tp.second,
                        pendingSeq, genesis!!.protocolVersion(),
                        System.currentTimeMillis() / 1000,
                    )
                    tx.solveLight(genesis!!.powTargetLight())
                    result.success(mapOf("id" to store(tx)))
                }
                "buildSetThreshold" -> {
                    val newP = (call.argument<Any>("newP") as Number).toLong()
                    val addr = address()
                        ?: run {
                            result.error("NOWALLET", "кошелёк не создан", null)
                            return
                        }
                    val tp = tip(addr)
                        ?: run {
                            result.error("NOTIP", "нет пиров или seq", null)
                            return
                        }
                    val tx = tnp.Tnp.newSetThreshold(
                        genesis!!.networkID(), addr, tp.first + 1, tp.second,
                        newP, genesis!!.protocolVersion(),
                        System.currentTimeMillis() / 1000,
                    )
                    tx.solveLight(genesis!!.powTargetLight())
                    result.success(mapOf("id" to store(tx)))
                }
                "buildGuardianOffer" -> {
                    val addr = address()
                        ?: run {
                            result.error("NOWALLET", "кошелёк не создан", null)
                            return
                        }
                    val tp = tip(addr)
                        ?: run {
                            result.error("NOTIP", "нет пиров или seq", null)
                            return
                        }
                    val tx = tnp.Tnp.newGuardianOffer(
                        genesis!!.networkID(), addr, tp.first + 1, tp.second,
                        genesis!!.protocolVersion(), System.currentTimeMillis() / 1000,
                    )
                    tx.solveLight(genesis!!.powTargetLight())
                    result.success(mapOf("id" to store(tx)))
                }
                "buildMigratePropose" -> {
                    // предложение публикуется в цепочке НОВОГО кошелька (11),
                    // поэтому вызывается на НОВОМ устройстве: свой адрес —
                    // новый, старый передан с экрана
                    val oldHex = call.argument<String>("old")!!
                    val nonce = (call.argument<Any>("nonce") as Number).toLong()
                    val approvals = call.argument<List<Map<String, Any>>>("approvals")
                        ?: emptyList()
                    val approvalsJson = JSONArray().apply {
                        approvals.forEach { a ->
                            put(JSONObject().put("guardian", a["guardian"]).put("sig", a["sig"]))
                        }
                    }.toString()
                    val newAddr = address()
                        ?: run {
                            result.error("NOWALLET", "кошелёк не создан", null)
                            return
                        }
                    val tp = tip(newAddr)
                        ?: run {
                            result.error("NOTIP", "нет пиров или seq", null)
                            return
                        }
                    val tx = tnp.Tnp.newMigratePropose(
                        genesis!!.networkID(), newAddr, tp.first + 1, tp.second,
                        unhex(oldHex), approvalsJson, nonce,
                        genesis!!.protocolVersion(), System.currentTimeMillis() / 1000,
                    ) ?: run {
                        result.error("BUILD", "миграция не собралась", null)
                        return
                    }
                    tx.solveLight(genesis!!.powTargetLight())
                    result.success(mapOf("id" to store(tx)))
                }
                "buildMigrateCancel" -> {
                    // отмена подписывается СТАРЫМ ключом в цепочке старого
                    // кошелька (11, вопрос №34) — вызывается на СТАРОМ устройстве
                    val newHex = call.argument<String>("new")!!
                    val proposeSeq = (call.argument<Any>("proposeSeq") as Number).toLong()
                    val addr = address()
                        ?: run {
                            result.error("NOWALLET", "кошелёк не создан", null)
                            return
                        }
                    val tp = tip(addr)
                        ?: run {
                            result.error("NOTIP", "нет пиров или seq", null)
                            return
                        }
                    val tx = tnp.Tnp.newMigrateCancel(
                        genesis!!.networkID(), addr, tp.first + 1, tp.second,
                        unhex(newHex), proposeSeq, genesis!!.protocolVersion(),
                        System.currentTimeMillis() / 1000,
                    )
                    tx.solveLight(genesis!!.powTargetLight())
                    result.success(mapOf("id" to store(tx)))
                }
                "migrateStatementHash" -> {
                    val oldHex = call.argument<String>("old")!!
                    val newHex = call.argument<String>("new")!!
                    val nonce = (call.argument<Any>("nonce") as Number).toLong()
                    // эпоха — текущая (подпись связана с эпохой, 11)
                    result.success(hex(
                        tnp.Tnp.migrateStatementHash(
                            genesis!!.networkID(), unhex(oldHex), unhex(newHex),
                            nonce, epochNow(),
                        ),
                    ))
                }
                "signMigrateStatement" -> {
                    // сторона гардиана: подпись утверждения миграции своим
                    // ключом (биометрия); отдаётся владельцу migrating-кошелька
                    val oldHex = call.argument<String>("old")!!
                    val newHex = call.argument<String>("new")!!
                    val nonce = (call.argument<Any>("nonce") as Number).toLong()
                    val title = call.argument<String>("title") ?: "Подтвердите миграцию"
                    if (duress.active) {
                        // приманка: ключ не используется (8.2)
                        result.success(mapOf("sig" to "", "fake" to true))
                        return
                    }
                    val hash = tnp.Tnp.migrateStatementHash(
                        genesis!!.networkID(), unhex(oldHex), unhex(newHex),
                        nonce, epochNow(),
                    )
                    BiometricGate.signWithBiometric(activity, wallet, title, hash) { sig ->
                        if (sig == null) result.success(mapOf("sig" to ""))
                        else result.success(mapOf("sig" to hex(sig)))
                    }
                }

                // -- подпись и публикация --
                "signTx" -> {
                    val id = call.argument<String>("id")!!
                    val title = call.argument<String>("title") ?: "Подтвердите операцию"
                    val tx = pendingTx[id]
                    if (tx == null) {
                        result.error("NOTFOUND", "транзакция не найдена", null)
                        return
                    }
                    if (duress.active) {
                        // приманка (8.2): подпись НЕ выполняется, ответ
                        // правдоподобен («ожидает подтверждения» ниже)
                        Thread.sleep(400)
                        result.success(mapOf("ok" to true, "fake" to true))
                        return
                    }
                    BiometricGate.signWithBiometric(activity, wallet, title, tx.signingHash()) { sig ->
                        android.util.Log.d("TnpBridge", "signTx callback sig=" + (sig != null))
                        if (sig == null) {
                            result.success(mapOf("ok" to false))
                        } else {
                            tx.setSignature(sig)
                            result.success(mapOf("ok" to true))
                        }
                    }
                }
                "submitTx" -> {
                    val id = call.argument<String>("id")!!
                    val tx = pendingTx[id]
                    if (tx == null) {
                        result.error("NOTFOUND", "транзакция не найдена", null)
                        return
                    }
                    if (duress.active) {
                        // приманка (8.2): публикации нет, ответ как у
                        // обычного перевода — «принято, ожидает сети»
                        pendingTx.remove(id)
                        result.success(mapOf("outcome" to "accepted", "code" to "OK", "fake" to true))
                        return
                    }
                    val n = node()
                        ?: run {
                            pendingTx.remove(id)
                            result.error("NONODE", "узел не запущен", null)
                            return
                        }
                    val res = n.submit(tx)
                    pendingTx.remove(id)
                    result.success(mapOf("outcome" to res.outcome(), "code" to res.code()))
                }
                "discardTx" -> {
                    pendingTx.remove(call.argument<String>("id")!!)
                    result.success(null)
                }

                // -- Bluetooth (12.4, 7.4) --
                "btAvailable" -> result.success(BluetoothSideChannel.available())
                "btPermissions" ->
                    result.success(BluetoothSideChannel.hasPermissions(appContext))
                "requestBtPermissions" -> {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                        activity.requestPermissions(arrayOf(Manifest.permission.BLUETOOTH_CONNECT), 7)
                    }
                    result.success(null)
                }
                "sendBluetooth" -> {
                    val id = call.argument<String>("id")!!
                    val device = call.argument<String>("device")!!
                    val tx = pendingTx.remove(id)
                    if (tx == null) {
                        result.error("NOTFOUND", "транзакция не найдена", null)
                        return
                    }
                    if (duress.active) {
                        result.success(mapOf("ok" to true, "fake" to true))
                        return
                    }
                    BluetoothSideChannel.send(device, tx) { err ->
                        if (err == null) result.success(mapOf("ok" to true))
                        else result.success(mapOf("ok" to false, "error" to err))
                    }
                }
                "btSendAddress" -> {
                    val device = call.argument<String>("device")!!
                    val addr = address()
                    if (addr == null) {
                        result.error("NOWALLET", "кошелёк не создан", null)
                        return
                    }
                    BtExchange.sendAddress(device, addr) { err ->
                        if (err == null) result.success(mapOf("ok" to true))
                        else result.success(mapOf("ok" to false, "error" to err))
                    }
                }
                "btReceiveAddress" -> {
                    if (!BtExchange.hasPermissions(appContext) || !BtExchange.available()) {
                        result.error("NOBT", "Bluetooth недоступен", null)
                        return
                    }
                    BtExchange.receiveAddress(60_000L) { addr, err ->
                        if (addr != null) result.success(mapOf("address" to hex(addr)))
                        else result.error("BTERR", err ?: "ошибка Bluetooth", null)
                    }
                }
                "btBondedDevices" -> {
                    // имена сопряжённых устройств (без адресов — приватность)
                    val adapter = BluetoothAdapter.getDefaultAdapter()
                    val names = if (adapter != null &&
                        BluetoothSideChannel.hasPermissions(appContext)
                    ) {
                        @SuppressLint("MissingPermission")
                        adapter.bondedDevices.map { it.name }
                    } else {
                        emptyList()
                    }
                    result.success(names)
                }

                // -- узел (12.2) --
                "nodeStart" -> {
                    NodeService.start(appContext)
                    result.success(true)
                }
                "nodeStop" -> {
                    NodeService.stop(appContext)
                    result.success(true)
                }
                "nodeStats" -> result.success(node()?.statsText())
                "publishAwaiting" -> {
                    node()?.publishAwaiting()
                    result.success(null)
                }
                "offlineStatus" -> {
                    val txid = unhex(call.argument<String>("txid")!!)
                    result.success(node()?.offlineStatus(txid) ?: "unknown")
                }
                "awaiting" -> result.success(node()?.awaitingJSON() ?: "[]")

                // -- запросы состояния (net/query.go через мост) --
                "balanceWithProof" -> {
                    if (duress.active) {
                        result.success(duress.decoyBalance())
                        return
                    }
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = address() ?: run {
                        result.error("NOWALLET", "кошелёк не создан", null)
                        return
                    }
                    result.success(n.balanceWithProof(addr))
                }
                "walletInfo" -> {
                    if (duress.active) {
                        result.success(decoyWalletJson())
                        return
                    }
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = call.argument<String?>("addr")?.let { unhex(it) } ?: address()
                    if (addr == null) {
                        result.error("NOWALLET", "кошелёк не создан", null)
                        return
                    }
                    result.success(n.walletInfoJSON(addr))
                }
                "vouchProgress" -> {
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = call.argument<String?>("addr")?.let { unhex(it) } ?: address()
                    result.success(n.vouchProgressJSON(addr ?: ByteArray(0)))
                }
                "slots" -> {
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = call.argument<String?>("addr")?.let { unhex(it) } ?: address()
                    result.success(n.slotsJSON(addr ?: ByteArray(0)))
                }
                "history" -> {
                    if (duress.active) {
                        result.success(decoyHistoryJson())
                        return
                    }
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = call.argument<String?>("addr")?.let { unhex(it) } ?: address()
                    val limit = (call.argument<Any>("limit") as? Number)?.toLong() ?: 0L
                    result.success(n.historyJSON(addr ?: ByteArray(0), limit))
                }
                "pendingTransfers" -> {
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = call.argument<String?>("addr")?.let { unhex(it) } ?: address()
                    result.success(n.pendingTransfersJSON(addr ?: ByteArray(0)))
                }
                "pendingMigrations" -> {
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = call.argument<String?>("addr")?.let { unhex(it) } ?: address()
                    result.success(n.pendingMigrationsJSON(addr ?: ByteArray(0)))
                }
                "mintWindow" -> {
                    val n = node() ?: run {
                        result.error("NONODE", "узел не запущен", null)
                        return
                    }
                    val addr = address() ?: run {
                        result.error("NOWALLET", "кошелёк не создан", null)
                        return
                    }
                    result.success(n.mintWindowJSON(addr))
                }
                "feeQuote" -> {
                    val amount = (call.argument<Any>("amount") as Number).toLong()
                    result.success(node()?.feeQuote(amount) ?: 0L)
                }
                "guardianOffers" -> result.success(node()?.guardianOffersJSON() ?: "[]")
                "registry" -> result.success(node()?.registryJSON() ?: "[]")
                "clock" -> result.success(node()?.clockJSON() ?: "{}")

                else -> result.notImplemented()
            }
            android.util.Log.d("TnpBridge", "<- " + call.method + " done")
        } catch (e: Exception) {
            android.util.Log.e("TnpBridge", "<- " + call.method + " err: " + e.message)
            result.error("ERR", e.message ?: e.javaClass.simpleName, null)
        }
    }

    // ------------------------------------------------------------------
    // Сборка транзакций (PoW здесь; подпись — биометрией по signTx)
    // ------------------------------------------------------------------

    /** tip цепочки: seq и prev от нескольких пиров (12.4, eclipse). */
    private fun tip(addr: ByteArray): Pair<Long, ByteArray>? {
        val n = node() ?: return null
        val t = n.tipFromPeers(addr, 2) ?: return null
        return t.seq() to t.prev()
    }

    private fun epochNow(): Long {
        val n = node() ?: return 1
        return JSONObject(n.clockJSON()).optLong("epoch", 1).coerceAtLeast(1)
    }

    private fun buildRegister(guardians: List<String>): String? {
        val gen = genesis!!
        // генерация ключа ДО чтения публичного ключа (латентный баг
        // порядка этапа 6: publicKey() до generateWithAttestation
        // возвращал null на первой регистрации)
        val pub0 = wallet.publicKey()
        val challenge0 = if (pub0 != null) {
            tnp.Tnp.attestationChallengeFor(pub0, gen.networkID())
        } else ByteArray(0)
        wallet.generateWithAttestation(challenge0)
        val pub = wallet.publicKey() ?: return null
        val networkId = gen.networkID()
        val challenge = tnp.Tnp.attestationChallengeFor(pub, networkId)
        val sandbox = JSONObject(gen.manifestJSON()).optBoolean("sandbox", false)
        val chain = if (sandbox) {
            tnp.Tnp.sandboxAttestationChain(pub, networkId, challenge)
        } else {
            wallet.attestationChain()
        }
        val addr = tnp.Tnp.addressFromPublicKey(pub, networkId) ?: return null
        // узел должен знать свой кошелёк (лёгкий уровень, 12.2)
        NodeController.ownWallet = addr
        val tx = tnp.Tnp.newRegisterGuardians(
            networkId, addr, pub, chain, challenge,
            android.provider.Settings.Secure.getString(
                activity.contentResolver, android.provider.Settings.Secure.ANDROID_ID,
            ) ?: "",
            JSONArray(guardians).toString(),
            gen.protocolVersion(), System.currentTimeMillis() / 1000,
        ) ?: return null
        // register — усиленный PoW Argon2id (7.6, C10/C11)
        tx.solveHard(gen.powTargetHard(), gen.argon2MemKiB(), gen.argon2Iterations(), gen.argon2Par())
        return store(tx)
    }

    /** Перевод (8.3): комиссия и признак «отложенный» видны ДО отправки. */
    private fun buildTransfer(recipient: ByteArray, amount: Long): Map<String, Any>? {
        val gen = genesis!!
        val n = node() ?: return null
        val addr = address() ?: return null
        val tip = n.tipFromPeers(addr, 2) ?: return null
        val tx = tnp.Tnp.newTransfer(
            gen.networkID(), addr, tip.seq() + 1, tip.prev(), recipient, amount,
            gen.protocolVersion(), System.currentTimeMillis() / 1000,
        )
        tx.solveLight(gen.powTargetLight())
        val info = JSONObject(n.walletInfoJSON(addr))
        val threshold = info.optLong("threshold_p", 0)
        val manifest = JSONObject(gen.manifestJSON())
        val delayed = amount > threshold &&
            manifest.optString("transfer_mode") == "free" &&
            !manifest.optBoolean("sandbox", false)
        return mapOf(
            "id" to store(tx),
            "fee" to n.feeQuote(amount),
            "delayed" to delayed,
            "delayHours" to manifest.optLong("transfer_delay_hours", 48),
        )
    }

    /** Выпуск (10.1, 7.6): усиленный PoW. */
    private fun buildMint(period: Long, amount: Long): String? {
        val gen = genesis!!
        val n = node() ?: return null
        val addr = address() ?: return null
        val tip = n.tipFromPeers(addr, 2) ?: return null
        val tx = tnp.Tnp.newMint(
            gen.networkID(), addr, tip.seq() + 1, tip.prev(), period, amount,
            gen.protocolVersion(), System.currentTimeMillis() / 1000,
        )
        tx.solveHard(gen.powTargetHard(), gen.argon2MemKiB(), gen.argon2Iterations(), gen.argon2Par())
        return store(tx)
    }

}
