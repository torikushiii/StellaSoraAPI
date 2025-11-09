package app

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	MongoURI      string
	MongoDatabase string
}

type App struct {
	config      Config
	httpServer  *http.Server
	mongoClient *mongo.Client
	initOnce    sync.Once
	startTime   time.Time
	endpoints   []string
}

func New(cfg Config) *App {
	return &App{
		config:    cfg,
		startTime: time.Now(),
		endpoints: []string{
			"/stella/",
			"/stella/assets/{friendlyName}",
			"/stella/characters",
			"/stella/character/{idOrName}",
			"/stella/discs",
			"/stella/disc/{idOrName}",
			"/stella/banners",
			"/stella/events",
		},
	}
}

func (a *App) Start(ctx context.Context, handler http.Handler, addr string) error {
	if err := a.initMongo(ctx); err != nil {
		return err
	}

	a.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return a.httpServer.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	if a.mongoClient != nil {
		return a.mongoClient.Disconnect(ctx)
	}

	return nil
}

func (a *App) MongoClient() *mongo.Client {
	return a.mongoClient
}

func (a *App) StartTime() time.Time {
	return a.startTime
}

func (a *App) DatabaseName() string {
	return a.config.MongoDatabase
}

func (a *App) Endpoints() []string {
	result := make([]string, len(a.endpoints))
	copy(result, a.endpoints)
	return result
}

func (a *App) initMongo(ctx context.Context) error {
	var err error
	a.initOnce.Do(func() {
		clientCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		client, connectErr := mongo.Connect(clientCtx, options.Client().ApplyURI(a.config.MongoURI))
		if connectErr != nil {
			err = connectErr
			return
		}

		if pingErr := client.Ping(clientCtx, nil); pingErr != nil {
			_ = client.Disconnect(clientCtx)
			err = pingErr
			return
		}

		a.mongoClient = client
	})

	return err
}
