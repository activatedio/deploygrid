package k8s

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/go-errors/errors"
	"github.com/sony/gobreaker/v2"
	"go.uber.org/fx"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"strings"
	"sync"
)

type componentRepository struct {
	client dynamic.Interface
	cb     *gobreaker.CircuitBreaker[[]repository.Component]
}

func (c *componentRepository) List(ctx context.Context, criteria repository.ComponentCriteria) ([]repository.Component, error) {
	return c.cb.Execute(func() ([]repository.Component, error) {

		parts := strings.Split(criteria.ApiVersion, "/")

		var (
			group, version string
		)

		if len(parts) == 2 {
			group = parts[0]
			version = parts[1]
		} else if len(parts) == 1 {
			group = ""
			version = parts[0]
		} else {
			return nil, errors.New("invalid api version")
		}

		var res dynamic.ResourceInterface

		nres := c.client.Resource(schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: criteria.Kind,
		})

		if criteria.Namespace != "" {
			res = nres.Namespace(criteria.Namespace)
		} else {
			res = nres
		}

		initial := true
		cont := ""
		var lists []*unstructured.UnstructuredList

		for initial || cont != "" {
			initial = false
			list, err := res.List(ctx, metav1.ListOptions{
				LabelSelector: criteria.LabelSelector,
				Continue:      cont,
			})
			if err != nil {
				return nil, err
			}
			lists = append(lists, list)
			cont = list.GetContinue()
		}

		return c.toComponents(lists)

	})
}

type apiVersionKind struct {
	apiVersion string
	kind       string
}

func decodeMap(in map[string]any, to any) error {

	bs, err := json.Marshal(in)

	if err != nil {
		return err
	}

	return json.Unmarshal(bs, to)

}

var (
	applicationHandler = func(in map[string]any) ([]repository.Component, error) {
		app := &Application{}

		err := decodeMap(in, app)

		if err != nil {
			return nil, err
		}

		compName := app.Spec.Source.Chart
		if compName == "" {
			parts := strings.Split(app.Spec.Source.RepoURL, "/")
			compName = parts[len(parts)-1]
		}

		return []repository.Component{
			{
				Namespace:     app.Namespace,
				ApiVersion:    app.APIVersion,
				Kind:          app.Kind,
				Name:          app.Name,
				ComponentName: compName,
				Labels:        app.Labels,
				Version:       app.Spec.Source.TargetRevision,
			},
		}, nil
	}
	deploymentHandler = func(in map[string]any) ([]repository.Component, error) {

		d := &appsv1.Deployment{}

		err := decodeMap(in, d)

		if err != nil {
			return nil, err
		}

		var res []repository.Component

		for _, c := range d.Spec.Template.Spec.Containers {

			imgParts := strings.Split(c.Image, ":")

			res = append(res, repository.Component{
				Namespace:        d.Namespace,
				ApiVersion:       d.APIVersion,
				Kind:             d.Kind,
				ComponentName:    d.Name,
				SubComponentName: c.Name,
				Name:             d.Name,
				Labels:           d.Labels,
				Version:          imgParts[len(imgParts)-1],
			})

		}

		return res, nil
	}
	handlers = map[apiVersionKind]func(in map[string]any) ([]repository.Component, error){
		{
			apiVersion: "argoproj.io/v1alpha1",
			kind:       "Application",
		}: applicationHandler,
		{
			apiVersion: "apps/v1",
			kind:       "Deployment",
		}: deploymentHandler,
	}
)

func (c *componentRepository) toComponents(lists []*unstructured.UnstructuredList) ([]repository.Component, error) {

	var res []repository.Component

	for _, list := range lists {
		err := list.EachListItem(func(obj runtime.Object) error {
			if u, ok := obj.(*unstructured.Unstructured); ok {
				if h, _ok := handlers[apiVersionKind{u.GetAPIVersion(), u.GetKind()}]; _ok {

					_res, err := h(u.Object)
					if err != nil {
						return err
					}
					res = append(res, _res...)

				} else {
					return errors.New(fmt.Sprintf("handler not found for apiVersion %s Kind %s", u.GetAPIVersion(), u.GetKind()))
				}
			} else {
				return errors.New("unexpected object")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func NewComponentRepository(name string, client dynamic.Interface) repository.ComponentRepository {
	return &componentRepository{
		client: client,
		cb: gobreaker.NewCircuitBreaker[[]repository.Component](gobreaker.Settings{
			Name: name,
		}),
	}
}

type cluster struct {
	config *config.ClusterConfig
	cb     *gobreaker.CircuitBreaker[repository.ComponentRepository]
}

type componentRepositoryClusterAwareAccessor struct {
	clusters     map[string]cluster
	repositoires map[string]repository.ComponentRepository
	lock         sync.Mutex
}

func (c *componentRepositoryClusterAwareAccessor) ClusterNames(ctx context.Context) []string {
	return nil
}

func (c *componentRepositoryClusterAwareAccessor) Get(ctx context.Context, clusterName string) (repository.ComponentRepository, error) {

	c.lock.Lock()
	defer c.lock.Unlock()

	if r, ok := c.repositoires[clusterName]; ok {
		return r, nil
	}

	cl, ok := c.clusters[clusterName]

	if !ok {
		return nil, errors.New(fmt.Sprintf("cluster not found: %s", clusterName))
	}

	r, err := cl.cb.Execute(func() (repository.ComponentRepository, error) {

		client, err := dynamic.NewForConfig(&rest.Config{
			Host: cl.config.Address,
		})

		if err != nil {
			return nil, err
		}

		return NewComponentRepository(cl.config.Name, client), nil
	})

	if err != nil {
		return nil, err
	}

	c.repositoires[clusterName] = r

	return r, nil
}

type ComponentRepositoryClusterAwareAccessorParams struct {
	fx.In
	ClustersConfig *config.ClustersConfig
}

func NewComponentRepositoryClusterAwareAccessor(params ComponentRepositoryClusterAwareAccessorParams) repository.ClusterAwareAccessor[repository.ComponentRepository] {

	clusters := map[string]cluster{}

	for _, c := range params.ClustersConfig.Clusters {
		clusters[c.Name] = cluster{
			config: &c,
			cb: gobreaker.NewCircuitBreaker[repository.ComponentRepository](gobreaker.Settings{
				Name: "factory",
			}),
		}
	}

	return &componentRepositoryClusterAwareAccessor{
		clusters: clusters,
	}
}
