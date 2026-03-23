package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/lupguo/linkstash/app/di"
	"github.com/lupguo/linkstash/app/infra/logger"
	"github.com/lupguo/linkstash/app/middleware"
)

// Version is set by ldflags at build time.
var Version = "dev"

func main() {
	confPath := flag.String("conf", "conf/app_dev.yaml", "config file path")
	flag.Parse()

	// Set version for DI layer
	di.AppVersion = Version

	app, err := di.InitializeApp(*confPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	// Initialize structured logger
	cleanup, err := logger.Setup(app.Config.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	// Start async worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.AnalysisUsecase.Start(ctx)

	// Wire URL handler with analysis usecase
	app.URLHandler.SetAnalysisUsecase(app.AnalysisUsecase)
	app.ShortURLHandler.SetAnalysisUsecase(app.AnalysisUsecase)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(chimw.Compress(5))

	// Public routes
	r.Post("/api/auth/token", app.AuthHandler.HandleToken)
	r.Get("/s/{code}", app.ShortURLHandler.HandleRedirect)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Web pages
	r.Get("/", app.WebHandler.HandleIndex)
	r.Get("/login", app.WebHandler.HandleLogin)
	r.Get("/cards", app.WebHandler.HandleIndexCards)
	r.Get("/urls/new", app.WebHandler.HandleNew)
	r.Get("/urls/{id}", app.WebHandler.HandleDetail)

	// Static files with cache headers
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", staticCacheMiddleware(http.StripPrefix("/static/", fileServer)))

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.JWTAuth(app.Config.Auth.JWTSecret))

		r.Post("/urls", app.URLHandler.HandleCreate)
		r.Get("/urls", app.URLHandler.HandleList)
		r.Get("/urls/{id}", app.URLHandler.HandleGet)
		r.Put("/urls/{id}", app.URLHandler.HandleUpdate)
		r.Delete("/urls/{id}", app.URLHandler.HandleDelete)
		r.Post("/urls/{id}/visit", app.URLHandler.HandleVisit)
		r.Post("/urls/{id}/reanalyze", app.URLHandler.HandleReanalyze)

		r.Get("/search", app.SearchHandler.HandleSearch)

		r.Post("/short-links", app.ShortURLHandler.HandleCreate)
		r.Get("/short-links", app.ShortURLHandler.HandleList)
		r.Put("/short-links/{id}", app.ShortURLHandler.HandleUpdate)
		r.Delete("/short-links/{id}", app.ShortURLHandler.HandleDelete)
	})

	addr := app.Config.Server.Addr()
	slog.Info("LinkStash starting", "addr", addr, "version", Version)

	// Create HTTP server for graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Listen for shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigCh
	slog.Info("received shutdown signal", "signal", sig)

	// Graceful shutdown with 10-second timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}

	// Cancel worker context
	cancel()

	// Close browser instances
	if app.BrowserService != nil {
		slog.Info("closing browser instances", "component", "shutdown")
		app.BrowserService.Close()
	}

	slog.Info("LinkStash stopped gracefully")
}

// staticCacheMiddleware adds Cache-Control headers for static assets.
func staticCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, ".css"),
			strings.HasSuffix(path, ".js"),
			strings.HasSuffix(path, ".woff2"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasSuffix(path, ".svg"),
			strings.HasSuffix(path, ".png"):
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		next.ServeHTTP(w, r)
	})
}
