package platform.tnp.android.flutterbridge

import android.Manifest
import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothServerSocket
import android.bluetooth.BluetoothSocket
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.content.ContextCompat
import java.io.DataInputStream
import java.io.DataOutputStream
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicReference

/**
 * Обмен адресами по Bluetooth при личной встрече (7.4: поручительство
 * через QR или Bluetooth; присутствие НЕ считается доказанным — только
 * обмен адресом кандидата без ручного ввода).
 *
 * Отдельный RFCOMM-сервис (не tnp-sidechannel): кадр = 4 байта BE длины
 * + байты адреса; одноразовый приём с таймаутом.
 */
object BtExchange {
    private const val SDP_NAME = "tnp-vouch-exchange"
    private val SDP_UUID = java.util.UUID.fromString(
        "5b1c0f3a-9d4e-4a2b-8e6f-1f2a3b4c5d61",
    )
    private val executor = Executors.newSingleThreadExecutor()
    private val server = AtomicReference<BluetoothServerSocket?>(null)

    fun hasPermissions(context: android.content.Context): Boolean =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            ContextCompat.checkSelfPermission(
                context, Manifest.permission.BLUETOOTH_CONNECT,
            ) == PackageManager.PERMISSION_GRANTED
        } else {
            true
        }

    fun available(): Boolean =
        BluetoothAdapter.getDefaultAdapter()?.isEnabled == true

    /** Отправка своего адреса кошелька на сопряжённое устройство. */
    @SuppressLint("MissingPermission")
    fun sendAddress(deviceName: String, address: ByteArray, onDone: (String?) -> Unit) {
        val adapter = BluetoothAdapter.getDefaultAdapter()
            ?: return onDone("нет Bluetooth")
        val device = adapter.bondedDevices.firstOrNull { it.name == deviceName }
            ?: return onDone("устройство не сопряжено: $deviceName")
        executor.execute {
            try {
                adapter.cancelDiscovery()
                val socket = device.createRfcommSocketToServiceRecord(SDP_UUID)
                try {
                    socket.connect()
                    DataOutputStream(socket.outputStream).use { out ->
                        val frame = ByteArray(4 + address.size)
                        frame[0] = (address.size shr 24).toByte()
                        frame[1] = (address.size shr 16).toByte()
                        frame[2] = (address.size shr 8).toByte()
                        frame[3] = address.size.toByte()
                        address.copyInto(frame, 4)
                        out.write(frame)
                        out.flush()
                    }
                    onDone(null)
                } finally {
                    runCatching { socket.close() }
                }
            } catch (e: Exception) {
                onDone(e.message ?: "bt")
            }
        }
    }

    /**
     * Одноразовый приём адреса (сервер RFCOMM, таймаут мс).
     * onResult(null, error) / onResult(address, null).
     */
    @SuppressLint("MissingPermission")
    fun receiveAddress(timeoutMs: Long, onResult: (ByteArray?, String?) -> Unit) {
        stop()
        val adapter = BluetoothAdapter.getDefaultAdapter()
            ?: return onResult(null, "нет Bluetooth")
        Thread {
            try {
                val serverSocket = adapter.listenUsingRfcommWithServiceRecord(SDP_NAME, SDP_UUID)
                server.set(serverSocket)
                adapter.cancelDiscovery()
                val socket: BluetoothSocket? = try {
                    serverSocket.accept(timeoutMs)
                } finally {
                    // accept с таймаутом закрывает сокет сервера; снимаем ссылку
                    server.set(null)
                }
                if (socket == null) {
                    onResult(null, "таймаут ожидания")
                    return
                }
                try {
                    DataInputStream(socket.inputStream).use { input ->
                        val head = ByteArray(4)
                        input.readFully(head)
                        val len = ((head[0].toInt() and 0xff) shl 24) or
                            ((head[1].toInt() and 0xff) shl 16) or
                            ((head[2].toInt() and 0xff) shl 8) or
                            (head[3].toInt() and 0xff)
                        if (len <= 0 || len > 64) {
                            onResult(null, "некорректный кадр")
                            return
                        }
                        val addr = ByteArray(len)
                        input.readFully(addr)
                        onResult(addr, null)
                    }
                } finally {
                    runCatching { socket.close() }
                }
            } catch (e: Exception) {
                onResult(null, e.message ?: "bt")
            }
        }.apply {
            isDaemon = true
            start()
        }
    }

    fun stop() {
        runCatching { server.getAndSet(null)?.close() }
    }
}
