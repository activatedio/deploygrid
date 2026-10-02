package k8s_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// TestWatch_SkipsUnservedResource checks that a repository whose resource the
// API server does not offer never lists it and reports no error; one whose
// resource is served starts normally.
func TestWatch_SkipsUnservedResource(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	scheme := runtime.NewScheme()
	client := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{gvr: "ApplicationList"})
	toResource := func(obj *unstructured.Unstructured) (*repository.Resource, error) {
		return &repository.Resource{Name: obj.GetName(), Kind: repository.KindApplication, ObjectName: obj.GetName()}, nil
	}

	notServed := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{}}
	store := repository.NewRecordingResourceStore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	k8s.NewResourceRepository(k8s.ResourceRepositoryParams{Client: client, Discovery: notServed, GroupVersionResource: gvr, ToResource: toResource}).Watch(ctx, store)
	time.Sleep(300 * time.Millisecond)
	assert.Empty(t, store.GetRecords(), "nothing listed and no error recorded while the CRD is absent")

	served := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{Resources: []*metav1.APIResourceList{{
		GroupVersion: "argoproj.io/v1alpha1",
		APIResources: []metav1.APIResource{{Name: "applications", Kind: "Application", Namespaced: true}},
	}}}}
	store2 := repository.NewRecordingResourceStore()
	k8s.NewResourceRepository(k8s.ResourceRepositoryParams{Client: client, Discovery: served, GroupVersionResource: gvr, ToResource: toResource}).Watch(ctx, store2)
	require.Eventually(t, func() bool {
		recs := store2.GetRecords()
		return len(recs) == 1 && recs[0].EventType == repository.ResourceStoreEventReplace
	}, 5*time.Second, 50*time.Millisecond, "served resource is listed")
}
