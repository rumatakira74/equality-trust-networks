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
	"regexp"
	"strings"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// paletteTone — тон палитры Equality: имя (для сетки утверждения) и hex.
type paletteTone struct {
	Name string // имя Material-тона (для подписи на сетке)
	Hex  string // "#RRGGBB"
}

// equalityPalette — палитра Equality (v9.10, решение 9; решение владельца
// 01.11.2026): 16 тонов Material уровня 500–600 без белого/чёрного/серых.
// background иконки — детерминированный цвет из SHA-256(network_id),
// одна сеть — один цвет всегда. Светлые тона (light blue, cyan,
// light green, lime, yellow, amber, orange) взяты уровнем 600 —
// контраст идентикона и подписей; остальные — 500. Сетка для утверждения
// владельцем — reports/stage12-1/palette.png (генерация — тест
// TestPaletteGrid, флаг EQ_PALETTE_GRID).
var equalityPalette = [16]paletteTone{
	{Name: "red", Hex: "#F44336"},         // Material red 500
	{Name: "pink", Hex: "#E91E63"},        // Material pink 500
	{Name: "purple", Hex: "#9C27B0"},     // Material purple 500
	{Name: "deep purple", Hex: "#673AB7"}, // Material deep purple 500
	{Name: "indigo", Hex: "#3F51B5"},      // Material indigo 500
	{Name: "blue", Hex: "#2196F3"},        // Material blue 500
	{Name: "light blue", Hex: "#039BE5"},  // Material light blue 600
	{Name: "cyan", Hex: "#00ACC1"},        // Material cyan 600
	{Name: "teal", Hex: "#009688"},        // Material teal 500
	{Name: "green", Hex: "#4CAF50"},       // Material green 500
	{Name: "light green", Hex: "#7CB342"}, // Material light green 600
	{Name: "lime", Hex: "#AFB42B"},        // Material lime 600
	{Name: "yellow", Hex: "#FBC02D"},     // Material yellow 600
	{Name: "amber", Hex: "#FFB300"},       // Material amber 600
	{Name: "orange", Hex: "#FB8C00"},     // Material orange 600
	{Name: "brown", Hex: "#795548"},       // Material brown 500
}

// slugRe — slug сети (решение владельца 01.11.2026): строчная латиница,
// цифры и дефисы, 3–24 символа, начинается со строчной буквы.
var slugRe = regexp.MustCompile(`^[a-z][a-z0-9-]{2,23}$`)

// validateSlug — slug: человекочитаемая метка сети; входит в имя APK
// и applicationId (санитизацией дефисов), поэтому алфавит ограничен.
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug пуст (для не-глобальной сети обязателен)")
	}
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("slug %q: допустимы строчная латиница, цифры и дефисы; 3–24 символа, первый — строчная буква (^[a-z][a-z0-9-]{2,23}$)", slug)
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

// ApplicationID — applicationId на сеть (решение владельца 01.11.2026,
// мини-этап 12.1): io.equality.<sanitized-slug>.<hash8>, глобальная —
// io.equality.global.<hash8>. sanitized-slug — slug с дефисами,
// заменёнными на «_» (дефисы в Android-пакете запрещены, расхождение №7
// отчёта этапа 12); <hash8> — первые 8 hex SHA-256(network_id), тот же
// хеш, что в label Eq-<hash8>. Уникальность — целиком на хеше:
// network_id уникален по построению (v9.10), равные slug разных
// организаторов больше не коллизируют (PENDING №2/№3 этапа 12 закрыты).
func ApplicationID(networkID []byte, slug string) string {
	h := sha256.Sum256(networkID)
	hash8 := hex.EncodeToString(h[:4])
	if wire.IsGlobalNetwork(networkID) || slug == "global" {
		return "io.equality.global." + hash8
	}
	return "io.equality." + strings.ReplaceAll(slug, "-", "_") + "." + hash8
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
		Background:   equalityPalette[idx].Hex,
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
