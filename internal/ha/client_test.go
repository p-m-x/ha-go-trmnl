package ha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(State{
			EntityID: "sensor.temperature",
			State:    "22.5",
			Attributes: map[string]any{
				"unit_of_measurement": "°C",
				"friendly_name":       "Living Room Temperature",
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	s, err := c.GetState(context.Background(), "sensor.temperature")
	if err != nil {
		t.Fatal(err)
	}
	if s.State != "22.5" {
		t.Errorf("state = %q, want 22.5", s.State)
	}
	if s.Unit() != "°C" {
		t.Errorf("unit = %q, want °C", s.Unit())
	}
	if s.FriendlyName() != "Living Room Temperature" {
		t.Errorf("friendly name = %q", s.FriendlyName())
	}
}

func TestFriendlyNameFallback(t *testing.T) {
	s := &State{EntityID: "sensor.living_room_temp"}
	if got := s.FriendlyName(); got != "Living Room Temp" {
		t.Errorf("FriendlyName() = %q", got)
	}
}
