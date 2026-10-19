// Flutter-хост приложения (этап 7): только Android. Платформенный слой
// этапа 6 (android/app/src/main/kotlin) компилируется сюда напрямую
// (srcDir, без дублирования); AAR ядра — gomobile bind (android/libs).
// Экран этапа 6 (ui/MainActivity.kt) исключён — его заменил Flutter-UI.
plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
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
        applicationId = "platform.tnp.tnp_app"
        // Keystore-аттестация и BIOMETRIC_STRONG — с API 26 (этап 6)
        minSdk = 26
        targetSdk = 36
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    sourceSets {
        getByName("main") {
            // общий платформенный слой (Keystore, PIN, узел, BT, NSD)
            java.srcDir("../../android/app/src/main/kotlin")
            // экран этапа 6 не входит в Flutter-приложение
            java.exclude("platform/tnp/android/ui/MainActivity.kt")
        }
    }

    buildTypes {
        release {
            // подпись debug — релизная подпись отдельно (этап 8)
            signingConfig = signingConfigs.getByName("debug")
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}

dependencies {
    // ядро (Go, gomobile bind) — собирается: gomobile bind ... -o android/libs/tnp.aar ./mobile
    implementation(files("../../android/libs/tnp.aar"))
    // платформенный слой
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.biometric:biometric:1.1.0")
}
