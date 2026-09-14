package render

import (
	"testing"
	"time"

	"github.com/przemek/ha-go-trmnl/internal/ha"
)

func TestRenderBMP(t *testing.T) {
	page := &Page{
		Title:     "Test HA",
		UpdatedAt: time.Now(),
		Columns:   2,
		Entities: []*ha.State{
			{EntityID: "sensor.temperature", State: "22.5", Attributes: map[string]any{"unit_of_measurement": "°C", "friendly_name": "Temperature"}},
			{EntityID: "light.living_room", State: "on", Attributes: map[string]any{"friendly_name": "Living Room Light"}},
			{EntityID: "switch.fan", State: "off", Attributes: map[string]any{"friendly_name": "Bedroom Fan"}},
		},
	}

	bmp, err := RenderBMP(page)
	if err != nil {
		t.Fatal(err)
	}
	if len(bmp) == 0 {
		t.Fatal("empty BMP output")
	}
	// BMP magic bytes
	if bmp[0] != 'B' || bmp[1] != 'M' {
		t.Errorf("expected BM header, got %q%q", bmp[0], bmp[1])
	}
}

func TestRenderPNG(t *testing.T) {
	page := &Page{
		Title:     "Test",
		UpdatedAt: time.Now(),
		Columns:   1,
		Entities: []*ha.State{
			{EntityID: "sensor.temp", State: "21.0"},
		},
	}
	png, err := RenderPNG(page)
	if err != nil {
		t.Fatal(err)
	}
	// PNG magic: \x89PNG
	if len(png) < 4 || png[1] != 'P' || png[2] != 'N' || png[3] != 'G' {
		t.Error("output is not a valid PNG")
	}
}
