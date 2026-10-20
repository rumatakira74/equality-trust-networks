package platform.tnp.android.keystore

import android.content.Context
import android.content.Intent
import android.provider.Settings
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricPrompt
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity

/**
 * Биометрический шлюз (этап 6): без настроенной биометрии регистрация
 * кошелька запрещена (8.1: signing_auth = BIOMETRIC_STRONG).
 * Отказ направляет на системный экран добавления отпечатка
 * (ACTION_BIOMETRIC_ENROLL).
 */
object BiometricGate {

    /** Доступен ли вход BIOMETRIC_STRONG (аппаратный отпечаток). */
    fun canUseStrongBiometrics(context: Context): Boolean =
        BiometricManager.from(context)
            .canAuthenticate(BiometricManager.Authenticators.BIOMETRIC_STRONG) ==
            BiometricManager.BIOMETRIC_SUCCESS

    /** Системный экран добавления биометрии (отказ в регистрации). */
    fun enrollIntent(): Intent =
        Intent(Settings.ACTION_BIOMETRIC_ENROLL).apply {
            putExtra(
                Settings.EXTRA_BIOMETRIC_AUTHENTICATORS_ALLOWED,
                BiometricManager.Authenticators.BIOMETRIC_STRONG,
            )
        }

    /**
     * Вход по отпечатку (8.2, этап 9): разблокировка приложения без
     * подписи (просмотр). Подтверждение деструктивных операций — PIN
     * (PIN-промпт — точка входа в режим-приманку; биометрия принуждаема
     * физически). Отпечаток предлагается, только когда приманка не
     * активна (DuressState.biometricAllowed).
     */
    fun authenticatePlain(
        activity: FragmentActivity,
        title: String,
        onResult: (Boolean) -> Unit,
    ) {
        val info = BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setNegativeButtonText("PIN")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()
        BiometricPrompt(
            activity,
            ContextCompat.getMainExecutor(activity),
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(
                    result: BiometricPrompt.AuthenticationResult,
                ) {
                    onResult(true)
                }

                override fun onAuthenticationError(code: Int, msg: CharSequence) {
                    onResult(false)
                }
            },
        ).authenticate(info)
    }

    /**
     * Подпись хеша ключом Keystore после биометрии: BiometricPrompt с
     * CryptoObject — ключ с setUserAuthenticationRequired(true) без
     * биометрии не подписывает.
     *
     * В режиме-приманке (тревожный вход) не вызывается никогда: ключ
     * подписи не используется (8.2).
     */
    fun signWithBiometric(
        activity: FragmentActivity,
        wallet: KeystoreWallet,
        title: String,
        hash: ByteArray,
        onResult: (ByteArray?) -> Unit,
    ) {
        val info = BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setNegativeButtonText("Отмена")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build()

        fun newPrompt() = BiometricPrompt(
            activity,
            ContextCompat.getMainExecutor(activity),
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                    val sig = result.cryptoObject?.signature
                        // plain-режим (API 35+): initSign повторно — уже
                        // внутри 30-секундного окна авторизации
                        ?: wallet.newSignature()
                    onResult(wallet.sign(sig, hash))
                }

                override fun onAuthenticationError(code: Int, msg: CharSequence) {
                    onResult(null)
                }
            },
        )

        val signature = wallet.newSignature()
        if (wallet.sigInitialized) {
            // initSign прошёл (окно авторизации открыто) — CryptoObject
            // валиден: биометрия криптографически связана с операцией
            try {
                android.util.Log.d("TnpBridge", "prompt.authenticate с CryptoObject")
                newPrompt().authenticate(info, BiometricPrompt.CryptoObject(signature))
            } catch (e: Exception) {
                android.util.Log.e("TnpBridge", "CryptoObject не принялся: " + e.message)
                android.util.Log.d("TnpBridge", "prompt.authenticate plain")
                newPrompt().authenticate(info)
            }
        } else {
            // API 35+: initSign на auth-bound ключе кидает
            // UserNotAuthenticatedException ДО старта операции —
            // CryptoObject невалиден ("Crypto primitive not
            // initialized"). Plain-биометрия: успешная аутентификация
            // открывает 30-с окно ключа (setUserAuthenticationParameters,
            // этап 6), подпись — в onAuthenticationSucceeded
            // (правка этапа 7).
            android.util.Log.d("TnpBridge", "prompt.authenticate plain (UNA)")
            newPrompt().authenticate(info)
        }
    }
}
