package di

import (
	"github.com/lupguo/linkstash/app/application"
	"github.com/lupguo/linkstash/app/domain/services"
	"github.com/lupguo/linkstash/app/handler"
	"github.com/lupguo/linkstash/app/infra/browser"
	"github.com/lupguo/linkstash/app/infra/config"
)

// App holds all initialized components needed by the server.
type App struct {
	Config          *config.Config
	AuthHandler     *handler.AuthHandler
	URLHandler      *handler.URLHandler
	SearchHandler   *handler.SearchHandler
	ShortURLHandler *handler.ShortURLHandler
	WebHandler      *handler.WebHandler
	AnalysisUsecase *application.AnalysisUsecase
	VisitService    *services.VisitService
	BrowserService  *browser.BrowserService
}
