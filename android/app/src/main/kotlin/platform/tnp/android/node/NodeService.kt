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
 * Этап 7: строки — литералы (не R): этот файл компилируется и в
 * Flutter-хост (app/), где пакет R другой.
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

    private fun createChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID, "Узел сети", NotificationManager.IMPORTANCE_LOW,
        )
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    private fun notification(): Notification =
        Notification.Builder(this, CHANNEL_ID)
            .setContentTitle("Сети доверия")
            // уведомление без сумм и адресов (8.2)
            .setContentText("Узел сети доверия работает")
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
