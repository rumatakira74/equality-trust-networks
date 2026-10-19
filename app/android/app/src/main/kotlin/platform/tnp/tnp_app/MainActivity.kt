package platform.tnp.tnp_app

import io.flutter.embedding.android.FlutterFragmentActivity
import io.flutter.embedding.engine.FlutterEngine
import platform.tnp.android.flutterbridge.FlutterBridge

/**
 * Главная активность Flutter-приложения (этап 7).
 *
 * FlutterFragmentActivity (не FlutterActivity): BiometricPrompt требует
 * FragmentActivity — подпись транзакций идёт через CryptoObject (8.1).
 * Платформенный слой этапа 6 (android/) скомпилирован в этот же модуль;
 * связь — MethodChannel platform.tnp/app (FlutterBridge).
 */
class MainActivity : FlutterFragmentActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        FlutterBridge.register(this, flutterEngine)
    }
}
