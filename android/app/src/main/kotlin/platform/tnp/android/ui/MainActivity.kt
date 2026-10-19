package platform.tnp.android.ui

import android.app.AlertDialog
import android.os.Bundle
import android.text.InputType
import android.view.ViewGroup
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.fragment.app.FragmentActivity
import platform.tnp.android.R
import platform.tnp.android.keystore.BiometricGate
import platform.tnp.android.node.NodeController
import platform.tnp.android.node.NodeService
import platform.tnp.android.pin.DuressState
import platform.tnp.android.pin.PinCheck
import platform.tnp.android.pin.PinManager
import platform.tnp.android.wallet.WalletFlow
import tnp.Genesis

/**
 * Главный экран (этап 6). Порядок настройки кошелька:
 * биометрия (BIOMETRIC_STRONG, иначе отказ + экран добавления
 * отпечатка) → PIN обычный и тревожный (8.2) → ключ Keystore с
 * аттестацией → register (PoW).
 *
 * PIN-вход: один экран, один ритм для обычного и тревожного PIN
 * (неразличимость, 8.2); тревожный вход активирует режим-приманку.
 */
class MainActivity : FragmentActivity() {

    private lateinit var pin: PinManager
    private lateinit var duress: DuressState
    private lateinit var flow: WalletFlow
    private lateinit var status: TextView
    private var genesis: Genesis? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        pin = PinManager(this)
        duress = DuressState(this)
        flow = WalletFlow(this)
        status = findViewById(R.id.status)

        // песочница: genesis с малыми PoW (№9 — финализация замерами)
        genesis = tnp.Tnp.sandboxGenesis(
            "tnp-sandbox-0001", 0, 3600,
            0x0010_0000_0000_0000L, 0x0010_0000_0000_0000L, 8, 1, 1,
        )
        NodeController.genesis = genesis
        NodeController.ownWallet = flow.address(genesis!!.networkID())
        NodeController.log = tnp.Logger { line -> runOnUiThread { appendStatus(line) } }

        findViewById<android.widget.Button>(R.id.btn_setup).setOnClickListener { setup() }
        findViewById<android.widget.Button>(R.id.btn_balance).setOnClickListener { balance() }
        findViewById<android.widget.Button>(R.id.btn_send).setOnClickListener { send() }
        findViewById<android.widget.Button>(R.id.btn_node).setOnClickListener { toggleNode() }
    }

    /** Настройка: биометрия → PIN (обычный + тревожный) → ключ → register. */
    private fun setup() {
        if (!BiometricGate.canUseStrongBiometrics(this)) {
            // отказ в регистрации без настроенной биометрии (8.1)
            Toast.makeText(this, "Настройте отпечаток", Toast.LENGTH_LONG).show()
            startActivity(BiometricGate.enrollIntent())
            return
        }
        askPin(getString(R.string.pin_setup_title)) { normal ->
            if (normal == null) return@askPin
            askPin(getString(R.string.pin_duress_setup_title), optional = true) { duressPin ->
                pin.setup(normal, duressPin ?: "")
                // ключ Keystore с аттестацией + register (PoW)
                val tx = flow.buildRegisterTx(genesis!!, androidId())
                if (tx == null) {
                    appendStatus("register: не собралась")
                    return@askPin
                }
                flow.signWithBiometric(
                    this, tx, tx.signingHash(), "Подтвердите регистрацию",
                ) { signed ->
                    if (signed == null) {
                        appendStatus("register: без подписи")
                    } else {
                        NodeService.start(this)
                        appendStatus("register: ${flow.submit(signed)}")
                    }
                }
            }
        }
    }

    /** Баланс: по Merkle-доказательству против чекпоинта (12.5). */
    private fun balance() {
        gate { unlocked ->
            if (!unlocked) return@gate
            if (duress.active) {
                // приманка: правдоподобный остаток, ядро не опрашивается
                appendStatus("баланс: ${duress.decoyBalance()}")
                return@gate
            }
            val node = NodeController.node() ?: run {
                appendStatus("узел не запущен")
                return@gate
            }
            val addr = flow.address(genesis!!.networkID()) ?: return@gate
            try {
                val b = node.balanceWithProof(addr)
                appendStatus("баланс (по чекпоинту): $b")
            } catch (e: Exception) {
                appendStatus("баланс: ${e.message}")
            }
        }
    }

    /** Перевод: seq от пиров → PoW → биометрическая подпись → публикация. */
    private fun send() {
        gate { unlocked ->
            if (!unlocked) return@gate
            if (duress.active) {
                // ключ подписи при приманке не используется (8.2)
                appendStatus("перевод недоступен")
                return@gate
            }
            val built = flow.buildTransferTx(
                genesis!!, flow.address(genesis!!.networkID())!!, 1,
            )
            if (built == null) {
                appendStatus("перевод: нет узла/пиров или seq")
                return@gate
            }
            val (tx, hash) = built
            flow.signWithBiometric(this, tx, hash, "Подтвердите перевод") { signed ->
                if (signed == null) {
                    appendStatus("перевод: без подписи")
                } else {
                    appendStatus("перевод: ${flow.submit(signed)}")
                }
            }
        }
    }

    /** Запуск/остановка foreground-узла. */
    private fun toggleNode() {
        if (NodeController.node() == null) {
            NodeService.start(this)
            appendStatus("узел: запуск")
        } else {
            NodeService.stop(this)
            appendStatus("узел: остановка")
        }
    }

    /**
     * Вход PIN: ОДИН экран для обычного и тревожного PIN (8.2).
     * Тревожный вход → режим-приманка (отпечаток отключён); обычный
     * после приманки — выход из приманки. Без PIN (не настроен) —
     * биометрия не предлагается.
     */
    private fun gate(onUnlocked: (Boolean) -> Unit) {
        if (!pin.isSet()) {
            Toast.makeText(this, "Сначала настройте кошелёк", Toast.LENGTH_SHORT).show()
            return onUnlocked(false)
        }
        askPin(getString(R.string.pin_title)) { entered ->
            val pin_ = entered ?: return@askPin
            when (pin.verify(pin_)) {
                PinCheck.NORMAL -> {
                    duress.exit()
                    onUnlocked(true)
                }

                PinCheck.DURESS -> {
                    duress.enter() // приманка: тот же экран, тот же ритм
                    onUnlocked(true)
                }

                PinCheck.NONE -> onUnlocked(false)
            }
        }
    }

    /**
     * Диалог PIN: одинаковый для входа и настройки, обычного и
     * тревожного (неразличимость, 8.2). optional: пустой = пропустить.
     */
    private fun askPin(
        title: String,
        optional: Boolean = false,
        onDone: (String?) -> Unit,
    ) {
        val input = EditText(this).apply {
            inputType = InputType.TYPE_CLASS_NUMBER or InputType.TYPE_NUMBER_VARIATION_PASSWORD
            layoutParams = LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT,
            )
        }
        val wrap = LinearLayout(this).apply {
            val pad = (resources.displayMetrics.density * 20).toInt()
            setPadding(pad, pad / 2, pad, 0)
            addView(input)
        }
        AlertDialog.Builder(this)
            .setTitle(title)
            .setView(wrap)
            .setPositiveButton("ОК") { _, _ -> onDone(input.text.toString()) }
            .apply {
                if (optional) {
                    setNegativeButton("Пропустить") { _, _ -> onDone(null) }
                } else {
                    setNegativeButton("Отмена") { _, _ -> onDone(null) }
                }
            }
            .show()
    }

    private fun androidId(): String =
        android.provider.Settings.Secure.getString(
            contentResolver, android.provider.Settings.Secure.ANDROID_ID,
        ) ?: ""

    private fun appendStatus(line: String) {
        status.append(line + "\n")
    }
}
