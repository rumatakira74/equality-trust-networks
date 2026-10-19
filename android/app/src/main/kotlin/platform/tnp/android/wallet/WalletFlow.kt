package platform.tnp.android.wallet

import android.content.Context
import platform.tnp.android.keystore.BiometricGate
import platform.tnp.android.keystore.KeystoreWallet
import androidx.fragment.app.FragmentActivity
import tnp.Genesis
import tnp.Tx

/**
 * Потоки кошелька поверх моста (tnp.*): регистрация с аттестацией
 * и PoW, перевод с получением seq от нескольких пиров перед подписью
 * (12.4), подпись — только через биометрию (Keystore).
 *
 * В режиме-приманке (DuressState.active) все операции возвращают
 * ошибку: ключ подписи не используется (8.2).
 */
class WalletFlow(
    private val context: Context,
    private val wallet: KeystoreWallet = KeystoreWallet(context),
) {
    private val bridge: Any = Any() // маркер: API-слой к ядру — tnp.*

    /** Адрес кошелька (32 байта) из публичного ключа Keystore. */
    fun address(networkId: ByteArray): ByteArray? {
        val pub = wallet.publicKey() ?: return null
        return tnp.Tnp.addressFromPublicKey(pub, networkId)
    }

    /**
     * Регистрация кошелька (9.3): challenge задаёт узел (R4),
     * аттестационная цепочка — из Keystore, PoW — усиленный Argon2id
     * (7.6, C10/C11; правка этапа 7: этап 6 решал лёгкий PoW — узел
     * всегда отвергал бы транзакцию с E_POW_HARD).
     * Требует настроенной биометрии (BIOMETRIC_STRONG) — проверка
     * вызывающим экраном до генерации ключа.
     *
     * guardiansJSON: ["hex", ...] — личные гардианы при создании
     * кошелька (11, вопрос №6); максимум 3, адреса известны реестру.
     *
     * Песочница (sandbox = true в манифесте): цепочка синтетическая
     * (SandboxAttestationChain, 6.5) — реальная цепочка KeyMint на
     * эмуляторе не сойдётся с синтетическими корнями; ключ всё равно
     * в Keystore (подпись — только биометрией).
     */
    fun buildRegisterTx(genesis: Genesis, androidId: String,
                        guardiansJSON: String = "[]"): Tx? {
        // генерация ключа ДО чтения публичного ключа (правка этапа 7:
        // на первой регистрации publicKey() ещё null — латентный баг
        // порядка этапа 6)
        val networkId = genesis.networkID()
        val pub0 = wallet.publicKey()
        val challenge0 =
            if (pub0 != null) tnp.Tnp.attestationChallengeFor(pub0, networkId)
            else ByteArray(0)
        wallet.generateWithAttestation(challenge0)
        val pub = wallet.publicKey() ?: return null
        val challenge = tnp.Tnp.attestationChallengeFor(pub, networkId)
        val sandbox = org.json.JSONObject(genesis.manifestJSON())
            .optBoolean("sandbox", false)
        val chain = if (sandbox) {
            tnp.Tnp.sandboxAttestationChain(pub, networkId, challenge)
        } else {
            wallet.attestationChain()
        }
        val addr = tnp.Tnp.addressFromPublicKey(pub, networkId) ?: return null
        val tx = tnp.Tnp.newRegisterGuardians(
            networkId, addr, pub, chain, challenge, androidId, guardiansJSON,
            genesis.protocolVersion(), System.currentTimeMillis() / 1000,
        ) ?: return null
        // register — усиленный PoW (Argon2id, 7.6)
        tx.solveHard(
            genesis.powTargetHard(), genesis.argon2MemKiB(),
            genesis.argon2Iterations(), genesis.argon2Par(),
        )
        return tx
    }

    /**
     * Перевод: seq/prev — от нескольких пиров перед подписью (12.4),
     * лёгкий PoW. Возвращает (tx, hash) — хеш уходит в Keystore.
     */
    fun buildTransferTx(
        genesis: Genesis,
        recipient: ByteArray,
        amount: Long,
        minAgree: Int = 2,
    ): Pair<Tx, ByteArray>? {
        val node = platform.tnp.android.node.NodeController.node() ?: return null
        val pub = wallet.publicKey() ?: return null
        val networkId = genesis.networkID()
        val addr = tnp.Tnp.addressFromPublicKey(pub, networkId) ?: return null
        val tip = node.tipFromPeers(addr, minAgree.toLong()) ?: return null
        val tx = tnp.Tnp.newTransfer(
            networkId, addr, tip.seq() + 1, tip.prev(), recipient, amount,
            genesis.protocolVersion(), System.currentTimeMillis() / 1000,
        )
        tx.solveLight(genesis.powTargetLight())
        return Pair(tx, tx.signingHash())
    }

    /** Подпись хеша биометрией (Keystore) и завершение транзакции. */
    fun signWithBiometric(
        activity: FragmentActivity,
        tx: Tx,
        hash: ByteArray,
        title: String,
        onDone: (Tx?) -> Unit,
    ) {
        BiometricGate.signWithBiometric(activity, wallet, title, hash) { sig ->
            if (sig == null) {
                onDone(null)
            } else {
                tx.setSignature(sig)
                onDone(tx)
            }
        }
    }

    /** Публикация подписанной транзакции (или Bluetooth при офлайне). */
    fun submit(tx: Tx): String {
        val node = platform.tnp.android.node.NodeController.node()
            ?: return "нет узла"
        val res = node.submit(tx)
        return "${res.outcome()}/${res.code()}"
    }
}
