//go:build android

// Зависимость от golang.org/x/mobile/bind (сборка AAR через gomobile bind;
// импорт держит модуль в go.mod — go mod tidy его не вычищает).
package tnp

import _ "golang.org/x/mobile/bind"
