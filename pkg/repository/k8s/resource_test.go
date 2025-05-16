package k8s_test

import (
	"context"
	"fmt"
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"testing"
	"time"
)

func TestResourceRepository_Watch(t *testing.T) {

	type s struct {
		arrange func() (context.Context, context.CancelFunc, dynamic.Interface, schema.GroupVersionResource, k8s.ToResource, repository.RecordingResourceStore)
		assert  func(context.Context, context.CancelFunc, repository.RecordingResourceStore)
	}

	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "namespaces",
	}
	gvrInvalid := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "invalids",
	}

	cases := map[string]s{
		"error": {
			arrange: func() (context.Context, context.CancelFunc, dynamic.Interface, schema.GroupVersionResource, k8s.ToResource, repository.RecordingResourceStore) {
				ctx, cancel := context.WithCancel(context.Background())
				return ctx, cancel, opsCl, gvrInvalid, func(obj *unstructured.Unstructured) (*repository.Resource, error) {
					panic("should not execute")
				}, repository.NewRecordingResourceStore()
			},
			assert: func(ctx context.Context, cancel context.CancelFunc, store repository.RecordingResourceStore) {

				assert.EventuallyWithT(t, func(c *assert.CollectT) {

					recs := store.GetRecords()

					assert.True(c, len(recs) > 1)
					for _, rec := range recs {
						assert.Equal(c, repository.ResourceStoreEventError, rec.EventType)
						assert.Nil(c, rec.Resource)
						assert.Nil(c, rec.ResourceArray)
						assert.NotNil(c, rec.Error)
					}

				}, 10*time.Second, 500*time.Millisecond)

				cancel()
			},
		},
		"default": {
			arrange: func() (context.Context, context.CancelFunc, dynamic.Interface, schema.GroupVersionResource, k8s.ToResource, repository.RecordingResourceStore) {
				ctx, cancel := context.WithCancel(context.Background())
				return ctx, cancel, opsCl, gvr, func(obj *unstructured.Unstructured) (*repository.Resource, error) {

					ns := &corev1.Namespace{}

					err := k8s.DecodeMap(obj.Object, ns)

					if err != nil {
						return nil, err
					}

					return &repository.Resource{
						Name: fmt.Sprintf("namespaces/%s", ns.Name),
						Components: []repository.Component{
							{
								DisplayName: ns.Name,
								Version:     "0",
							},
						},
					}, nil

				}, repository.NewRecordingResourceStore()
			},
			assert: func(ctx context.Context, cancel context.CancelFunc, store repository.RecordingResourceStore) {

				assert.EventuallyWithT(t, func(c *assert.CollectT) {

					rec := store.GetRecords()

					assert.Len(c, rec, 1)
					assert.Equal(c, repository.ResourceStoreEventReplace, rec[0].EventType)
					assert.Len(c, rec[0].ResourceArray, 6)

				}, time.Second, 200*time.Millisecond)

				nsName := "unit"
				nsMap, err := k8s.EncodeToMap(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: nsName,
					},
				})
				ns := &unstructured.Unstructured{
					Object: nsMap,
				}
				util.Check(err)

				_, err = opsCl.Resource(gvr).Create(context.Background(), ns, metav1.CreateOptions{})
				util.Check(err)

				assert.EventuallyWithT(t, func(c *assert.CollectT) {

					rec := store.GetRecords()

					assert.Len(c, rec, 2)
					assert.Equal(c, repository.ResourceStoreEventAdd, rec[1].EventType)
					assert.Equal(c, []repository.Component{
						{
							DisplayName: nsName,
							Version:     "0",
						},
					}, rec[1].Resource.Components)

				}, time.Second, 200*time.Millisecond)

				err = opsCl.Resource(gvr).Delete(context.Background(), nsName, metav1.DeleteOptions{})

				assert.EventuallyWithT(t, func(c *assert.CollectT) {

					rec := store.GetRecords()

					l := len(rec)
					if l != 5 {
						assert.Fail(c, fmt.Sprintf("len of %d not yet 5", l))
						return
					}

					assert.Len(c, rec, 5)
					assert.Equal(c, repository.ResourceStoreEventModify, rec[2].EventType)
					assert.Equal(c, repository.ResourceStoreEventModify, rec[3].EventType)
					assert.Equal(c, repository.ResourceStoreEventDelete, rec[4].EventType)
					assert.Equal(c, []repository.Component{
						{
							DisplayName: nsName,
							Version:     "0",
						},
					}, rec[4].Resource.Components)

				}, 10*time.Second, 200*time.Millisecond)

				cancel()
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {

			ctx, cancel, cl, gvr, toR, store := v.arrange()

			unit := k8s.NewResourceRepository(k8s.ResourceRepositoryParams{
				Client:               cl,
				GroupVersionResource: gvr,
				ToResource:           toR,
			})

			unit.Watch(ctx, store)

			v.assert(ctx, cancel, store)
		})
	}
}
