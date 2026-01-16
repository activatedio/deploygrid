package main

import (
	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
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

	err := NewRootCmd().Execute()

	if err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}

const (
	FlagConfig      = "config"
	FlagConfigShort = "c"
)

func NewRootCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use: "deploygrid",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("Starting deploygrid")

			configPath := cmd.Flag(FlagConfig).Value.String()

			v := apiinfraconfig.NewConfig(configPath)
			lc := config.NewLoggingConfig(v)
			apiinfrazerolog.ConfigureLogging(lc)

			fx.New(deploygridfx.Index(v), fx.Invoke(func(server *runner.RunningServer) {
				log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
			})).Run()

			return nil
		},
	}

	cmd.PersistentFlags().StringP(FlagConfig, FlagConfigShort, "", "path to the configuration file")

	return cmd
}
