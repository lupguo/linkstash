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
	"github.com/lupguo/linkstash/app/infra/browser"
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

func ProvideBrowserService(cfg *config.Config) *browser.BrowserService {
	return browser.NewBrowserService(cfg.Browser, cfg.Proxy.HTTPProxy)
}

func ProvideKeywordSearch(cfg *config.Config, database *gorm.DB) search.KeywordSearcher {
	if cfg.Database.IsMySQL() {
		return search.NewLikeKeywordSearch(database)
	}
	return search.NewFTS5KeywordSearch(database)
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
	browserSvc *browser.BrowserService,
) *services.WorkerService {
	return services.NewWorkerService(urlRepo, llmLogRepo, embeddingRepo, llmClient, cfg.LLM.Prompts, httpClient, browserSvc)
}

func ProvideAuthConfig(cfg *config.Config) *config.AuthConfig {
	return &cfg.Auth
}

func ProvideWebHandler() *handler.WebHandler {
	return handler.NewWebHandler("web", AppVersion)
}

// --- Provider sets ---

var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDB,
	ProvideHTTPClient,
	ProvideLLMClient,
	ProvideBrowserService,
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
