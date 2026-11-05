package platform.tnp.android.node

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder

/**
 * Foreground service узла сети (12.2): держит Go-узел (AAR) живым,
 * пока приложение в фоне. Уровень full/light выбирает NodeController
 * (зарядка + Wi-Fi); смена уровня — перезапуск узла.
 *
 * Строки (темы fix-native-strings, fix-sweep): файл компилируется и в
 * Flutter-хост (app/, пакет R — platform.tnp.tnp_app), и в приложение
 * этапа 6 (android/, R — platform.tnp.android), поэтому R не
 * импортируется — ресурс ищется по имени (localized). Оба хоста
 * получают значения из ARB (values[-<lang>]/strings.xml, задача
 * generateNativeStrings); если ресурса в хосте нет — остаётся прежний
 * литерал.
 */
class NodeService : Service() {

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIFICATION_ID, notification())
        if (intent?.action == ACTION_STOP) {
            // close() узла может блокировать — в фон; stopSelf после
            NodeController.stopAsync(this) { stopSelf() }
            return START_NOT_STICKY
        }
        // ANR-блокер 8.1 (латентный баг, вскрыт симуляцией 8.1): старт
        // узла (bootstrap до ~11 с, первичная сверка журнала) — на
        // фоновом executor; главный поток свободен, готовность — опрос
        // nodeStats (паттерн этапа 7)
        NodeController.ensureStartedAsync(this)
        return START_STICKY
    }

    /**
     * Строка из ресурсов хоста по имени (темы fix-native-strings,
     * fix-sweep): R не импортируется, потому что файл собирается в двух
     * хостах с разными пакетами R. Оба хоста берут значения из ARB
     * (values[-<lang>]/strings.xml, задача generateNativeStrings); если
     * нужного имени в ресурсах нет — остаётся прежний литерал, то есть
     * поведение не меняется.
     */
    private fun localized(name: String, fallback: String): String {
        val id = resources.getIdentifier(name, "string", packageName)
        return if (id != 0) getString(id) else fallback
    }

    private fun createChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            localized("node_channel_name", "Узел сети"),
            NotificationManager.IMPORTANCE_LOW,
        )
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    private fun notification(): Notification =
        Notification.Builder(this, CHANNEL_ID)
            .setContentTitle(localized("node_notification_title", "Сети доверия"))
            // уведомление без сумм и адресов (8.2)
            .setContentText(
                localized("node_notification_text", "Узел сети доверия работает"),
            )
            .setSmallIcon(android.R.drawable.stat_notify_sync_noanim)
            .setOngoing(true)
            .build()

    companion object {
        const val CHANNEL_ID = "tnp_node"
        const val NOTIFICATION_ID = 1
        const val ACTION_STOP = "platform.tnp.android.node.STOP"

        fun start(context: Context) {
            context.startForegroundService(Intent(context, NodeService::class.java))
        }

        fun stop(context: Context) {
            val intent = Intent(context, NodeService::class.java).apply {
                action = ACTION_STOP
            }
            context.startForegroundService(intent)
        }
    }
}
