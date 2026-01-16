package main

import (
	"os"

	"github.com/rs/zerolog/log"
)

func main() {
	if err := os.RemoveAll("./pkg/generated"); err != nil {
		log.Fatal().Err(err).Msg("unable to remove ./pkg/generated")
	}
}
