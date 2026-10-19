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
        val signature = wallet.newSignature()
        val prompt = BiometricPrompt(
            activity,
            ContextCompat.getMainExecutor(activity),
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                    val sig = result.cryptoObject?.signature ?: signature
                    onResult(wallet.sign(sig, hash))
                }

                override fun onAuthenticationError(code: Int, msg: CharSequence) {
                    onResult(null)
                }
            },
        )
        prompt.authenticate(
            BiometricPrompt.PromptInfo.Builder()
                .setTitle(title)
                .setNegativeButtonText("Отмена")
                .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
                .build(),
            BiometricPrompt.CryptoObject(signature),
        )
    }
}
