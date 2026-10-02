package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.uber.org/fx"

	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	apiinfrazerolog "github.com/activatedio/deploygrid/pkg/apiinfra/zerolog"
	"github.com/activatedio/deploygrid/pkg/collector"
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

func loadConfig(cmd *cobra.Command) *config.Main {
	configPath := cmd.Flag(FlagConfig).Value.String()
	if configPath == "" {
		configPath = os.Getenv(config.EnvConfigPath)
	}
	m := config.NewMainConfig(apiinfraconfig.NewConfig(configPath))
	apiinfrazerolog.ConfigureLogging(&m.Logging)
	return m
}

func NewRootCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "deploygrid",
		Short: "DeployGrid API server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			log.Info().Msg("Starting deploygrid")
			m := loadConfig(cmd)

			fx.New(deploygridfx.Index(m), fx.Invoke(func(server *runner.RunningServer) {
				log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
			})).Run()

			return nil
		},
	}

	cmd.PersistentFlags().StringP(FlagConfig, FlagConfigShort, "", "path to the configuration file (default $CONFIG_PATH)")
	cmd.AddCommand(newCollectorCmd())

	return cmd
}

func newCollectorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collector",
		Short: "Watch the current cluster and push observations to a deploygrid server",
		Long: `Runs inside (or next to) an observed cluster. Configure it through the
collector section of the configuration file or with environment variables:
DEPLOYGRID_COLLECTOR_SERVER, DEPLOYGRID_COLLECTOR_CLUSTER and
DEPLOYGRID_COLLECTOR_TOKEN (or DEPLOYGRID_COLLECTOR_TOKEN_FILE).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m := loadConfig(cmd)
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			return collector.RunFromConfig(ctx, &m.Collector)
		},
	}
	return cmd
}
