// Flutter-хост приложения (этап 7): только Android. Платформенный слой
// этапа 6 (android/app/src/main/kotlin) компилируется сюда напрямую
// (srcDir, без дублирования); AAR ядра — gomobile bind (android/libs).
// Экран этапа 6 (ui/MainActivity.kt) исключён — его заменил Flutter-UI.
import java.nio.file.Files
import java.nio.file.attribute.PosixFilePermission
import java.nio.file.attribute.PosixFilePermissions
import java.security.MessageDigest
import java.util.Properties

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

// Боевая release-подпись (тема A8): единый ключ std для всех std-сетей,
// отдельный rec для rec-сетей. Signing-конфиг (пути/пароли) генерирует
// конфигуратор (cmd/tnp-testnet configure) в локальный файл вне git:
// путь — env TNP_SIGNING_PROPERTIES (или -P), дефолт
// ~/.tnp-signing/signing.properties; вид ключа — TNP_KEY_KIND=std|rec
// (env или -P; печатает configure). Приватные ключи и пароли в git и CI
// не попадают; отсутствие/недоступность — явный отказ release-сборки,
// debug-сборки не затронуты. Тема fix-release-keys (P1 аудита Sonnet-2):
// release-сборка дополнительно сверяет SHA-256 файла ключа с отпечатком
// TNP_SIGNING_KEY_SHA256 и требует права файлов ключей не шире 0600
// (задача verifySigning); release-артефакт без боевой подписи отвергается.
val wantsRelease = gradle.startParameter.taskNames.any {
    it.contains("elease", ignoreCase = true)
}
val signingPropsPath = (System.getenv("TNP_SIGNING_PROPERTIES")
    ?: project.findProperty("TNP_SIGNING_PROPERTIES") as String?)
    ?: File(System.getProperty("user.home"), ".tnp-signing/signing.properties").absolutePath
val signingKeyKind = (System.getenv("TNP_KEY_KIND")
    ?: project.findProperty("TNP_KEY_KIND") as String?)

// Отпечаток боевого ключа (тема fix-release-keys, шаг 1; P1 аудита
// Sonnet-2 §1.4): ожидаемый SHA-256 ФАЙЛА keystore. Значение задаёт
// владелец (env TNP_SIGNING_KEY_SHA256 или -P), готовую команду сборки
// печатает configure; независимый от репозитория источник для сверки —
// docs/RELEASE-KEYS.md. Без отпечатка release-сборка не подтверждает,
// что APK подписан своим ключом, и отвергается.
val signingKeySha256 = (System.getenv("TNP_SIGNING_KEY_SHA256")
    ?: project.findProperty("TNP_SIGNING_KEY_SHA256") as String?)
    ?.trim()?.lowercase()

// Задачи, производящие release-артефакт (APK/AAB): на них навешивается
// verifySigning, а их присутствие в графе при выключенном signing-конфиге
// — отказ: неподписанный release-APK не должен собираться молча.
val releaseArtifactTasks = setOf(
    "packageRelease", "packageReleaseUniversalApk", "packageReleaseBundle",
    "makeApkFromBundleForRelease", "assembleRelease", "bundleRelease",
)

// sha256Hex — SHA-256 файла в hex нижнего регистра (как печатает
// configure и apksigner).
fun sha256Hex(file: File): String {
    val md = MessageDigest.getInstance("SHA-256")
    file.inputStream().use { input ->
        val buf = ByteArray(64 * 1024)
        while (true) {
            val n = input.read(buf)
            if (n < 0) break
            md.update(buf, 0, n)
        }
    }
    return md.digest().joinToString("") { "%02x".format(it) }
}

// permProblem — права файла ключей шире 0600 (только чтение и запись
// владельца): пароли лежат рядом с keystore открытым текстом, доступ
// группы, остальных и исполнение отвергаются. null — права в порядке
// или ФС без POSIX-прав.
fun permProblem(file: File): String? {
    if (!file.isFile) return null
    val perms = try {
        Files.getPosixFilePermissions(file.toPath())
    } catch (e: UnsupportedOperationException) {
        return null
    }
    val allowed = setOf(PosixFilePermission.OWNER_READ, PosixFilePermission.OWNER_WRITE)
    if ((perms - allowed).isEmpty()) return null
    return "${file.absolutePath}: ${PosixFilePermissions.toString(perms)}"
}

// verifySigning — проверка боевого ключа перед подписью (шаги 1–2, 4):
// SHA-256 файла ключа сверяется с TNP_SIGNING_KEY_SHA256, права файлов
// ключей — не шире 0600. Несовпадение отпечатка и широкие права —
// отказ; отсутствие отпечатка при release-сборке (wantsRelease) — отказ,
// при обычной — предупреждение. Release-артефакт без боевой подписи —
// отказ (не молчаливый неподписанный APK).
val gradleRef = gradle
val verifySigning by tasks.registering {
    group = "verification"
    description = "Сверка SHA-256 файла боевого ключа с TNP_SIGNING_KEY_SHA256 и прав файлов ключей (≤0600)"
    val propsPath = signingPropsPath
    val keyKind = signingKeyKind
    val expectedSha = signingKeySha256
    val releaseBuild = wantsRelease
    doLast {
        val releaseRequested = gradleRef.taskGraph.allTasks.map { it.name }
            .filter { it in releaseArtifactTasks }
        if (!releaseBuild && releaseRequested.isNotEmpty()) {
            throw GradleException(
                "release-сборка без боевой подписи: в графе задач ${releaseRequested.joinToString(", ")} " +
                    "при выключенном signing-конфиге (wantsRelease=false — в запрошенных задачах " +
                    "${gradleRef.startParameter.taskNames} нет подстроки «elease»). Неподписанный " +
                    "release-APK не собирается молча: отладочная сборка — assembleDebug / " +
                    "flutter build apk --debug; релиз — настройте ключи (cmd/tnp-testnet configure, " +
                    "docs/RELEASE-KEYS.md) и собирайте assembleRelease / flutter build apk --release."
            )
        }
        val propsFile = File(propsPath)
        if (!propsFile.isFile) {
            if (releaseBuild) {
                throw GradleException(
                    "release-сборка требует боевой signing-конфиг: $propsPath не найден. " +
                        "Сгенерируйте ключи и запустите cmd/tnp-testnet configure " +
                        "(см. docs/RELEASE-KEYS.md); путь — env TNP_SIGNING_PROPERTIES."
                )
            }
            logger.warn(
                "verifySigning: signing-конфиг $propsPath не найден — ключей нет, отпечаток не сверен " +
                    "(не release-сборка; release-сборку без ключей градл отвергает)"
            )
            return@doLast
        }
        if (keyKind == null) {
            val msg = "verifySigning: вид боевого ключа не задан (env TNP_KEY_KIND=std|rec) — " +
                "отпечаток не сверен"
            if (releaseBuild) throw GradleException("$msg; release-сборка требует явный вид ключа.")
            logger.warn("$msg (не release-сборка)")
            return@doLast
        }
        val props = Properties().apply { propsFile.inputStream().use { load(it) } }
        val storePath = props.getProperty("$keyKind.storeFile")
            ?: throw GradleException("signing-конфиг $propsPath: нет $keyKind.storeFile")
        val store = File(storePath)
        if (!store.isFile) {
            throw GradleException(
                "боевой keystore недоступен: ${store.absolutePath} (см. docs/RELEASE-KEYS.md)"
            )
        }
        val permProblems = listOf(propsFile, store, File(store.parentFile, "credentials.txt"))
            .mapNotNull { permProblem(it) }
        if (permProblems.isNotEmpty()) {
            throw GradleException(
                "права файлов ключей шире 0600 (только rw владельца) — пароли рядом с ключом " +
                    "читаемы группой/остальными:\n  " + permProblems.joinToString("\n  ") +
                    "\nнужен chmod 600 на эти файлы (docs/RELEASE-KEYS.md)."
            )
        }
        val actual = sha256Hex(store)
        when {
            expectedSha == null -> {
                if (releaseBuild) {
                    throw GradleException(
                        "release-сборка без ожидаемого отпечатка ключа: TNP_SIGNING_KEY_SHA256 не задан — " +
                            "сборка не подтверждает, что APK подписан своим ключом. Фактический " +
                            "SHA-256 файла ${store.name}: $actual. Отпечаток печатает " +
                            "cmd/tnp-testnet configure; сверьте с docs/RELEASE-KEYS.md."
                    )
                }
                logger.warn(
                    "verifySigning: TNP_SIGNING_KEY_SHA256 не задан — отпечаток ключа не сверен " +
                        "(не release-сборка). Фактический SHA-256 файла ${store.name}: $actual"
                )
            }
            actual != expectedSha -> throw GradleException(
                "отпечаток боевого ключа не совпал: ${store.absolutePath} → SHA-256 $actual, " +
                    "ожидался $expectedSha (TNP_SIGNING_KEY_SHA256). Подпись чужим ключом " +
                    "отвергнута (docs/RELEASE-KEYS.md)."
            )
            else -> logger.lifecycle(
                "verifySigning: отпечаток ключа ${store.name} совпал ($actual), " +
                    "права файлов ключей ≤0600"
            )
        }
    }
}

tasks.matching { it.name in releaseArtifactTasks }.configureEach {
    dependsOn(verifySigning)
}

// --- нативные строки Android из ARB (тема fix-native-strings, п.а+п.б) ---
// Строки, которые не проходят через вызов Dart, берутся штатным средством
// Android — ресурсами values[-<lang>]/strings.xml: уведомление
// foreground-сервиса узла (NodeService.kt) и label приложения (манифест).
// Единый источник переводов остаётся один — ARB (app/lib/l10n): задача
// generateNativeStrings пишет из них strings.xml при сборке, системная
// локаль Android сама выбирает каталог values-<lang>. Сгенерированные
// файлы не коммитятся (app/android/.gitignore) и пересоздаются на чистой
// сборке; вручную не правятся.
val nativeStringKeys = linkedMapOf(
    "appLabel" to "app_name",
    "nodeChannelName" to "node_channel_name",
    "nodeNotificationTitle" to "node_notification_title",
    "nodeNotificationText" to "node_notification_text",
    "errTxNotFound" to "err_tx_not_found",
    "signDefaultTitle" to "sign_default_title",
    "btnCancel" to "btn_cancel",
)
// en — шаблон ARB (fallback), кладётся в values/ — язык по умолчанию;
// остальные восемь — в values-<lang>/, их выбирает системная локаль
val nativeStringLocales = listOf("ru", "es", "pt", "zh", "fr", "de", "it", "ja")
val nativeArbDir = file("../../lib/l10n")
val nativeResDir = file("src/main/res")
val nativeArbFiles = fileTree(nativeArbDir) { include("app_*.arb") }

// xmlEscapeText — экранирование значения для strings.xml: служебные
// символы XML и обязательные для aapt апостроф и кавычка (без этого
// сборка ресурсов падает на французском «l'opération»)
fun xmlEscapeText(value: String): String = value
    .replace("&", "&amp;")
    .replace("<", "&lt;")
    .replace(">", "&gt;")
    .replace("'", "\\'")
    .replace("\"", "\\\"")

val generateNativeStrings by tasks.registering {
    group = "build"
    description = "Генерирует res/values[-<lang>]/strings.xml из ARB (тема fix-native-strings)"

    val outputFiles = listOf(File(nativeResDir, "values/strings.xml")) +
        nativeStringLocales.map { File(nativeResDir, "values-$it/strings.xml") }
    inputs.files(nativeArbFiles)
        .withPropertyName("arb")
        .withPathSensitivity(PathSensitivity.RELATIVE)
    outputs.files(outputFiles).withPropertyName("stringsXml")

    doLast {
        @Suppress("UNCHECKED_CAST")
        fun readArb(lang: String): Map<String, Any?> {
            val file = File(nativeArbDir, "app_$lang.arb")
            if (!file.isFile) throw GradleException("нет ARB: $file")
            return groovy.json.JsonSlurper().parse(file) as Map<String, Any?>
        }

        fun writeStrings(lang: String) {
            val data = readArb(lang)
            val dir = if (lang == "en") {
                File(nativeResDir, "values")
            } else {
                File(nativeResDir, "values-$lang")
            }
            val xml = StringBuilder()
            xml.append("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
            xml.append("<!-- Сгенерировано из app/lib/l10n/app_$lang.arb ")
            xml.append("(тема fix-native-strings). Вручную не править. -->\n")
            xml.append("<resources>\n")
            for ((arbKey, resName) in nativeStringKeys) {
                val value = data[arbKey] as? String
                    ?: throw GradleException("ARB app_$lang.arb: нет строкового ключа $arbKey")
                xml.append("    <string name=\"$resName\">${xmlEscapeText(value)}</string>\n")
            }
            xml.append("</resources>\n")
            dir.mkdirs()
            File(dir, "strings.xml").writeText(xml.toString(), Charsets.UTF_8)
        }

        writeStrings("en")
        for (lang in nativeStringLocales) writeStrings(lang)
        logger.lifecycle(
            "native strings: ${outputFiles.size} файлов из ARB " +
                "(values + ${nativeStringLocales.joinToString(", ")})"
        )
    }
}

// Задача идёт до обработки ресурсов: mergeResources должен видеть готовые
// strings.xml. Хук ленивый (matching/configureEach) — не зависит от того,
// объявлена ли preBuild раньше в этом скрипте.
tasks.matching { it.name == "preBuild" }.configureEach {
    dependsOn(generateNativeStrings)
}

android {
    namespace = "platform.tnp.tnp_app"
    compileSdk = 36
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        // Идентификация сети (этап 12, v9.10): applicationId на сеть
        // io.equality.<slug> (глобальная — io.equality.global) и label
        // Eq-<hash8> задаются при сборке APK конкретной сети
        // (-P TNP_APPLICATION_ID=... TNP_APP_LABEL=...); значения
        // генерирует конфигуратор (android.json). По умолчанию —
        // универсальная сборка конструктора.
        applicationId = (project.findProperty("TNP_APPLICATION_ID") as String?)
            ?: "platform.tnp.tnp_app"
        // Keystore-аттестация и BIOMETRIC_STRONG — с API 26 (этап 6)
        minSdk = 26
        targetSdk = 36
        versionCode = flutter.versionCode
        versionName = flutter.versionName
        // label приложения: Eq- + 8 hex SHA-256(network_id) (v9.10,
        // решение 7); в launcher — только Eq-хеш, display_name живёт
        // внутри приложения и всегда в паре с ним
        // по умолчанию — ресурс из ARB (values[-<lang>]/strings.xml,
        // задача generateNativeStrings, тема fix-native-strings): label
        // локализуется системной локалью; сетевая сборка по-прежнему
        // переопределяет его литералом через -P TNP_APP_LABEL
        manifestPlaceholders["appLabel"] =
            (project.findProperty("TNP_APP_LABEL") as String?) ?: "@string/app_name"
    }

    buildTypes {
        release {
            // боевая подпись (тема A8): signing-конфиг из локального файла
            // вне git; release без ключей — явный отказ, debug не затронут
            signingConfig = if (!wantsRelease) {
                null
            } else {
                val propsFile = File(signingPropsPath)
                if (!propsFile.isFile) {
                    throw GradleException(
                        "release-сборка требует боевой signing-конфиг: $signingPropsPath не найден. " +
                            "Сгенерируйте ключи и запустите cmd/tnp-testnet configure " +
                            "(см. docs/RELEASE-KEYS.md); путь — env TNP_SIGNING_PROPERTIES. " +
                            "Отпечаток ключа для сверки печатает configure (TNP_SIGNING_KEY_SHA256)."
                    )
                }
                if (signingKeyKind != "std" && signingKeyKind != "rec") {
                    throw GradleException(
                        "release-сборка требует вид боевого ключа: env TNP_KEY_KIND=std|rec " +
                            "(std — все std-сети, rec — rec-сети; вид печатает configure)."
                    )
                }
                val props = Properties().apply { propsFile.inputStream().use { load(it) } }
                fun req(name: String): String = props.getProperty("$signingKeyKind.$name")
                    ?: throw GradleException(
                        "signing-конфиг $signingPropsPath: нет $signingKeyKind.$name"
                    )
                val store = File(req("storeFile"))
                if (!store.isFile) {
                    throw GradleException(
                        "боевой keystore недоступен: ${store.absolutePath} (см. docs/RELEASE-KEYS.md)"
                    )
                }
                signingConfigs.create("release-$signingKeyKind") {
                    this.storeFile = store
                    storePassword = req("storePassword")
                    keyAlias = req("keyAlias")
                    keyPassword = req("keyPassword")
                }
            }
            // R8-минификация (этап 8, задача 2): tnp.** — JNI-связка AAR
            // (Go вызывает методы Java через JNI — обфускация ломает связку)
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

// общий платформенный слой этапа 6 компилируется прямо в хост
// (без дублирования); экран этапа 6 (ui/MainActivity.kt) исключён —
// его заменил Flutter-UI, файл ссылается на R этапа 6.
// AGP 9 (newDsl): источник добавляется в Kotlin-задачу напрямую.
tasks.withType<org.jetbrains.kotlin.gradle.tasks.KotlinCompile>().configureEach {
    source(
        fileTree("${rootProject.projectDir}/../../android/app/src/main/kotlin") {
            exclude("platform/tnp/android/ui/**")
        },
    )
}

flutter {
    source = "../.."
}

dependencies {
    // ядро (Go, gomobile bind) — собирается: gomobile bind ... -o android/libs/tnp.aar ./mobile
    implementation(files("${rootProject.projectDir}/../../android/libs/tnp.aar"))
    // платформенный слой
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.biometric:biometric:1.4.0-alpha07")
}
