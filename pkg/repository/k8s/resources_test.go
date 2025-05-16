package k8s_test

import (
	"context"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/dynamic"
	"testing"
	"time"
)

func TestResources(t *testing.T) {

	type s struct {
		arrange func() (dynamic.Interface, func(p dynamic.Interface) repository.ResourceRepository)
		assert  func(cancel context.CancelFunc, store repository.RecordingResourceStore)
	}

	cases := map[string]s{
		"applications": {
			arrange: func() (dynamic.Interface, func(p dynamic.Interface) repository.ResourceRepository) {
				return opsCl, k8s.NewApplicationRepository
			},
			assert: func(cancel context.CancelFunc, store repository.RecordingResourceStore) {

				assert.EventuallyWithT(t, func(c *assert.CollectT) {

					recs := store.GetRecords()

					assert.Len(c, recs, 1)

					for _, rec := range recs {
						assert.Equal(t, repository.ResourceStoreEventReplace, rec.EventType)
						assert.Len(t, rec.ResourceArray, 6)
					}

				}, 5*time.Second, 500*time.Millisecond)

				cancel()
			},
		},
		"deployments": {
			arrange: func() (dynamic.Interface, func(p dynamic.Interface) repository.ResourceRepository) {
				return apps1Cl, k8s.NewDeploymentRepository
			},
			assert: func(cancel context.CancelFunc, store repository.RecordingResourceStore) {

				assert.EventuallyWithT(t, func(c *assert.CollectT) {

					recs := store.GetRecords()

					assert.Len(c, recs, 1)

					for _, rec := range recs {
						assert.Equal(t, repository.ResourceStoreEventReplace, rec.EventType)
						assert.True(t, len(rec.ResourceArray) > 6)
					}

				}, 5*time.Second, 500*time.Millisecond)

				time.Sleep(5 * time.Second)
				cancel()
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			cl, ctor := v.arrange()

			store := repository.NewRecordingResourceStore()
			ctx, cancel := context.WithCancel(context.Background())

			unit := ctor(cl)

			unit.Watch(ctx, store)

			v.assert(cancel, store)
		})
	}
}
