// Flutter-хост приложения (этап 7): только Android. Платформенный слой
// этапа 6 (android/app/src/main/kotlin) компилируется сюда напрямую
// (srcDir, без дублирования); AAR ядра — gomobile bind (android/libs).
// Экран этапа 6 (ui/MainActivity.kt) исключён — его заменил Flutter-UI.
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
// debug-сборки не затронуты.
val wantsRelease = gradle.startParameter.taskNames.any {
    it.contains("elease", ignoreCase = true)
}
val signingPropsPath = (System.getenv("TNP_SIGNING_PROPERTIES")
    ?: project.findProperty("TNP_SIGNING_PROPERTIES") as String?)
    ?: File(System.getProperty("user.home"), ".tnp-signing/signing.properties").absolutePath
val signingKeyKind = (System.getenv("TNP_KEY_KIND")
    ?: project.findProperty("TNP_KEY_KIND") as String?)

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
        manifestPlaceholders["appLabel"] =
            (project.findProperty("TNP_APP_LABEL") as String?) ?: "Сети доверия"
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
                            "(см. docs/RELEASE-KEYS.md); путь — env TNP_SIGNING_PROPERTIES."
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
