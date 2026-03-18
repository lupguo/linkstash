package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/lupguo/linkstash/app/application"
	"github.com/lupguo/linkstash/app/domain/services"
	"github.com/lupguo/linkstash/app/handler"
	"github.com/lupguo/linkstash/app/infra/config"
	"github.com/lupguo/linkstash/app/infra/db"
	"github.com/lupguo/linkstash/app/infra/llm"
	"github.com/lupguo/linkstash/app/infra/search"
	"github.com/lupguo/linkstash/app/middleware"
)

func main() {
	confPath := flag.String("conf", "configs/app_dev.yaml", "config file path")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*confPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	database, err := db.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize repositories
	urlRepo := db.NewURLRepoImpl(database)
	visitRepo := db.NewVisitRepoImpl(database)
	llmLogRepo := db.NewLLMLogRepoImpl(database)
	embeddingRepo := db.NewEmbeddingRepoImpl(database)
	shortURLRepo := db.NewShortURLRepoImpl(database)

	// Initialize LLM client
	llmClient := llm.NewLLMClient(cfg.LLM.Chat, cfg.LLM.Embedding)

	// Initialize search infrastructure
	keywordSearch := search.NewKeywordSearch(database)
	vectorSearch := search.NewVectorSearch(embeddingRepo)
	if err := vectorSearch.LoadAll(); err != nil {
		log.Printf("[Warning] Failed to load embeddings cache: %v", err)
	}

	// Initialize services
	urlService := services.NewURLService(urlRepo)
	workerService := services.NewWorkerService(urlRepo, llmLogRepo, embeddingRepo, llmClient, cfg.LLM.Prompts)
	searchService := services.NewSearchService(keywordSearch, vectorSearch, llmClient)
	shortURLService := services.NewShortURLService(shortURLRepo)
	visitService := services.NewVisitService(visitRepo)

	// Initialize use cases
	urlUsecase := application.NewURLUsecase(urlService)
	analysisUsecase := application.NewAnalysisUsecase(workerService)
	searchUsecase := application.NewSearchUsecase(searchService, urlRepo)
	shortURLUsecase := application.NewShortURLUsecase(shortURLService)

	// Start async worker
	ctx := context.Background()
	analysisUsecase.Start(ctx)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(&cfg.Auth)
	urlHandler := handler.NewURLHandler(urlUsecase)
	urlHandler.SetAnalysisUsecase(analysisUsecase)
	searchHandler := handler.NewSearchHandler(searchUsecase)
	shortURLHandler := handler.NewShortURLHandler(shortURLUsecase)
	webHandler := handler.NewWebHandler(urlUsecase, searchUsecase, &cfg.Auth, "web")

	_ = visitService // available for future visit tracking integration

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	// Public routes
	r.Post("/api/auth/token", authHandler.HandleToken)
	r.Get("/s/{code}", shortURLHandler.HandleRedirect) // short link redirect (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Web pages (auth checked in handler)
	r.Get("/", webHandler.HandleIndex)
	r.Get("/login", webHandler.HandleLogin)
	r.Get("/urls/{id}", webHandler.HandleDetail)
	r.Get("/search", webHandler.HandleSearch)
	r.Get("/short", webHandler.HandleShort)

	// Static files
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))

		r.Post("/urls", urlHandler.HandleCreate)
		r.Get("/urls", urlHandler.HandleList)
		r.Get("/urls/{id}", urlHandler.HandleGet)
		r.Put("/urls/{id}", urlHandler.HandleUpdate)
		r.Delete("/urls/{id}", urlHandler.HandleDelete)
		r.Post("/urls/{id}/visit", urlHandler.HandleVisit)

		r.Get("/search", searchHandler.HandleSearch)

		r.Post("/short-links", shortURLHandler.HandleCreate)
		r.Get("/short-links", shortURLHandler.HandleList)
		r.Delete("/short-links/{id}", shortURLHandler.HandleDelete)
	})

	addr := cfg.Server.Addr()
	log.Printf("[Server] LinkStash starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
	fmt.Println("Server stopped")
}
