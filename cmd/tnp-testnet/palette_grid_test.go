// Сетка палитры Equality для утверждения владельцем (мини-этап 12.1):
// 16 квадратов с подписями (имя тона, hex, индекс) на светлом и тёмном
// фоне. Пишется тестом TestPaletteGrid при выставленном EQ_PALETTE_GRID,
// палитра — единственный источник (equalityPalette из android.go).
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// gridFont — растровый шрифт 5×7 для подписей сетки (верхний регистр
// латиницы, цифры, «#», «-», пробел); шрифтов в stdlib нет, зависимости
// ради одной артефакт-картинки не вводим.
var gridFont = map[rune]string{
	' ': "...../...../...../...../...../...../.....",
	'-': "...../...../...../#####/...../...../.....",
	'#': "...../.#.#./#####/.#.#./#####/.#.#./.....",
	'0': ".###./#...#/#..##/#.#.#/##..#/#...#/.###.",
	'1': "..#../.##../..#../..#../..#../..#../.###.",
	'2': ".###./#...#/....#/...#./..#../.#.../#####",
	'3': ".###./#...#/....#/..##./....#/#...#/.###.",
	'4': "...#./..##./.#.#./#..#./#####/...#./...#.",
	'5': "#####/#..../####./....#/....#/#...#/.###.",
	'6': "..##./.#.../#..../####./#...#/#...#/.###.",
	'7': "#####/....#/...#./..#../.#.../.#.../.#...",
	'8': ".###./#...#/#...#/.###./#...#/#...#/.###.",
	'9': ".###./#...#/#...#/.####/....#/...#./.##..",
	'A': ".###./#...#/#...#/#####/#...#/#...#/#...#",
	'B': "####./#...#/#...#/####./#...#/#...#/####.",
	'C': ".###./#...#/#..../#..../#..../#...#/.###.",
	'D': "####./#...#/#...#/#...#/#...#/#...#/####.",
	'E': "#####/#..../#..../####./#..../#..../#####",
	'F': "#####/#..../#..../####./#..../#..../#....",
	'G': ".###./#...#/#..../#..##/#...#/#...#/.###.",
	'H': "#...#/#...#/#...#/#####/#...#/#...#/#...#",
	'I': ".###./..#../..#../..#../..#../..#../.###.",
	'J': "..###/...#./...#./...#./...#./#..#./.##..",
	'K': "#...#/#..#./#.#../##.../#.#../#..#./#...#",
	'L': "#..../#..../#..../#..../#..../#..../#####",
	'M': "#...#/##.##/#.#.#/#.#.#/#...#/#...#/#...#",
	'N': "#...#/##..#/#.#.#/#..##/#...#/#...#/#...#",
	'O': ".###./#...#/#...#/#...#/#...#/#...#/.###.",
	'P': "####./#...#/#...#/####./#..../#..../#....",
	'Q': ".###./#...#/#...#/#...#/#.#.#/#..##/.####",
	'R': "####./#...#/#...#/####./#.#../#..#./#...#",
	'S': ".####/#..../#..../.###./....#/....#/####.",
	'T': "#####/..#../..#../..#../..#../..#../..#..",
	'U': "#...#/#...#/#...#/#...#/#...#/#...#/.###.",
	'V': "#...#/#...#/#...#/#...#/#...#/.#.#./..#..",
	'W': "#...#/#...#/#...#/#.#.#/#.#.#/##.##/#...#",
	'X': "#...#/#...#/.#.#./..#../.#.#./#...#/#...#",
	'Y': "#...#/#...#/.#.#./..#../..#../..#../..#..",
	'Z': "#####/....#/...#./..#../.#.../#..../#####",
}

// drawText — вывод строки шрифтом 5×7 (шаг 6 px); незнакомый глиф — ошибка
// (сетка обязана подписываться целиком, молчаливые пропуски недопустимы).
func drawText(img *image.RGBA, x, y int, s string, c color.RGBA) error {
	for _, r := range s {
		g, ok := gridFont[r]
		if !ok {
			return fmt.Errorf("глиф %q не задан в шрифте сетки", r)
		}
		for ry, row := range strings.Split(g, "/") {
			for rx := 0; rx < len(row); rx++ {
				if row[rx] == '#' {
					img.SetRGBA(x+rx, y+ry, c)
				}
			}
		}
		x += 6
	}
	return nil
}

// writePaletteGrid — сетка 16 тонов палитры: две панели (светлый и тёмный
// фон), 4×4 квадрата, под каждым подписи: имя тона, hex, индекс.
func writePaletteGrid(path string) error {
	const (
		margin = 20
		cols   = 4
		cellW  = 120
		square = 80
		lineH  = 10
		cellH  = square + 8 + 3*lineH + 10
	)
	panelH := margin + 14 + 4*cellH + margin
	width := margin + cols*cellW + margin
	img := image.NewRGBA(image.Rect(0, 0, width, 2*panelH))
	light, err := parseHexColor("#FFFFFF")
	if err != nil {
		return err
	}
	dark, err := parseHexColor("#121212")
	if err != nil {
		return err
	}
	textLight, err := parseHexColor("#212121")
	if err != nil {
		return err
	}
	textDark, err := parseHexColor("#FFFFFF")
	if err != nil {
		return err
	}
	for y := 0; y < 2*panelH; y++ {
		for x := 0; x < width; x++ {
			if y < panelH {
				img.SetRGBA(x, y, light)
			} else {
				img.SetRGBA(x, y, dark)
			}
		}
	}
	for pi := 0; pi < 2; pi++ {
		y0 := pi * panelH
		txt, cap := textLight, "LIGHT"
		if pi == 1 {
			txt, cap = textDark, "DARK"
		}
		if err := drawText(img, margin, y0+8, cap, txt); err != nil {
			return err
		}
		for i, tone := range equalityPalette {
			x := margin + (i%cols)*cellW
			y := y0 + margin + 14 + (i/cols)*cellH
			c, err := parseHexColor(tone.Hex)
			if err != nil {
				return err
			}
			for sy := 0; sy < square; sy++ {
				for sx := 0; sx < square; sx++ {
					img.SetRGBA(x+sx, y+sy, c)
				}
			}
			ly := y + square + 8
			for _, line := range []string{
				strings.ToUpper(tone.Name),
				tone.Hex,
				strconv.Itoa(i),
			} {
				if err := drawText(img, x, ly, line, txt); err != nil {
					return err
				}
				ly += lineH
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writePNG(path, img)
}

// TestPaletteGrid — генерация сетки палитры для утверждения владельцем:
// EQ_PALETTE_GRID=reports/stage12-1/palette.png \
// go test ./cmd/tnp-testnet/ -run TestPaletteGrid
func TestPaletteGrid(t *testing.T) {
	path := os.Getenv("EQ_PALETTE_GRID")
	if path == "" {
		t.Skip("сетка пишется только при EQ_PALETTE_GRID=<путь>")
	}
	if err := writePaletteGrid(path); err != nil {
		t.Fatalf("сетка палитры: %v", err)
	}
	t.Logf("сетка палитры: %s", path)
}
