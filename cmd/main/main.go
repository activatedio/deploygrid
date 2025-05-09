package main

import (
	apiinfraviper "github.com/activatedio/deploygrid/pkg/apiinfra/viper"
	apiinfrazerolog "github.com/activatedio/deploygrid/pkg/apiinfra/zerolog"
	"github.com/activatedio/deploygrid/pkg/config"
	deploygridfx "github.com/activatedio/deploygrid/pkg/fx"
	"github.com/activatedio/deploygrid/pkg/runner"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func main() {

	v := apiinfraviper.NewViper()
	lc := config.NewLoggingConfig(v)
	apiinfrazerolog.ConfigureLogging(lc)

	err := NewRootCmd(v).Execute()

	if err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}

func NewRootCmd(v *viper.Viper) *cobra.Command {

	return &cobra.Command{
		Use: "deploygrid",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("Starting deploygrid")

			fx.New(deploygridfx.Index(v), fx.Invoke(func(server *runner.RunningServer) {
				log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
			})).Run()

			return nil
		},
	}
}
