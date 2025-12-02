package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

var mode string
var chartName string

func init() {
	const (
		defaultMode      = "clean"
		defaultChartName = "default"
	)
	flag.StringVar(&mode, "mode", defaultMode, "mode for use")
	flag.StringVar(&chartName, "chart-name", defaultChartName, "chart name")
}

func main() {

	flag.Parse()

	path := flag.Arg(0)

	var p *printer
	var err error

	p = &printer{
		nodeVisitor: helmNodeVisitor,
		documentVisitor: &helmDocumentVisitor{
			chartName: chartName,
		},
	}
	fmt.Println("# START CRD {{- if .Values.crds.enabled }}")

	var dir []os.DirEntry
	fmt.Println(path)
	dir, err = os.ReadDir(path)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open directory")
	}

	var f *os.File
	for _, e := range dir {
		if !e.IsDir() {
			f, err = os.Open(filepath.Join(path, e.Name()))
			if err != nil {
				log.Fatal().Err(err).Msg("failed to open file")
			}
			err = p.run(f)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to process CRD")
			}
		}
	}

	fmt.Println("# END CRD {{- end }}")

	if err != nil {
		log.Fatal().Err(err).Msg("failed to run")
	}

}
