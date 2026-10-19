# AAR ядра (gomobile bind, этап 6): состав и размеры

Сборка: `gomobile bind -target=android -androidapi 26 -ldflags "-checklinkname=0" -o android/libs/tnp.aar ./mobile`
(Go 1.27.2, NDK 27.2). Флаг -checklinkname=0 обязателен: зависимость
wlynxg/anet (транспорт WebRTC в libp2p, только Android) использует
go:linkname на net.zoneCache; Go 1.23+ запрещает такие ссылки без
этого флага.

| Компонент AAR | Размер (несжато) |
| --- | --- |
| jni/arm64-v8a/libgojni.so | 51,1 МБ |
| jni/armeabi-v7a/libgojni.so | 66,2 МБ |
| jni/x86/libgojni.so | 66,6 МБ |
| jni/x86_64/libgojni.so | 54,1 МБ |
| classes.jar (JNI-обёртки) | 18 КБ |
| AAR целиком | 103,6 МБ |

Замечания:
- .so содержит всё ядро + libp2p (полный сетевой стек). Сжатие в APK/AAB
  уменьшает передаваемый размер примерно вдвое; Play-доставка по ABI
  отдаёт устройству только его .so (~51 МБ для arm64).
- armeabi-v7a/x86 можно исключить (abiFilters arm64-v8a) — почти все
  актуальные устройства arm64.
