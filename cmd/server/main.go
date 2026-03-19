package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/lupguo/linkstash/app/di"
	"github.com/lupguo/linkstash/app/middleware"
)

func main() {
	confPath := flag.String("conf", "conf/app_dev.yaml", "config file path")
	flag.Parse()

	app, err := di.InitializeApp(*confPath)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Start async worker
	ctx := context.Background()
	app.AnalysisUsecase.Start(ctx)

	// Wire URL handler with analysis usecase
	app.URLHandler.SetAnalysisUsecase(app.AnalysisUsecase)
	app.ShortURLHandler.SetAnalysisUsecase(app.AnalysisUsecase)

	// Setup router
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

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

	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.JWTAuth(app.Config.Auth.JWTSecret))

		r.Post("/urls", app.URLHandler.HandleCreate)
		r.Get("/urls", app.URLHandler.HandleList)
		r.Get("/urls/{id}", app.URLHandler.HandleGet)
		r.Put("/urls/{id}", app.URLHandler.HandleUpdate)
		r.Delete("/urls/{id}", app.URLHandler.HandleDelete)
		r.Post("/urls/{id}/visit", app.URLHandler.HandleVisit)

		r.Get("/search", app.SearchHandler.HandleSearch)

		r.Post("/short-links", app.ShortURLHandler.HandleCreate)
		r.Get("/short-links", app.ShortURLHandler.HandleList)
		r.Put("/short-links/{id}", app.ShortURLHandler.HandleUpdate)
		r.Delete("/short-links/{id}", app.ShortURLHandler.HandleDelete)
	})

	addr := app.Config.Server.Addr()
	log.Printf("[Server] LinkStash starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
