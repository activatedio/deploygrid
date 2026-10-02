package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/deploygrid"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// Version is stamped into heartbeats; set at build time with -ldflags.
var Version = "dev"

// Options configure one collector run.
type Options struct {
	Server            string
	Cluster           string
	Token             string
	Flush             time.Duration
	Heartbeat         time.Duration
	Resources         *repository.Resources
	KubernetesVersion string
	HTTPClient        *http.Client
}

// Collector watches a cluster and pushes observations.
type Collector struct {
	opts  Options
	sinks map[string]*kindSink
	wake  chan struct{}
}

// New prepares a collector; Run starts it.
func New(opts Options) *Collector {
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	if opts.Flush <= 0 {
		opts.Flush = 2 * time.Second
	}
	if opts.Heartbeat <= 0 {
		opts.Heartbeat = 30 * time.Second
	}
	c := &Collector{opts: opts, sinks: map[string]*kindSink{}, wake: make(chan struct{}, 1)}
	for kind := range opts.Resources.ByKind() {
		c.sinks[kind] = newKindSink(kind, c.notify)
	}
	return c
}

func (c *Collector) notify() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// maxBackoff caps the delay between retries after a failed push.
const maxBackoff = time.Minute

// Run watches until the context is cancelled, pushing coalesced changes
// every Flush and a heartbeat every Heartbeat. Failed pushes are retried
// with exponential backoff.
func (c *Collector) Run(ctx context.Context) {
	for kind, repo := range c.opts.Resources.ByKind() {
		repo.Watch(ctx, c.sinks[kind])
	}

	heartbeat := time.NewTicker(c.opts.Heartbeat)
	defer heartbeat.Stop()
	s := &schedule{c: c}

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.wake:
			s.onWake()
		case <-s.flush:
			s.onFlush(ctx)
		case <-heartbeat.C:
			s.onHeartbeat(ctx)
		}
	}
}

// schedule decides when the next push happens: a flush after changes, an
// immediate heartbeat when idle, and backoff-spaced retries after failures.
type schedule struct {
	c       *Collector
	flush   <-chan time.Time
	backoff time.Duration
}

func (s *schedule) arm(d time.Duration) {
	if s.flush == nil {
		s.flush = time.After(d)
	}
}

func (s *schedule) onWake() {
	if s.backoff == 0 {
		s.arm(s.c.opts.Flush)
	}
}

func (s *schedule) onFlush(ctx context.Context) {
	s.flush = nil
	s.backoff = s.c.attempt(ctx, false, s.backoff)
	if s.backoff > 0 {
		s.arm(s.backoff)
	}
}

func (s *schedule) onHeartbeat(ctx context.Context) {
	if s.backoff != 0 {
		return // a retry is already scheduled
	}
	s.backoff = s.c.attempt(ctx, true, s.backoff)
	if s.backoff > 0 {
		s.arm(s.backoff)
	}
}

// attempt pushes once and returns the backoff to wait before the next try:
// zero after success, doubled (up to maxBackoff) after a failure.
func (c *Collector) attempt(ctx context.Context, force bool, backoff time.Duration) time.Duration {
	err := c.push(ctx, force)
	if err == nil {
		return 0
	}
	if backoff == 0 {
		backoff = c.opts.Flush
	} else {
		backoff = min(backoff*2, maxBackoff)
	}
	log.Error().Err(err).Dur("retry_in", backoff).Msg("push failed; next push will be a snapshot")
	return backoff
}

// push sends pending batches, or a heartbeat when nothing is pending and
// force is set. On failure the batches are restored so the next push is a
// snapshot.
func (c *Collector) push(ctx context.Context, force bool) error {
	var batches []batch
	for _, kind := range sortedKinds(c.sinks) {
		if b, ok := c.sinks[kind].take(); ok {
			batches = append(batches, b)
		}
	}
	if len(batches) == 0 && !force {
		return nil
	}

	obs := c.observation(batches)
	resp, err := c.send(ctx, obs)
	if err != nil {
		for _, b := range batches {
			c.sinks[b.kind].restore()
		}
		return err
	}
	if resp.Resync {
		log.Info().Msg("server requested a resync")
		for _, s := range c.sinks {
			s.RequestFull()
		}
		c.notify()
	}
	return nil
}

// observation turns batches into one wire message. Snapshot batches and
// delta batches cannot share a message, so when both are present the deltas
// are promoted to snapshots (the mirror has the complete state anyway).
func (c *Collector) observation(batches []batch) *deploygrid.Observation {
	obs := &deploygrid.Observation{
		Cluster: c.opts.Cluster,
		Heartbeat: deploygrid.ObservationHeartbeat{
			CollectorVersion:  Version,
			KubernetesVersion: c.opts.KubernetesVersion,
			SentAt:            time.Now(),
		},
	}
	anyFull := false
	for _, b := range batches {
		anyFull = anyFull || b.full
		obs.Errors = append(obs.Errors, b.errors...)
	}
	if anyFull {
		obs.Snapshot = true
		for _, b := range batches {
			obs.Kinds = append(obs.Kinds, b.kind)
			if b.full {
				obs.Resources = append(obs.Resources, b.upserts...)
				continue
			}
			// promote: send everything the mirror holds for this kind
			c.sinks[b.kind].RequestFull()
			full, _ := c.sinks[b.kind].take()
			obs.Resources = append(obs.Resources, full.upserts...)
		}
		return obs
	}
	for _, b := range batches {
		obs.Resources = append(obs.Resources, b.upserts...)
		obs.Removed = append(obs.Removed, b.removes...)
	}
	return obs
}

func (c *Collector) send(ctx context.Context, obs *deploygrid.Observation) (*deploygrid.ObservationResponse, error) {
	body, err := json.Marshal(obs)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(c.opts.Server, "/") + "/observations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.opts.Token)

	res, err := c.opts.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("server returned %d: %s", res.StatusCode, strings.TrimSpace(string(payload)))
	}
	out := &deploygrid.ObservationResponse{}
	if err := json.Unmarshal(payload, out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	log.Debug().Int("accepted", out.Accepted).Bool("snapshot", obs.Snapshot).Msg("pushed observation")
	return out, nil
}

func sortedKinds(m map[string]*kindSink) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// deterministic order keeps logs and tests stable
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// RunFromConfig builds the Kubernetes client from the configuration and runs
// a collector until ctx is cancelled.
func RunFromConfig(ctx context.Context, cfg *config.CollectorConfig) error {
	if err := cfg.ValidateForRun(); err != nil {
		return err
	}
	token := cfg.Token
	if cfg.TokenFile != "" {
		bs, err := os.ReadFile(cfg.TokenFile)
		if err != nil {
			return fmt.Errorf("read token file: %w", err)
		}
		token = strings.TrimSpace(string(bs))
	}
	if token == "" {
		return errors.New("empty collector token")
	}

	restConfig, err := targetRestConfig(cfg)
	if err != nil {
		return err
	}
	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	k8sVersion := ""
	if dc, err := discovery.NewDiscoveryClientForConfig(restConfig); err == nil {
		if v, err := dc.ServerVersion(); err == nil {
			k8sVersion = v.GitVersion
		}
	}

	log.Info().Str("server", cfg.Server).Str("cluster", cfg.Cluster).Str("kubernetes", k8sVersion).Msg("starting collector")
	New(Options{
		Server:            cfg.Server,
		Cluster:           cfg.Cluster,
		Token:             token,
		Flush:             time.Duration(cfg.FlushSeconds) * time.Second,
		Heartbeat:         time.Duration(cfg.HeartbeatSeconds) * time.Second,
		Resources:         k8s.NewResources(client),
		KubernetesVersion: k8sVersion,
	}).Run(ctx)
	return nil
}

func targetRestConfig(cfg *config.CollectorConfig) (*rest.Config, error) {
	if cfg.KubeConfigPath == "" {
		return rest.InClusterConfig()
	}
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.KubeConfigPath}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: cfg.Context}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}
