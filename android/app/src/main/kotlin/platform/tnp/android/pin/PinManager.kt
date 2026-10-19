package platform.tnp.android.pin

import android.content.Context
import android.os.Build
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.io.File
import java.security.KeyStore
import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * PIN обычный и тревожный (8.2 Документа).
 *
 * Неразличимость при вводе: один экран, один ритм — приложение проверяет
 * ОБА хеша одинаковым кодом и не показывает, какой PIN сработал.
 * Вход по тревожному — режим-приманка (DuressState): правдоподобный
 * остаток и история на уровне приложения; ключ подписи не используется.
 * Отпечаток при активном тревожном режиме отключён.
 *
 * Хранилище: файл, зашифрованный AES-GCM ключом Keystore
 * (tnp-pin-master); allowBackup=false — в бэкап не попадает.
 */
class PinManager(context: Context) {
    private val file: File = File(context.filesDir, FILE_NAME)
    private val keyStore: KeyStore =
        KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }
    private val rng = SecureRandom()

    /** Настроен ли PIN (обычный или оба). */
    fun isSet(): Boolean = read() != null

    /**
     * Настройка: обычный PIN обязателен, тревожный — опционален
     * (пустая строка = не задан). Оба хеша пишутся одним шифрованием —
     * по файлу нельзя узнать, задан ли тревожный PIN.
     */
    fun setup(normalPin: String, duressPin: String) {
        require(normalPin.length in 4..16) { "PIN: 4-16 символов" }
        val salt = ByteArray(16).also { rng.nextBytes(it) }
        val rec = Record(
            salt = salt,
            normalHash = hash(normalPin, salt),
            duressHash = if (duressPin.isBlank()) null else hash(duressPin, salt),
        )
        write(rec)
    }

    /**
     * Проверка PIN. Возвращает NORMAL / DURESS / NONE. Один и тот же код
     * для обоих хешей — по времени/поведению вход неразличим.
     */
    fun verify(pin: String): PinCheck {
        val rec = read() ?: return PinCheck.NONE
        val h = hash(pin, rec.salt)
        if (constantTimeEq(h, rec.normalHash)) return PinCheck.NORMAL
        if (rec.duressHash != null && constantTimeEq(h, rec.duressHash)) {
            return PinCheck.DURESS
        }
        return PinCheck.NONE
    }

    fun wipe() {
        if (file.exists()) file.delete()
    }

    // -- хранилище: AES-GCM (Keystore) поверх файла --

    private fun masterKey(): SecretKey {
        if (!keyStore.containsAlias(ALIAS_MASTER)) {
            val kg = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE)
            kg.init(
                KeyGenParameterSpec.Builder(
                    ALIAS_MASTER,
                    KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
                )
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setKeySize(256)
                    .build(),
            )
            kg.generateKey()
        }
        return keyStore.getKey(ALIAS_MASTER, null) as SecretKey
    }

    private fun write(rec: Record) {
        val plain = rec.encode()
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.ENCRYPT_MODE, masterKey())
        val iv = cipher.iv
        val enc = cipher.doFinal(plain)
        file.writeBytes(iv + enc)
    }

    private fun read(): Record? {
        if (!file.exists()) return null
        val blob = file.readBytes()
        if (blob.size < 13) return null
        val iv = blob.copyOfRange(0, 12)
        val enc = blob.copyOfRange(12, blob.size)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        return try {
            cipher.init(Cipher.DECRYPT_MODE, masterKey(), GCMParameterSpec(128, iv))
            Record.decode(cipher.doFinal(enc))
        } catch (e: Exception) {
            null
        }
    }

    private fun hash(pin: String, salt: ByteArray): ByteArray =
        PinHash.hash(pin, salt)

    /** Сравнение за постоянное время (не раскрывает, какой хеш сравнивался). */
    private fun constantTimeEq(a: ByteArray, b: ByteArray): Boolean =
        PinHash.constantTimeEq(a, b)

    private data class Record(
        val salt: ByteArray,
        val normalHash: ByteArray,
        val duressHash: ByteArray?,
    ) {
        fun encode(): ByteArray {
            var out = ByteArray(0)
            out += u32(salt.size) + salt
            out += u32(normalHash.size) + normalHash
            out += u32(duressHash?.size ?: 0)
            duressHash?.let { out += it }
            return out
        }

        companion object {
            fun decode(b: ByteArray): Record {
                var off = 0
                fun take(): ByteArray {
                    val n = ((b[off].toInt() and 0xff) shl 24) or
                        ((b[off + 1].toInt() and 0xff) shl 16) or
                        ((b[off + 2].toInt() and 0xff) shl 8) or
                        (b[off + 3].toInt() and 0xff)
                    off += 4
                    val out = b.copyOfRange(off, off + n)
                    off += n
                    return out
                }

                val salt = take()
                val normal = take()
                val duressLen = ((b[off].toInt() and 0xff) shl 24) or
                    ((b[off + 1].toInt() and 0xff) shl 16) or
                    ((b[off + 2].toInt() and 0xff) shl 8) or
                    (b[off + 3].toInt() and 0xff)
                off += 4
                val duress = if (duressLen > 0) take() else null
                return Record(salt, normal, duress)
            }
        }
    }

    companion object {
        const val ANDROID_KEYSTORE = "AndroidKeyStore"
        const val ALIAS_MASTER = "tnp-pin-master"
        const val FILE_NAME = "pin_state.bin"
        const val PBKDF2_ITERATIONS = 120_000

        private fun u32(n: Int): ByteArray = byteArrayOf(
            (n ushr 24).toByte(), (n ushr 16).toByte(),
            (n ushr 8).toByte(), n.toByte(),
        )
    }
}

enum class PinCheck { NONE, NORMAL, DURESS }
