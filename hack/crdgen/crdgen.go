package main

import (
	"github.com/activatedio/deploygrid/hack"
	"github.com/activatedio/wrangler/crdgen"
)

func main() {
	crdgen.Run(hack.ListCRDs())
}
