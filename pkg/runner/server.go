package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-errors/errors"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/activatedio/deploygrid/pkg/config"
)

type recoveryLogger struct {
}

func (r *recoveryLogger) Println(i ...interface{}) {
	evt := log.Error()
	for idx, val := range i {
		evt = evt.Interface(fmt.Sprintf("%d", idx), val)
	}
	evt.Msg("recover from panic")
}

func NewServer(router *mux.Router, serverConfig *config.ServerConfig, lifecycle fx.Lifecycle) *RunningServer {

	var h http.Handler = router
	if origins := serverConfig.CorsOrigins(); len(origins) > 0 {
		h = cors.New(cors.Options{
			AllowedOrigins: origins,
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
		}).Handler(h)
	}

	h = handlers.RecoveryHandler(handlers.RecoveryLogger(&recoveryLogger{}), handlers.PrintRecoveryStack(true))(h)

	server := &http.Server{
		Handler:           h,
		Addr:              fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port),
		ReadHeaderTimeout: 10 * time.Second,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				err := server.ListenAndServe()
				if !errors.Is(err, http.ErrServerClosed) {
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("Shutting down server")
			return server.Shutdown(ctx)
		},
	})

	return &RunningServer{
		Host: serverConfig.Host,
		Port: serverConfig.Port,
	}
}
