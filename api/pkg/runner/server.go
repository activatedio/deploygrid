package runner

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"net/http"
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

	c := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowCredentials: true,
		Debug:            true,
	})

	h := handlers.RecoveryHandler(handlers.RecoveryLogger(&recoveryLogger{}), handlers.PrintRecoveryStack(true))(c.Handler(router))

	server := &http.Server{
		Handler: h,
		Addr:    fmt.Sprintf("%s:%d", serverConfig.Host, serverConfig.Port),
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				server.ListenAndServe()
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
