package zerolog

import (
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
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
			l, err := zerolog.ParseLevel(lvl)
			util.Check(err)
			zerolog.SetGlobalLevel(l)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

	}

}
