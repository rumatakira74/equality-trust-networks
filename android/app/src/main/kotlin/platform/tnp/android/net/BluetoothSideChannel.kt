package platform.tnp.android.net

import android.Manifest
import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothServerSocket
import android.bluetooth.BluetoothSocket
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.content.ContextCompat
import tnp.Node
import tnp.Tx
import java.io.DataInputStream
import java.io.DataOutputStream
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicReference

/**
 * Bluetooth-канал (12.4, 4.10): прямая передача ПОДПИСАННОЙ транзакции
 * получателю. «Получено по Bluetooth» ≠ оплата: транзакция кладётся
 * со статусом «ожидает публикации» (tnp.Node.OnOfflineTx); проверка
 * seq/prev — при публикации (PublishAwaiting), быстрая находка
 * эквивокации (12.4).
 *
 * Кадр: 4 байта BE длины + wire-кодирование транзакции.
 */
object BluetoothSideChannel {
    private const val SDP_NAME = "tnp-sidechannel"
    private val SDP_UUID = java.util.UUID.fromString(
        "5b1c0f3a-9d4e-4a2b-8e6f-1f2a3b4c5d60",
    )

    private val serverThread = AtomicReference<Thread?>(null)
    private val executor = Executors.newSingleThreadExecutor()

    /** Разрешения BT есть (S+ — BLUETOOTH_CONNECT). */
    fun hasPermissions(context: Context): Boolean =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            ContextCompat.checkSelfPermission(
                context, Manifest.permission.BLUETOOTH_CONNECT,
            ) == PackageManager.PERMISSION_GRANTED
        } else {
            true
        }

    /** Адаптер BT включён. */
    fun available(): Boolean =
        BluetoothAdapter.getDefaultAdapter()?.isEnabled == true

    /**
     * Приём транзакций (сервер RFCOMM): каждая кладётся в узел со
     * статусом «ожидает публикации». Запускается при старте узла.
     */
    @SuppressLint("MissingPermission")
    fun startServer(node: Node, onError: (String) -> Unit) {
        stopServer()
        val adapter = BluetoothAdapter.getDefaultAdapter() ?: return
        val thread = Thread {
            var server: BluetoothServerSocket? = null
            try {
                server = adapter.listenUsingRfcommWithServiceRecord(SDP_NAME, SDP_UUID)
                while (!Thread.currentThread().isInterrupted) {
                    val socket = server.accept() ?: break
                    executor.execute { receiveOne(socket, node, onError) }
                }
            } catch (e: Exception) {
                if (!Thread.currentThread().isInterrupted) onError(e.message ?: "bt")
            } finally {
                runCatching { server?.close() }
            }
        }
        thread.isDaemon = true
        thread.start()
        serverThread.set(thread)
    }

    fun stopServer() {
        serverThread.getAndSet(null)?.interrupt()
    }

    private fun receiveOne(socket: BluetoothSocket, node: Node, onError: (String) -> Unit) {
        try {
            DataInputStream(socket.inputStream).use { input ->
                val head = ByteArray(4)
                input.readFully(head)
                // читаем кадр по длине (лимит BtFrame.MAX)
                val len = ((head[0].toInt() and 0xff) shl 24) or
                    ((head[1].toInt() and 0xff) shl 16) or
                    ((head[2].toInt() and 0xff) shl 8) or
                    (head[3].toInt() and 0xff)
                if (len <= 0 || len > BtFrame.MAX) return
                val payload = ByteArray(len)
                input.readFully(payload)
                val tx: Tx? = tnp.Tnp.decodeTx(payload)
                if (tx != null) {
                    node.onOfflineTx(tx) // «ожидает публикации» (12.4)
                }
            }
        } catch (e: Exception) {
            onError(e.message ?: "bt")
        } finally {
            runCatching { socket.close() }
        }
    }

    /** Отправка подписанной транзакции на сопряжённое устройство. */
    @SuppressLint("MissingPermission")
    fun send(deviceName: String, tx: Tx, onError: (String) -> Unit) {
        val adapter = BluetoothAdapter.getDefaultAdapter()
            ?: return onError("нет Bluetooth")
        val device = adapter.bondedDevices.firstOrNull {
            it.name == deviceName
        } ?: return onError("устройство не сопряжено: $deviceName")
        executor.execute {
            try {
                adapter.cancelDiscovery()
                val socket = device.createRfcommSocketToServiceRecord(SDP_UUID)
                try {
                    socket.connect()
                    val frame = BtFrame.encode(tx.encode())
                    DataOutputStream(socket.outputStream).use { out ->
                        out.write(frame)
                        out.flush()
                    }
                } finally {
                    runCatching { socket.close() }
                }
            } catch (e: Exception) {
                onError(e.message ?: "bt")
            }
        }
    }
}
