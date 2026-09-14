// ha-trmnld is a TRMNL-compatible display server that renders Home Assistant
// entity states on e-ink displays.
//
// Quick start:
//
//	export HA_URL=http://homeassistant.local:8123
//	export HA_TOKEN=<long-lived-access-token>
//	export HA_ENTITIES=sensor.temperature,light.living_room,switch.fan
//	export BASE_URL=http://<this-machine-ip>:8080
//	./ha-trmnld
//
// Point your TRMNL device at http://<this-machine-ip>:8080 and it will show
// your Home Assistant entities on the e-ink screen.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/przemek/ha-go-trmnl/internal/api"
	"github.com/przemek/ha-go-trmnl/internal/config"
	"github.com/przemek/ha-go-trmnl/internal/ha"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n\n", err)
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  HA_URL=http://homeassistant.local:8123 \\")
		fmt.Fprintln(os.Stderr, "  HA_TOKEN=<token> \\")
		fmt.Fprintln(os.Stderr, "  HA_ENTITIES=sensor.temp,light.hall \\")
		fmt.Fprintln(os.Stderr, "  BASE_URL=http://192.168.1.100:8080 \\")
		fmt.Fprintln(os.Stderr, "  ha-trmnld")
		os.Exit(1)
	}

	cacheDir, err := os.MkdirTemp("", "ha-trmnl-*")
	if err != nil {
		slog.Error("failed to create cache dir", "err", err)
		os.Exit(1)
	}
	defer os.RemoveAll(cacheDir)

	haClient := ha.New(cfg.HAURL, cfg.HAToken)
	apiHandler := api.New(cfg, haClient, cacheDir)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Device API endpoints
	apiHandler.Mount(r)

	// Serve cached BMP images to the device firmware
	r.Get("/images/{filename}", func(w http.ResponseWriter, req *http.Request) {
		fname := chi.URLParam(req, "filename")
		// Basic path traversal guard
		if filepath.Base(fname) != fname {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fpath := filepath.Join(cacheDir, fname)
		http.ServeFile(w, req, fpath)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("ha-trmnld starting",
			"addr", cfg.ListenAddr,
			"base_url", cfg.BaseURL,
			"ha_url", cfg.HAURL,
			"entities", cfg.HAEntities,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
