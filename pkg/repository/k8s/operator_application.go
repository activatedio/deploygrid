package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/jsonpath"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
)

const (
	defaultDesiredVersionPath  = "{.spec.version}"
	defaultHealthConditionType = "Ready"
)

// OperatorApplicationConverter turns an operator's application custom
// resource into an observed Resource according to its configuration.
type OperatorApplicationConverter struct {
	cfg       config.ApplicationKindConfig
	desired   *jsonpath.JSONPath
	running   *jsonpath.JSONPath
	pinned    *jsonpath.JSONPath
	env       *jsonpath.JSONPath
	condition string
}

func compilePath(name, expr string) (*jsonpath.JSONPath, error) {
	if expr == "" {
		return nil, nil
	}
	if !strings.HasPrefix(expr, "{") {
		expr = "{" + expr + "}"
	}
	jp := jsonpath.New(name).AllowMissingKeys(true)
	if err := jp.Parse(expr); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return jp, nil
}

// NewOperatorApplicationConverter validates the paths once.
func NewOperatorApplicationConverter(cfg config.ApplicationKindConfig) (*OperatorApplicationConverter, error) {
	desiredPath := cfg.DesiredVersionPath
	if desiredPath == "" {
		desiredPath = defaultDesiredVersionPath
	}
	desired, err := compilePath("desiredVersionPath", desiredPath)
	if err != nil {
		return nil, err
	}
	running, err := compilePath("runningVersionPath", cfg.RunningVersionPath)
	if err != nil {
		return nil, err
	}
	pinned, err := compilePath("pinnedVersionsPath", cfg.PinnedVersionsPath)
	if err != nil {
		return nil, err
	}
	env, err := compilePath("environmentPath", cfg.EnvironmentPath)
	if err != nil {
		return nil, err
	}
	cond := cfg.HealthConditionType
	if cond == "" {
		cond = defaultHealthConditionType
	}
	return &OperatorApplicationConverter{cfg: cfg, desired: desired, running: running, pinned: pinned, env: env, condition: cond}, nil
}

// strings evaluates a path and returns every scalar result as text.
func evalStrings(jp *jsonpath.JSONPath, obj map[string]any) []string {
	if jp == nil {
		return nil
	}
	results, err := jp.FindResults(obj)
	if err != nil {
		return nil
	}
	var out []string
	for _, group := range results {
		for _, v := range group {
			if !v.IsValid() || !v.CanInterface() {
				continue
			}
			switch x := v.Interface().(type) {
			case string:
				if x != "" {
					out = append(out, x)
				}
			case nil:
			default:
				out = append(out, fmt.Sprint(x))
			}
		}
	}
	return out
}

// conditionStatus returns the status of a condition type from
// .status.conditions, or "" when absent.
func conditionStatus(obj map[string]any, condType string) string {
	conds, _, _ := unstructured.NestedSlice(obj, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := m["type"].(string); t == condType {
			s, _ := m["status"].(string)
			return s
		}
	}
	return ""
}

// health rolls conditions up: an explicit Degraded wins, then the configured
// readiness condition, then Progressing.
func (c *OperatorApplicationConverter) health(obj map[string]any) string {
	switch {
	case conditionStatus(obj, "Degraded") == string(metav1.ConditionTrue):
		return repository.HealthDegraded
	case conditionStatus(obj, c.condition) == string(metav1.ConditionTrue):
		return repository.HealthHealthy
	case conditionStatus(obj, "Progressing") == string(metav1.ConditionTrue):
		return repository.HealthProgressing
	case conditionStatus(obj, c.condition) == string(metav1.ConditionFalse):
		return repository.HealthDegraded
	default:
		return repository.HealthUnknown
	}
}

// versionOf reduces a version or image reference to a plain version.
func versionOf(v string) string {
	if strings.Contains(v, "/") || strings.Contains(v, ":") || strings.Contains(v, "@") {
		return ParseImageReference(v).Version()
	}
	return v
}

// runningVersion derives the running version from the configured path. Image
// references are reduced to their version; versions listed as pinned are set
// aside. When every remaining value agrees that is the version, otherwise ""
// (the builder then reports inconsistency and picks the majority from the
// workloads).
func runningVersion(values, pinned []string) (version string, comps []repository.Component) {
	isPinned := map[string]bool{}
	for _, p := range pinned {
		isPinned[versionOf(p)] = true
	}
	seen := map[string]bool{}
	for _, v := range values {
		ver := versionOf(v)
		if ver != v {
			ref := ParseImageReference(v)
			name := ref.Repository[strings.LastIndex(ref.Repository, "/")+1:]
			comps = append(comps, repository.Component{Name: name, Kind: repository.VersionKindContainer, Version: ver, Image: v})
		}
		if !isPinned[ver] {
			seen[ver] = true
		}
	}
	if len(seen) == 1 {
		for v := range seen {
			version = v
		}
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i].Name < comps[j].Name })
	return version, comps
}

// Convert builds the Resource for one custom resource object.
func (c *OperatorApplicationConverter) Convert(obj *unstructured.Unstructured) (*repository.Resource, error) {
	o := obj.Object
	desired := ""
	if vals := evalStrings(c.desired, o); len(vals) > 0 {
		desired = vals[0]
	}
	pinnedRaw := evalStrings(c.pinned, o)
	pinned := make([]string, 0, len(pinnedRaw))
	for _, p := range pinnedRaw {
		pinned = append(pinned, versionOf(p))
	}
	running, comps := runningVersion(evalStrings(c.running, o), pinned)

	component := c.cfg.Component
	if component == "" {
		component = obj.GetLabels()[labelName]
	}
	if component == "" {
		component = obj.GetName()
	}
	env := ""
	if vals := evalStrings(c.env, o); len(vals) > 0 {
		env = vals[0]
	}

	return &repository.Resource{
		Name:               repository.CustomResourceName(c.cfg.Group, c.cfg.Kind, obj.GetNamespace(), obj.GetName()),
		Kind:               c.cfg.Key(),
		Namespace:          obj.GetNamespace(),
		ObjectName:         obj.GetName(),
		Labels:             obj.GetLabels(),
		Annotations:        obj.GetAnnotations(),
		Components:         comps,
		DesiredVersion:     desired,
		SyncRevision:       running,
		Health:             c.health(o),
		DefaultComponent:   component,
		DefaultEnvironment: env,
		PinnedVersions:     pinned,
	}, nil
}

// NewOperatorApplicationRepository observes one operator application kind.
// An invalid configuration yields a repository that reports the error to
// its store instead of panicking at start.
func NewOperatorApplicationRepository(client dynamic.Interface, cfg config.ApplicationKindConfig) repository.ResourceRepository {
	conv, err := NewOperatorApplicationConverter(cfg)
	if err != nil {
		return &brokenRepository{err: fmt.Errorf("application kind %s: %w", cfg.Key(), err)}
	}
	return NewResourceRepository(ResourceRepositoryParams{
		Client: client,
		GroupVersionResource: schema.GroupVersionResource{
			Group:    cfg.Group,
			Version:  cfg.Version,
			Resource: cfg.Resource,
		},
		ToResource: conv.Convert,
	})
}

type brokenRepository struct{ err error }

func (b *brokenRepository) Watch(_ context.Context, store repository.ResourceStore) {
	store.Error(b.err)
}
