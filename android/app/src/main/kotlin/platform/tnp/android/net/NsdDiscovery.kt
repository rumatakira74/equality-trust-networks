package platform.tnp.android.net

import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo

/**
 * NSD/mDNS-обнаружение в локальной Wi-Fi-сети (12.3, шаг 4 каскада).
 *
 * Go-узел уже анонсирует и находит пиры через libp2p mDNS (служба
 * "tnp-<первые 8 hex network_id>", домен _p2p._udp) — это тот же
 * протокол mDNS, что и NSD. Этот класс — отображение найденных пиров
 * на уровне приложения (диагностика/bootstrap-экран).
 */
class NsdDiscovery(context: Context) {
    private val nsd = context.getSystemService(NsdManager::class.java)
    private var listener: NsdManager.DiscoveryListener? = null

    /** Запуск обнаружения; onFound вызывается в потоке NSD. */
    fun discover(onFound: (String) -> Unit) {
        stop()
        val l = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(serviceType: String) {}
            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {}
            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {}
            override fun onDiscoveryStopped(serviceType: String) {}
            override fun onServiceLost(serviceInfo: NsdServiceInfo) {}

            override fun onServiceFound(serviceInfo: NsdServiceInfo) {
                // только наши службы (tnp-<hex>)
                if (serviceInfo.serviceName.startsWith("tnp-")) {
                    onFound(serviceInfo.serviceName)
                }
            }
        }
        listener = l
        nsd.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, l)
    }

    fun stop() {
        listener?.let { nsd.stopServiceDiscovery(it) }
        listener = null
    }

    companion object {
        /** Домен libp2p mDNS (NSD-совместимый). */
        const val SERVICE_TYPE = "_p2p._udp."
    }
}
