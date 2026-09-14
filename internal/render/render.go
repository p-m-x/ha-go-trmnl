// Package render produces 800×480 monochrome BMP images for TRMNL e-ink displays.
package render

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/przemek/ha-go-trmnl/internal/ha"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	displayW = 800
	displayH = 480
)

// Page holds preprocessed data ready to be drawn.
type Page struct {
	Title     string
	UpdatedAt time.Time
	Entities  []*ha.State
	Columns   int
}

// RenderBMP renders a Page to a 1-bit BMP suitable for TRMNL firmware.
func RenderBMP(p *Page) ([]byte, error) {
	img := renderRGBA(p)
	return encodeBMP(img), nil
}

// RenderPNG renders a Page to a PNG (useful for previewing).
func RenderPNG(p *Page) ([]byte, error) {
	img := renderRGBA(p)
	var buf bytes.Buffer
	dc := gg.NewContextForRGBA(img)
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderRGBA draws the page onto an 800×480 RGBA canvas.
func renderRGBA(p *Page) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, displayW, displayH))
	dc := gg.NewContextForRGBA(img)

	// Background: white
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	dc.SetRGB(0, 0, 0)

	const (
		margin  = 24.0
		headerH = 56.0
		footerH = 32.0
		cardPad = 12.0
	)

	// ── Header ───────────────────────────────────────────────────────────────
	drawText(img, p.Title, displayW/2, int(margin)+14, color.Black, true)

	// Separator below header
	drawHLine(img, int(margin), displayW-int(margin), int(headerH), color.Black)

	// ── Footer ───────────────────────────────────────────────────────────────
	ts := "Updated " + p.UpdatedAt.Format("15:04:05")
	drawText(img, ts, displayW-int(margin)-len(ts)*7, displayH-int(footerH)+16, color.Black, false)
	drawHLine(img, int(margin), displayW-int(margin), displayH-int(footerH), color.Black)

	// ── Entity grid ──────────────────────────────────────────────────────────
	cols := p.Columns
	if cols < 1 {
		cols = 1
	}
	if cols > 2 {
		cols = 2
	}

	areaTop := headerH + cardPad
	areaBot := float64(displayH) - footerH - cardPad
	areaH := areaBot - areaTop
	areaW := float64(displayW) - 2*margin

	rows := int(math.Ceil(float64(len(p.Entities)) / float64(cols)))
	if rows == 0 {
		rows = 1
	}
	cellW := areaW / float64(cols)
	cellH := areaH / float64(rows)

	for i, e := range p.Entities {
		col := i % cols
		row := i / cols
		x := int(margin + float64(col)*cellW)
		y := int(areaTop + float64(row)*cellH)
		w := int(cellW)
		h := int(cellH)

		drawCard(img, e, x, y, w, h, int(cardPad))
	}

	return img
}

// drawCard renders a single entity card inside the given cell.
func drawCard(img *image.RGBA, e *ha.State, x, y, w, h, pad int) {
	// Card border
	drawRect(img, x+pad/2, y+pad/2, w-pad, h-pad, color.Black)

	cx := x + w/2

	// Domain icon
	icon := domainIcon(e.Domain())
	drawText(img, icon, cx-len(icon)*4, y+pad/2+14, color.Black, false)

	// Friendly name (smaller text, centered)
	name := truncate(e.FriendlyName(), 22)
	nameX := cx - len(name)*4
	drawText(img, name, nameX, y+h/2-4, color.Black, false)

	// State + unit (larger, bold-simulated by drawing twice slightly offset)
	stateStr := formatState(e)
	stateX := cx - len(stateStr)*5
	drawText(img, stateStr, stateX+1, y+h/2+22, color.Black, false)
	drawText(img, stateStr, stateX, y+h/2+22, color.Black, false)
}

// domainIcon maps HA domains to a printable symbol.
func domainIcon(domain string) string {
	icons := map[string]string{
		"sensor":              "[SNS]",
		"binary_sensor":       "[BIN]",
		"switch":              "[SWT]",
		"light":               "[LGT]",
		"climate":             "[CLM]",
		"media_player":        "[MED]",
		"cover":               "[CVR]",
		"alarm_control_panel": "[ALM]",
		"person":              "[PRS]",
		"weather":             "[WTH]",
		"input_boolean":       "[IBL]",
		"input_number":        "[INU]",
		"input_select":        "[ISL]",
		"automation":          "[AUT]",
		"script":              "[SCR]",
		"device_tracker":      "[TRK]",
	}
	if icon, ok := icons[domain]; ok {
		return icon
	}
	return "[???]"
}

// formatState returns a display-friendly state string with optional unit.
func formatState(e *ha.State) string {
	state := e.State
	unit := e.Unit()
	if unit != "" {
		return state + " " + unit
	}
	switch strings.ToLower(state) {
	case "on":
		return "ON"
	case "off":
		return "OFF"
	case "unavailable":
		return "N/A"
	case "unknown":
		return "?"
	}
	return truncate(state, 16)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "..."
}

// ── Primitive drawing helpers ─────────────────────────────────────────────────

func drawText(img *image.RGBA, s string, x, y int, c color.Color, _ bool) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func drawHLine(img *image.RGBA, x1, x2, y int, c color.Color) {
	for x := x1; x <= x2; x++ {
		img.Set(x, y, c)
	}
}

func drawVLine(img *image.RGBA, x, y1, y2 int, c color.Color) {
	for y := y1; y <= y2; y++ {
		img.Set(x, y, c)
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	drawHLine(img, x, x+w, y, c)
	drawHLine(img, x, x+w, y+h, c)
	drawVLine(img, x, y, y+h, c)
	drawVLine(img, x+w, y, y+h, c)
}

// ── BMP encoding ─────────────────────────────────────────────────────────────
// Encodes an RGBA image to a 1-bit monochrome BMP (TRMNL firmware expects BMP).

func encodeBMP(img *image.RGBA) []byte {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	// Row stride for 1-bit BMP must be padded to 4 bytes.
	rowBytes := (w + 7) / 8
	stride := (rowBytes + 3) &^ 3
	pixelDataSize := stride * h

	fileSize := 62 + pixelDataSize // 14 (file hdr) + 40 (info hdr) + 8 (palette) + pixels
	buf := make([]byte, fileSize)
	p := 0

	// File header
	buf[0], buf[1] = 'B', 'M'
	p += 2
	binary.LittleEndian.PutUint32(buf[p:], uint32(fileSize))
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], 0) // reserved
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], 62) // pixel data offset
	p += 4

	// BITMAPINFOHEADER (40 bytes)
	binary.LittleEndian.PutUint32(buf[p:], 40) // header size
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], uint32(w))
	p += 4
	// Negative height = top-down bitmap (stored as two's complement uint32)
	binary.LittleEndian.PutUint32(buf[p:], uint32(-int32(h)))
	p += 4
	binary.LittleEndian.PutUint16(buf[p:], 1) // planes
	p += 2
	binary.LittleEndian.PutUint16(buf[p:], 1) // bits per pixel
	p += 2
	binary.LittleEndian.PutUint32(buf[p:], 0) // compression: none
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], uint32(pixelDataSize))
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], 2835) // X pixels/meter (~72 DPI)
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], 2835)
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], 2) // colors in table
	p += 4
	binary.LittleEndian.PutUint32(buf[p:], 0) // important colors
	p += 4

	// Color table: index 0 = black, index 1 = white (RGBQUAD: B G R reserved)
	copy(buf[p:], []byte{0x00, 0x00, 0x00, 0x00}) // black
	p += 4
	copy(buf[p:], []byte{0xFF, 0xFF, 0xFF, 0x00}) // white
	p += 4

	// Pixel data
	row := make([]byte, stride)
	for y := 0; y < h; y++ {
		for i := range row {
			row[i] = 0
		}
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x, y)
			luma := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
			if luma > 127 {
				// white → bit = 1
				row[x/8] |= 0x80 >> (x % 8)
			}
		}
		copy(buf[p:], row)
		p += stride
	}

	return buf
}
