package platform.tnp.android.node

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import platform.tnp.android.R

/**
 * Foreground service узла сети (12.2): держит Go-узел (AAR) живым,
 * пока приложение в фоне. Уровень full/light выбирает NodeController
 * (зарядка + Wi-Fi); смена уровня — перезапуск узла.
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
            NodeController.stop(this)
            stopSelf()
            return START_NOT_STICKY
        }
        NodeController.ensureStarted(this)
        return START_STICKY
    }

    private fun createChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            getString(R.string.node_channel),
            NotificationManager.IMPORTANCE_LOW,
        )
        getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
    }

    private fun notification(): Notification =
        Notification.Builder(this, CHANNEL_ID)
            .setContentTitle(getString(R.string.app_name))
            .setContentText(getString(R.string.node_notification))
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
