// Артефакты Android-сборки сети (этап 12, шаг 3): имя APK, applicationId,
// label Eq-<hash8>, display_name и adaptive-иконка (вариант 3 — гибрид).
// Генерируются конфигуратором автоматически из параметров сети; имя —
// эргономика и лицензионная прозрачность (recurring = on → суффикс rec),
// не идентичность: истинная изоляция — network_id в genesis (v9.10).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// equalityPalette — палитра Equality (~16 цветов; v9.10, решение 9):
// background иконки — детерминированный цвет из SHA-256(network_id),
// одна сеть — один цвет всегда. Состав палитры — черновой (точные тона
// владельцем не заданы; помечено в отчёте этапа 12).
var equalityPalette = [16]string{
	"#E63946", "#F4A261", "#E9C46A", "#2A9D8F",
	"#264653", "#457B9D", "#A8DADC", "#6D597A",
	"#B5838D", "#8E9AAF", "#BC6C25", "#606C38",
	"#283618", "#4A5759", "#7F5539", "#0B4F6C",
}

// validateSlug — slug: строчные латинские буквы, цифры и дефисы
// (человекочитаемая метка сети; входит в имя APK и applicationId).
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug пуст (для не-глобальной сети обязателен)")
	}
	if len(slug) > 32 {
		return fmt.Errorf("slug длиннее 32 символов: %q", slug)
	}
	for _, r := range slug {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
		if !ok {
			return fmt.Errorf("slug %q: допустимы строчные латинские буквы, цифры и дефисы", slug)
		}
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return fmt.Errorf("slug %q: не начинается и не заканчивается дефисом", slug)
	}
	return nil
}

// APKName — имя файла APK (решение 5): equality-{global|slug}-{std|rec}-v{версия}.apk.
func APKName(slug string, recurring bool, version string) string {
	kind := "global"
	if !wire.IsGlobalNetwork([]byte(slug)) {
		kind = slug
	}
	build := "std"
	if recurring {
		build = "rec" // лицензионная прозрачность: rec — коммерческая сборка
	}
	return fmt.Sprintf("equality-%s-%s-v%s.apk", kind, build, version)
}

// ApplicationID — applicationId на сеть (решение 6): io.equality.<slug>,
// глобальная — io.equality.global. Одинаковый applicationId у двух сетей
// недопустим (Android перезапишет чужое приложение вместе с данными
// кошелька); коллизия при равных slug у разных организаторов — PENDING.
func ApplicationID(slug string) string {
	if wire.IsGlobalNetwork([]byte(slug)) || slug == "global" {
		return "io.equality.global"
	}
	return "io.equality." + slug
}

// IconSpec — параметры adaptive-иконки (вариант 3 — гибрид):
// background — детерминированный цвет из SHA-256(network_id) по палитре
// Equality; foreground — логотип организатора (если задан) или
// автогенерированный идентикон из network_id; rec-сборки — бейдж в углу.
type IconSpec struct {
	Background   string // "#RRGGBB" из палитры
	PaletteIndex int
	Foreground   string // "logo" | "identicon"
	Badge        bool   // rec-сборка — бейдж в углу foreground
}

// iconSpec — детерминированные параметры иконки из network_id.
func iconSpec(networkID []byte, logoGiven bool, recurring bool) IconSpec {
	h := sha256.Sum256(networkID)
	idx := int(h[0]) % len(equalityPalette)
	fg := "identicon"
	if logoGiven {
		fg = "logo"
	}
	return IconSpec{
		Background:   equalityPalette[idx],
		PaletteIndex: idx,
		Foreground:   fg,
		Badge:        recurring,
	}
}

// writeIconAssets — генерация PNG-артефактов adaptive-иконки в каталог
// <out>/icon: background (сплошной цвет), foreground (идентикон из
// network_id или копия логотипа организатора), бейдж rec-сборки рисуется
// в углу foreground. Размер 432×432 (108dp × 4).
func writeIconAssets(dir string, networkID []byte, spec IconSpec, logoPath string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	bg, err := parseHexColor(spec.Background)
	if err != nil {
		return err
	}
	// background — сплошной цвет
	if err := writePNG(filepath.Join(dir, "background.png"), solidImage(bg)); err != nil {
		return err
	}
	// foreground: логотип организатора (копия) или идентикон
	if spec.Foreground == "logo" && logoPath != "" {
		return copyFile(logoPath, filepath.Join(dir, "foreground.png"))
	}
	if err := writePNG(filepath.Join(dir, "foreground.png"),
		identiconImage(networkID, bg, spec.Badge)); err != nil {
		return err
	}
	return nil
}

// identiconImage — автогенерированный идентикон из network_id: сетка 5×5,
// зеркальная по вертикали (первые 3 колонки из хеша), блоки цвета палитры
// на прозрачном фоне внутри безопасной зоны; rec-сборка — бейдж в углу.
func identiconImage(networkID []byte, c color.RGBA, badge bool) *image.RGBA {
	const size = 432
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	h := sha256.Sum256(networkID)
	// сетка 5×5, ячейка 48 px — 240 px в центре (безопасная зона ~66%)
	const cell = 48
	off := (size - cell*5) / 2
	bit := 0
	takeBit := func() bool {
		b := (h[bit/8]>>(uint(bit)%8))&1 == 1
		bit++
		if bit >= 256 { // хеш кончился — повторяем сначала
			bit = 0
		}
		return b
	}
	for row := 0; row < 5; row++ {
		cols := [5]bool{}
		cols[0] = takeBit()
		cols[1] = takeBit()
		cols[2] = takeBit()
		cols[3] = cols[1] // зеркальная симметрия
		cols[4] = cols[0]
		for col := 0; col < 5; col++ {
			if !cols[col] {
				continue
			}
			x0 := off + col*cell
			y0 := off + row*cell
			for y := y0; y < y0+cell; y++ {
				for x := x0; x < x0+cell; x++ {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
	if badge {
		// бейдж rec-сборки: круг в углу foreground (акцент поверх палитры)
		const r = 40
		cx, cy := size-72, size-72
		badgeCol := color.RGBA{R: 230, G: 57, B: 70, A: 255} // #E63946
		for y := cy - r; y <= cy+r; y++ {
			for x := cx - r; x <= cx+r; x++ {
				dx, dy := x-cx, y-cy
				if dx*dx+dy*dy <= r*r {
					img.SetRGBA(x, y, badgeCol)
				}
			}
		}
		// белое кольцо — бейдж читается поверх любого фона
		for y := cy - r - 6; y <= cy+r+6; y++ {
			for x := cx - r - 6; x <= cx+r+6; x++ {
				dx, dy := x-cx, y-cy
				d := dx*dx + dy*dy
				if d <= (r+6)*(r+6) && d > r*r {
					img.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				}
			}
		}
	}
	return img
}

func solidImage(c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 432, 432))
	for y := 0; y < 432; y++ {
		for x := 0; x < 432; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("цвет %q: нужен #RRGGBB", s)
	}
	v, err := hex.DecodeString(s)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{R: v[0], G: v[1], B: v[2], A: 255}, nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
