package service

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/activatedio/deploygrid/pkg/history"
)

// Reconciler is the single periodic loop: it builds every System, records
// version changes into the history ring and, when a control cluster is
// configured, hands the results to the status writer (which also emits
// Events for the changes).
type Reconciler struct {
	grids    GridService
	ring     *history.Ring
	status   *StatusWriter // nil without a control cluster
	interval time.Duration
	// warmupUntil marks the period after start-up during which cells are
	// seeded rather than diffed, because cluster watches are still syncing.
	warmupUntil time.Time
	cancel      context.CancelFunc
}

// warmup is how long after start-up version changes are not recorded.
const warmup = 45 * time.Second

func (r *Reconciler) run(ctx context.Context) {
	// an early first pass so the ring is seeded and statuses appear quickly
	r.Sync(ctx)
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.Sync(ctx)
		}
	}
}

// Sync runs one reconciliation pass.
func (r *Reconciler) Sync(ctx context.Context) {
	if r.status != nil {
		r.status.Prepare()
	}
	results, err := r.grids.BuildAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("reconcile: build grids")
		return
	}
	now := time.Now()
	for name, res := range results {
		var changes []history.Change
		if now.Before(r.warmupUntil) {
			r.ring.Seed(res.Grid)
		} else {
			changes = r.ring.Observe(res.Grid, now)
		}
		for _, ch := range changes {
			log.Info().Str("system", ch.System).Str("component", ch.Component).Str("environment", ch.Environment).
				Str("from", ch.From).Str("to", ch.To).Msg("version changed")
		}
		if r.status != nil {
			r.status.Apply(name, res, changes, now)
		}
	}
	if r.status != nil {
		r.status.Finish()
	}
}

type ReconcilerParams struct {
	fx.In
	Grids     GridService
	Ring      *history.Ring
	Status    *StatusWriter `optional:"true"`
	Lifecycle fx.Lifecycle
}

func NewReconciler(params ReconcilerParams) *Reconciler {
	r := &Reconciler{
		grids:    params.Grids,
		ring:     params.Ring,
		status:   params.Status,
		interval: 15 * time.Second,
	}
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			r.warmupUntil = time.Now().Add(warmup)
			runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			r.cancel = cancel
			go r.run(runCtx)
			return nil
		},
		OnStop: func(_ context.Context) error {
			if r.cancel != nil {
				r.cancel()
			}
			return nil
		},
	})
	return r
}
