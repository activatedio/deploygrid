package mux

import (
	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go/openapi3"
	"net/http"
)

type OpenapiConfig struct {
	Title       string
	Version     string
	Description string
}

type openapiImpl struct {
	config *OpenapiConfig
}

func (o *openapiImpl) Mount(router *mux.Router, builders ...OpenapiBuilder) error {

	r := openapi3.NewReflector()
	ss := r.SpecSchema()
	ss.SetTitle(o.config.Title)
	ss.SetVersion(o.config.Version)
	ss.SetDescription(o.config.Description)

	for _, b := range builders {
		err := b(r)
		if err != nil {
			return err
		}
	}

	out, err := r.Spec.MarshalJSON()

	if err != nil {
		return err
	}

	router.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}).Methods(http.MethodGet)

	return nil
}

func NewOpenapi(config *OpenapiConfig) Openapi {
	return &openapiImpl{
		config: config,
	}
}
