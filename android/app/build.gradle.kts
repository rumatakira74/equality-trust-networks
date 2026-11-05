// Приложение платформы сетей доверия (этап 6): Keystore-аттестация,
// биометрия, PIN (8.2), узел full/light (12.2) через AAR ядра.
plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "platform.tnp.android"
    compileSdk = 36

    defaultConfig {
        applicationId = "platform.tnp.android"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.6.0"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }
}

// --- нативные строки Android из ARB (тема fix-sweep, FINDING-fix-native-strings-01 п.1) ---
// Тот же механизм, что во Flutter-хосте (app/android/app/build.gradle.kts):
// строки, которые не проходят через вызов Dart — уведомление
// foreground-сервиса узла (NodeService.kt ищет ресурс по имени) и label
// приложения (манифест) — берутся из ресурсов values[-<lang>]/strings.xml.
// Единый источник переводов — ARB (app/lib/l10n): задача
// generateNativeStrings пишет из них strings.xml при сборке, системная
// локаль Android сама выбирает каталог values-<lang>. Карта ключей
// совпадает с Flutter-хостом (проверяется тестом native_strings_test.dart).
// Сгенерированные файлы не коммитятся (android/app/.gitignore) и
// пересоздаются на чистой сборке; вручную не правятся. Собственные строки
// UI этапа 6 (кнопки, PIN-заголовки) остаются в res/values/host_ui.xml.
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
val nativeArbDir = file("../../app/lib/l10n")
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
    description = "Генерирует res/values[-<lang>]/strings.xml из ARB (тема fix-sweep)"

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
            xml.append("(тема fix-sweep). Вручную не править. -->\n")
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

dependencies {
    implementation(files("../libs/tnp.aar"))
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation("androidx.biometric:biometric:1.4.0-alpha07")
    implementation("androidx.constraintlayout:constraintlayout:2.1.4")

    testImplementation("junit:junit:4.13.2")
    androidTestImplementation("androidx.test.ext:junit:1.2.1")
    androidTestImplementation("androidx.test:runner:1.6.2")
    androidTestImplementation("androidx.test:rules:1.6.1")
}
