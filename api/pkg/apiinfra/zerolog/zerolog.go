package zerolog

import (
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func ConfigureLogging(config *config.LoggingConfig) {

	if config.DevMode {
		fmt.Println("Enabling dev logging")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(&zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		if lvl := config.Level; lvl != "" {
			fmt.Println("Setting logging level: " + lvl)
			l, err := zerolog.ParseLevel(lvl)
			util.Check(err)
			zerolog.SetGlobalLevel(l)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

	}

}
