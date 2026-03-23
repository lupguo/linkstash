//go:build wireinject
// +build wireinject

package di

import (
	"net/http"
	"time"

	"github.com/google/wire"
	"github.com/lupguo/linkstash/app/application"
	"github.com/lupguo/linkstash/app/domain/repos"
	"github.com/lupguo/linkstash/app/domain/services"
	"github.com/lupguo/linkstash/app/handler"
	"github.com/lupguo/linkstash/app/infra/config"
	"github.com/lupguo/linkstash/app/infra/db"
	"github.com/lupguo/linkstash/app/infra/llm"
	"github.com/lupguo/linkstash/app/infra/search"
	"gorm.io/gorm"
)

// --- Provider functions ---

func ProvideConfig(confPath string) (*config.Config, error) {
	return config.Load(confPath)
}

func ProvideDB(cfg *config.Config) (*gorm.DB, error) {
	return db.InitDB(&cfg.Database)
}

func ProvideURLRepo(database *gorm.DB) repos.URLRepo {
	return db.NewURLRepoImpl(database)
}

func ProvideVisitRepo(database *gorm.DB) repos.VisitRepo {
	return db.NewVisitRepoImpl(database)
}

func ProvideLLMLogRepo(database *gorm.DB) repos.LLMLogRepo {
	return db.NewLLMLogRepoImpl(database)
}

func ProvideEmbeddingRepo(database *gorm.DB) repos.EmbeddingRepo {
	return db.NewEmbeddingRepoImpl(database)
}

func ProvideLLMClient(cfg *config.Config, httpClient *http.Client) *llm.LLMClient {
	return llm.NewLLMClient(cfg.LLM.Chat, cfg.LLM.Embedding, httpClient)
}

func ProvideHTTPClient(cfg *config.Config) *http.Client {
	return config.NewHTTPClient(cfg.Proxy, 30*time.Second)
}

func ProvideKeywordSearch(database *gorm.DB) *search.KeywordSearch {
	return search.NewKeywordSearch(database)
}

func ProvideVectorSearch(embeddingRepo repos.EmbeddingRepo) *search.VectorSearch {
	vs := search.NewVectorSearch(embeddingRepo)
	if err := vs.LoadAll(); err != nil {
		// Non-fatal: log warning and continue
		println("[Warning] Failed to load embeddings cache:", err.Error())
	}
	return vs
}

func ProvideWorkerService(
	urlRepo repos.URLRepo,
	llmLogRepo repos.LLMLogRepo,
	embeddingRepo repos.EmbeddingRepo,
	llmClient *llm.LLMClient,
	cfg *config.Config,
	httpClient *http.Client,
) *services.WorkerService {
	return services.NewWorkerService(urlRepo, llmLogRepo, embeddingRepo, llmClient, cfg.LLM.Prompts, httpClient)
}

func ProvideAuthConfig(cfg *config.Config) *config.AuthConfig {
	return &cfg.Auth
}

func ProvideShortConfig(cfg *config.Config) *config.ShortConfig {
	return &cfg.Short
}

func ProvideWebHandler(
	urlUsecase *application.URLUsecase,
	searchUsecase *application.SearchUsecase,
	authCfg *config.AuthConfig,
	shortCfg *config.ShortConfig,
	cfg *config.Config,
) *handler.WebHandler {
	return handler.NewWebHandler(urlUsecase, searchUsecase, authCfg, shortCfg, cfg.Categories, "web")
}

// --- Provider sets ---

var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDB,
	ProvideHTTPClient,
	ProvideLLMClient,
	ProvideKeywordSearch,
	ProvideVectorSearch,
)

var RepoSet = wire.NewSet(
	ProvideURLRepo,
	ProvideVisitRepo,
	ProvideLLMLogRepo,
	ProvideEmbeddingRepo,
)

var ServiceSet = wire.NewSet(
	services.NewURLService,
	ProvideWorkerService,
	services.NewSearchService,
	services.NewVisitService,
)

var UsecaseSet = wire.NewSet(
	application.NewURLUsecase,
	application.NewAnalysisUsecase,
	application.NewSearchUsecase,
)

var HandlerSet = wire.NewSet(
	handler.NewAuthHandler,
	handler.NewURLHandler,
	handler.NewSearchHandler,
	handler.NewShortURLHandler,
	ProvideWebHandler,
	ProvideAuthConfig,
	ProvideShortConfig,
)

// InitializeApp creates a fully wired App instance.
func InitializeApp(confPath string) (*App, error) {
	wire.Build(
		InfraSet,
		RepoSet,
		ServiceSet,
		UsecaseSet,
		HandlerSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
