// Package api implements the TRMNL device HTTP API.
//
// The TRMNL firmware expects three endpoints:
//   GET  /api/setup   – one-time device registration
//   GET  /api/display – fetch the current screen image URL + refresh rate
//   POST /api/log     – upload device telemetry (accepted, ignored)
//
// Images are rendered on demand and cached for one refresh cycle.
package api

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/przemek/ha-go-trmnl/internal/config"
	"github.com/przemek/ha-go-trmnl/internal/ha"
	"github.com/przemek/ha-go-trmnl/internal/render"
)

// Handler serves the TRMNL device API.
type Handler struct {
	cfg    *config.Config
	haClient *ha.Client
	cacheDir string
	baseURL  string

	mu          sync.Mutex
	cachedFile  string
	cachedUntil time.Time
}

// New creates a Handler.
func New(cfg *config.Config, haClient *ha.Client, cacheDir string) *Handler {
	return &Handler{
		cfg:      cfg,
		haClient: haClient,
		cacheDir: cacheDir,
		baseURL:  cfg.BaseURL,
	}
}

// Mount registers all device API routes onto r.
func (h *Handler) Mount(r chi.Router) {
	r.Get("/api/setup", h.Setup)
	r.Get("/api/display", h.Display)
	r.Post("/api/log", h.Log)
	r.Get("/preview", h.Preview)
}

// Preview renders the current screen as a PNG and returns it for browser inspection.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	states, errs := h.haClient.GetStates(r.Context(), h.cfg.HAEntities)
	for _, e := range errs {
		slog.Warn("ha entity fetch error (preview)", "err", e)
	}
	page := &render.Page{
		Title:     h.cfg.Title,
		UpdatedAt: time.Now(),
		Entities:  states,
		Columns:   h.cfg.Columns,
	}
	png, err := render.RenderPNG(page)
	if err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

// Setup handles device registration.
// Real go-trmnl validates against a pre-registered MAC allowlist; we accept
// any device and return a static API key so the firmware can proceed.
func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	mac := r.Header.Get("ID") // firmware sends MAC in the "ID" header
	if mac == "" {
		mac = "unknown"
	}
	apiKey := fmt.Sprintf("%x", md5.Sum([]byte(mac)))
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       200,
		"api_key":      apiKey,
		"friendly_id":  mac,
		"image_url":    "",
		"filename":     "",
		"message":      "Welcome to ha-go-trmnl",
	})
}

// Display fetches HA entity states, renders the screen image (with TTL
// caching) and returns the URL + refresh rate to the firmware.
func (h *Handler) Display(w http.ResponseWriter, r *http.Request) {
	imgURL, err := h.currentImageURL(r.Context())
	if err != nil {
		slog.Error("render failed", "err", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       0,
		"image_url":    imgURL,
		"filename":     filepath.Base(imgURL),
		"refresh_rate": h.cfg.RefreshRate,
		"update_firmware": false,
		"firmware_url":    "",
		"reset_firmware":  false,
		"special_function": "sleep",
	})
}

// Log accepts device telemetry and discards it (no persistent store).
func (h *Handler) Log(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// currentImageURL returns the URL of the current rendered image, using a
// cached version when it is still fresh.
func (h *Handler) currentImageURL(ctx context.Context) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cachedFile != "" && time.Now().Before(h.cachedUntil) {
		return h.baseURL + "/images/" + filepath.Base(h.cachedFile), nil
	}

	states, errs := h.haClient.GetStates(ctx, h.cfg.HAEntities)
	for _, e := range errs {
		slog.Warn("ha entity fetch error", "err", e)
	}

	page := &render.Page{
		Title:     h.cfg.Title,
		UpdatedAt: time.Now(),
		Entities:  states,
		Columns:   h.cfg.Columns,
	}

	bmp, err := render.RenderBMP(page)
	if err != nil {
		return "", fmt.Errorf("rendering BMP: %w", err)
	}

	hash := fmt.Sprintf("%x", md5.Sum(bmp))
	fname := hash + ".bmp"
	fpath := filepath.Join(h.cacheDir, fname)

	if err := os.WriteFile(fpath, bmp, 0o644); err != nil {
		return "", fmt.Errorf("writing image cache: %w", err)
	}

	h.cachedFile = fpath
	h.cachedUntil = time.Now().Add(time.Duration(h.cfg.RefreshRate) * time.Second)

	return h.baseURL + "/images/" + fname, nil
}

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := jsonEncoder(w)
	_ = enc.Encode(v)
}
