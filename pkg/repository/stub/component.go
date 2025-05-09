package stub

import (
	"context"
	"errors"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
	"os"
)

type componentRepository struct {
	data []repository.Component
}

func (c *componentRepository) List(ctx context.Context, criteria repository.ComponentCriteria) ([]repository.Component, error) {
	var result []repository.Component

	for _, el := range c.data {
		if criteria.Namespace != "" && criteria.Namespace != el.Namespace {
			continue
		}
		if el.Kind == criteria.Kind && el.ApiVersion == criteria.ApiVersion {
			result = append(result, el)
		}
	}

	return result, nil
}

func NewComponentRepository(data []repository.Component) repository.ComponentRepository {
	return &componentRepository{
		data: data,
	}
}

func mustParseData(in []byte) map[string][]repository.Component {

	d := map[string][]repository.Component{}
	err := yaml.Unmarshal(in, &d)
	util.Check(err)
	return d
}

func mustParseDataFromFile(path string) map[string][]repository.Component {

	bs, err := os.ReadFile(path)
	util.Check(err)
	return mustParseData(bs)
}

type componentRepositoryClusterAwareAccessor struct {
	repositories map[string]repository.ComponentRepository
}

func (c *componentRepositoryClusterAwareAccessor) ClusterNames(ctx context.Context) []string {
	var res []string

	for k, _ := range c.repositories {
		res = append(res, k)
	}

	return res
}

func (c *componentRepositoryClusterAwareAccessor) Get(ctx context.Context, clusterName string) (repository.ComponentRepository, error) {

	if r, ok := c.repositories[clusterName]; ok {
		return r, nil
	} else {
		return nil, errors.New(fmt.Sprintf("cluster not found: %s", clusterName))
	}

}

type ComponentRepositoryClusterAwareAccessorParams struct {
	fx.In
	RepositoryStubConfig *config.RepositoryStubConfig
}

func NewComponentRepositoryClusterAwareAccessor(params ComponentRepositoryClusterAwareAccessorParams) repository.ClusterAwareAccessor[repository.ComponentRepository] {

	c := params.RepositoryStubConfig

	var data map[string][]repository.Component

	if c.StaticDataBytes != nil {
		data = mustParseData(c.StaticDataBytes)
	} else {
		data = mustParseDataFromFile(c.StaticDataPath)
	}

	repos := map[string]repository.ComponentRepository{}

	for k, v := range data {
		repos[k] = NewComponentRepository(v)
	}

	return &componentRepositoryClusterAwareAccessor{
		repositories: repos,
	}
}
