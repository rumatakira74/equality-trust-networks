# R8 (этап 8, задача 2): JNI-связка gomobile AAR — Go-код вызывает методы
# классов tnp.** через JNI; переименование/удаление ломает связку.
-keep class tnp.** { *; }
-dontwarn tnp.**

# платформенный слой вызывается из Flutter по reflection-каналу и из Go (Logger)
-keep class platform.tnp.android.** { *; }
