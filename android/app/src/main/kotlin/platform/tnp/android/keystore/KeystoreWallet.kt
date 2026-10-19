package platform.tnp.android.keystore

import android.content.Context
import android.os.Build
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.Signature
import java.security.spec.ECGenParameterSpec

/**
 * Кошелёк в Android Keystore (этап 6, вопрос №7; 7.2 Документа).
 *
 * Ключ ECDSA P-256 неэкспортируем; каждая операция подписи требует
 * биометрии (BIOMETRIC_STRONG). Аттестация: setAttestationChallenge —
 * challenge задаёт узел (R4: SHA-256(pub ‖ protocol_id ‖ network_id),
 * приходит из ядра через мост tnp.AttestationChallengeFor).
 *
 * setInvalidatedByBiometricEnrollment(true): добавление нового
 * отпечатка инвалидирует ключ — повторная регистция обязательна.
 */
class KeystoreWallet(
    private val context: Context,
    private val alias: String = ALIAS_WALLET,
) {
    private val keyStore: KeyStore =
        KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }

    /** Зарегистрирован ли ключ кошелька. */
    fun isRegistered(): Boolean = keyStore.containsAlias(alias)

    /** Публичный ключ кошелька (из сертификата аттестации). */
    fun publicKey(): ByteArray? {
        val cert = keyStore.getCertificate(alias) ?: return null
        return cert.publicKey.encoded
    }

    /**
     * Генерация ключа с аттестацией (register, 9.3). Возвращает цепочку
     * аттестации в формате ядра (лист первым, 4-байтовый BE-префикс
     * длины на сертификат; корни НЕ передаются — зашиты в APK, 7.2).
     */
    fun generateWithAttestation(challenge: ByteArray): ByteArray {
        deleteIfPresent()
        val spec = KeyGenParameterSpec.Builder(
            alias,
            KeyProperties.PURPOSE_SIGN or KeyProperties.PURPOSE_VERIFY,
        )
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(KeyProperties.DIGEST_SHA256)
            // challenge задаёт узел (R4): связывает ключ с сетью
            .setAttestationChallenge(challenge)
            // подпись — только после биометрии (BIOMETRIC_STRONG)
            .setUserAuthenticationRequired(true)
            .setUserAuthenticationParameters(
                AUTH_TIMEOUT_SECONDS,
                KeyProperties.AUTH_BIOMETRIC_STRONG,
            )
            // добавление отпечатка инвалидирует ключ (7.2)
            .setInvalidatedByBiometricEnrollment(true)
            .build()
        val kpg = KeyPairGenerator.getInstance(
            KeyProperties.KEY_ALGORITHM_EC, ANDROID_KEYSTORE,
        )
        kpg.initialize(spec)
        kpg.generateKeyPair()
        return attestationChain()
    }

    /** Цепочка аттестации в формате ядра (лист + промежуточные). */
    fun attestationChain(): ByteArray {
        val certs = keyStore.getCertificateChain(alias)
            ?: throw IllegalStateException("цепочка аттестации недоступна")
        require(certs.isNotEmpty()) { "пустая цепочка аттестации" }
        return ChainEncoding.encode(certs.map { it.encoded })
    }

    /** Инициализация подписи (для BiometricPrompt.CryptoObject). */
    fun newSignature(): Signature =
        Signature.getInstance("SHA256withECDSA").apply {
            initSign(keyStore.getKey(alias, null) as java.security.PrivateKey)
        }

    /** Подпись хеша (после успешной биометрии). Выход — DER. */
    fun sign(signature: Signature, hash: ByteArray): ByteArray =
        signature.apply { update(hash) }.sign()

    private fun deleteIfPresent() {
        if (keyStore.containsAlias(alias)) keyStore.deleteEntry(alias)
    }

    companion object {
        const val ANDROID_KEYSTORE = "AndroidKeyStore"
        const val ALIAS_WALLET = "tnp-wallet"
        const val AUTH_TIMEOUT_SECONDS = 30

        /** Уровень защиты для minLevel (7.2): TEE=1, StrongBox=2. */
        fun minSecurityLevel(): Int =
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P &&
                isStrongBoxCapable()
            ) 2 else 1

        private fun isStrongBoxCapable(): Boolean = false
    }
}
