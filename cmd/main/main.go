package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	apiinfrazerolog "github.com/activatedio/deploygrid/pkg/apiinfra/zerolog"
	"github.com/activatedio/deploygrid/pkg/config"
	deploygridfx "github.com/activatedio/deploygrid/pkg/fx"
	"github.com/activatedio/deploygrid/pkg/runner"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			log.Info().Msg("Starting deploygrid")

			configPath := cmd.Flag(FlagConfig).Value.String()
			if configPath == "" {
				configPath = os.Getenv(config.EnvConfigPath)
			}

			m := config.NewMainConfig(apiinfraconfig.NewConfig(configPath))
			apiinfrazerolog.ConfigureLogging(&m.Logging)

			fx.New(deploygridfx.Index(m), fx.Invoke(func(server *runner.RunningServer) {
				log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
			})).Run()

			return nil
		},
	}

	cmd.PersistentFlags().StringP(FlagConfig, FlagConfigShort, "", "path to the configuration file (default $CONFIG_PATH)")

	return cmd
}
