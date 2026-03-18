//go:build wireinject
// +build wireinject

package di

import (
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

func ProvideShortURLRepo(database *gorm.DB) repos.ShortURLRepo {
	return db.NewShortURLRepoImpl(database)
}

func ProvideLLMClient(cfg *config.Config) *llm.LLMClient {
	return llm.NewLLMClient(cfg.LLM.Chat, cfg.LLM.Embedding)
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
) *services.WorkerService {
	return services.NewWorkerService(urlRepo, llmLogRepo, embeddingRepo, llmClient, cfg.LLM.Prompts)
}

func ProvideAuthConfig(cfg *config.Config) *config.AuthConfig {
	return &cfg.Auth
}

func ProvideWebHandler(
	urlUsecase *application.URLUsecase,
	searchUsecase *application.SearchUsecase,
	authCfg *config.AuthConfig,
) *handler.WebHandler {
	return handler.NewWebHandler(urlUsecase, searchUsecase, authCfg, "web")
}

// --- Provider sets ---

var InfraSet = wire.NewSet(
	ProvideConfig,
	ProvideDB,
	ProvideLLMClient,
	ProvideKeywordSearch,
	ProvideVectorSearch,
)

var RepoSet = wire.NewSet(
	ProvideURLRepo,
	ProvideVisitRepo,
	ProvideLLMLogRepo,
	ProvideEmbeddingRepo,
	ProvideShortURLRepo,
)

var ServiceSet = wire.NewSet(
	services.NewURLService,
	ProvideWorkerService,
	services.NewSearchService,
	services.NewShortURLService,
	services.NewVisitService,
)

var UsecaseSet = wire.NewSet(
	application.NewURLUsecase,
	application.NewAnalysisUsecase,
	application.NewSearchUsecase,
	application.NewShortURLUsecase,
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
