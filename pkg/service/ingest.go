package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/repository"
)

// ErrUnauthorized is returned when a bearer token matches no cluster.
var ErrUnauthorized = errors.New("unauthorized")

// TokenResolver maps a collector bearer token to the cluster it may report
// for.
type TokenResolver interface {
	ClusterForToken(ctx context.Context, token string) (string, error)
}

// ObservationService records what collectors push.
type ObservationService interface {
	// Authenticate returns the cluster name for a bearer token.
	Authenticate(ctx context.Context, token string) (string, error)
	Ingest(ctx context.Context, cluster string, obs *deploygrid.Observation) (*deploygrid.ObservationResponse, error)
}

type observationService struct {
	registry  *SourceRegistry
	resolvers []TokenResolver
}

func (s *observationService) Authenticate(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrUnauthorized
	}
	for _, r := range s.resolvers {
		cluster, err := r.ClusterForToken(ctx, token)
		if err != nil && !errors.Is(err, ErrUnauthorized) {
			return "", err
		}
		if cluster != "" {
			return cluster, nil
		}
	}
	return "", ErrUnauthorized
}

func (s *observationService) Ingest(_ context.Context, cluster string, obs *deploygrid.Observation) (*deploygrid.ObservationResponse, error) {
	resp := &deploygrid.ObservationResponse{Cluster: cluster}

	var err error
	if obs.Snapshot {
		err = s.ingestSnapshot(cluster, obs, resp)
	} else {
		err = s.ingestDelta(cluster, obs, resp)
	}
	if err != nil {
		return nil, err
	}

	s.registry.SetPushErrors(cluster, obs.Errors)
	s.registry.RecordHeartbeat(cluster, Heartbeat{
		LastSeen:          time.Now(),
		CollectorVersion:  obs.Heartbeat.CollectorVersion,
		KubernetesVersion: obs.Heartbeat.KubernetesVersion,
		Pushed:            true,
	})
	return resp, nil
}

// ingestSnapshot replaces every resource of the listed kinds.
func (s *observationService) ingestSnapshot(cluster string, obs *deploygrid.Observation, resp *deploygrid.ObservationResponse) error {
	if len(obs.Kinds) == 0 {
		return fmt.Errorf("snapshot without kinds")
	}
	byKind := map[string][]*repository.Resource{}
	for _, k := range obs.Kinds {
		byKind[k] = nil
	}
	for _, r := range obs.Resources {
		if _, ok := byKind[r.Kind]; !ok {
			return fmt.Errorf("snapshot resource %s has kind %q not listed in kinds", r.Name, r.Kind)
		}
		byKind[r.Kind] = append(byKind[r.Kind], r)
	}
	for kind, rs := range byKind {
		st, _ := s.registry.Store(cluster, kind)
		if err := st.Replace(rs); err != nil {
			return err
		}
		resp.Accepted += len(rs)
	}
	return nil
}

// ingestDelta upserts and removes individual resources. A delta for a kind
// never seen in full means the server probably restarted: it is recorded
// anyway and a resync is requested.
func (s *observationService) ingestDelta(cluster string, obs *deploygrid.Observation, resp *deploygrid.ObservationResponse) error {
	for _, r := range obs.Resources {
		st, existed := s.registry.Store(cluster, r.Kind)
		if !existed {
			resp.Resync = true
		}
		if err := st.Add(r); err != nil {
			return err
		}
		resp.Accepted++
	}
	for _, r := range obs.Removed {
		st, existed := s.registry.Store(cluster, r.Kind)
		if !existed {
			resp.Resync = true
			continue
		}
		if err := st.Delete(r); err != nil {
			return err
		}
		resp.Accepted++
	}
	return nil
}

// staticTokenResolver maps the tokens configured on clusters.
type staticTokenResolver struct {
	byToken map[string]string
}

func (r *staticTokenResolver) ClusterForToken(_ context.Context, token string) (string, error) {
	return r.byToken[token], nil
}

// NewStaticTokenResolver builds a resolver from token → cluster pairs.
func NewStaticTokenResolver(tokens map[string]string) TokenResolver {
	return &staticTokenResolver{byToken: tokens}
}

// NewConfigTokenResolver maps the static tokens of agent-mode clusters in
// the configuration.
func NewConfigTokenResolver(clusters *config.ClustersConfig) TokenResolver {
	tokens := map[string]string{}
	for _, c := range clusters.Clusters {
		if c.EffectiveMode() == config.ClusterModeAgent && c.Token != "" {
			tokens[c.Token] = c.Name
		}
	}
	return NewStaticTokenResolver(tokens)
}

type ObservationServiceParams struct {
	fx.In
	Registry  *SourceRegistry
	Resolvers []TokenResolver `group:"token_resolvers"`
}

func NewObservationService(params ObservationServiceParams) ObservationService {
	return &observationService{
		registry:  params.Registry,
		resolvers: params.Resolvers,
	}
}
