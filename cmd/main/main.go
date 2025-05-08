package main

import (
	"github.com/activatedio/deploygrid/pkg/apiinfra/config"
	apiinfraviper "github.com/activatedio/deploygrid/pkg/apiinfra/viper"
	apiinfrazerolog "github.com/activatedio/deploygrid/pkg/apiinfra/zerolog"
	"github.com/rs/zerolog/log"
	cobra "github.com/spf13/cobra"
)

func main() {

	v := apiinfraviper.NewViper()
	lc := config.NewLoggingConfig(v)
	apiinfrazerolog.ConfigureLogging(lc)

	err := NewRootCmd().Execute()

	if err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}

func NewRootCmd() *cobra.Command {

	return &cobra.Command{
		Use: "deploygrid",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("Starting deploygrid")
			return nil
		},
	}
}
