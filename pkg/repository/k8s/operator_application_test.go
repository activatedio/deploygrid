package k8s_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"

	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

// riteSuite is trimmed from a live `kubectl get rsuite -o json` on the
// kind-ritesuite cluster (2026-10-01): a suite mid-rollout with one failed
// migration job.
const riteSuite = `{
 "apiVersion": "platform.ritesuite.com/v1alpha1",
 "kind": "RiteSuite",
 "metadata": {"name": "dev", "namespace": "ritesuite", "uid": "2cf0fd51-50b4-4c06-9b16-f584f20fdacf"},
 "spec": {"preset": "dev", "registry": "registry.ops.quarterhill.com/roadside/next", "version": "0.2.0"},
 "status": {
  "conditions": [
   {"type": "Available", "status": "False", "reason": "ComponentsNotReady", "message": "14/16 components ready"},
   {"type": "Ready", "status": "False", "reason": "JobFailed", "message": "ai-assistant: Job failed; the running version is kept"},
   {"type": "Progressing", "status": "True", "reason": "AwaitingJobs", "message": "rollout held for ai-assistant until their Jobs succeed"},
   {"type": "Degraded", "status": "True", "reason": "JobFailed", "message": "ai-assistant: Job failed"}
  ],
  "observedGeneration": 59, "readyComponents": 14, "totalComponents": 16,
  "ingress": {"ready": true, "url": "https://ritesuite.localtest/"},
  "components": [
   {"name": "alarm-interface", "image": "registry.ops.quarterhill.com/roadside/next/alarm-interface:0.2.0", "ready": true, "replicas": 1, "readyReplicas": 1},
   {"name": "management", "image": "registry.ops.quarterhill.com/roadside/next/management:0.2.0", "ready": true, "replicas": 1, "readyReplicas": 1}
  ]
 }
}`

func riteSuiteKind() config.ApplicationKindConfig {
	return config.ApplicationKindConfig{
		Group: "platform.ritesuite.com", Version: "v1alpha1", Resource: "ritesuites", Kind: "RiteSuite",
		Component:          "ritesuite",
		RunningVersionPath: "{.status.components[*].image}",
	}
}

func TestOperatorApplicationConverter(t *testing.T) {
	obj := &unstructured.Unstructured{}
	require.NoError(t, json.Unmarshal([]byte(riteSuite), &obj.Object))

	conv, err := k8s.NewOperatorApplicationConverter(riteSuiteKind())
	require.NoError(t, err)
	res, err := conv.Convert(obj)
	require.NoError(t, err)

	a := assert.New(t)
	a.Equal("customresources/platform.ritesuite.com/RiteSuite/ritesuite/dev", res.Name)
	a.Equal("application/platform.ritesuite.com/ritesuites", res.Kind)
	a.True(repository.IsOperatorApplication(res.Kind))
	a.Equal("0.2.0", res.DesiredVersion, "default desired path is .spec.version")
	a.Equal("0.2.0", res.SyncRevision, "all reported images agree on the tag")
	a.Equal(repository.HealthDegraded, res.Health, "Degraded=True wins over Progressing")
	a.Equal("ritesuite", res.DefaultComponent)
	require.Len(t, res.Components, 2)
	a.Equal("alarm-interface", res.Components[0].Name)
	a.Equal("0.2.0", res.Components[0].Version)

	// a mid-rollout suite reports two tags: no single running version
	mixed := &unstructured.Unstructured{}
	require.NoError(t, json.Unmarshal([]byte(riteSuite), &mixed.Object))
	comps, _, _ := unstructured.NestedSlice(mixed.Object, "status", "components")
	comps[1].(map[string]any)["image"] = "registry.ops.quarterhill.com/roadside/next/management:0.3.0"
	require.NoError(t, unstructured.SetNestedSlice(mixed.Object, comps, "status", "components"))
	res, err = conv.Convert(mixed)
	require.NoError(t, err)
	a.Empty(res.SyncRevision)
	a.Len(res.Components, 2)

	// a pinned service: its tag is set aside, the rest agree
	pinnedSuite := &unstructured.Unstructured{}
	require.NoError(t, json.Unmarshal([]byte(riteSuite), &pinnedSuite.Object))
	comps, _, _ = unstructured.NestedSlice(pinnedSuite.Object, "status", "components")
	comps[1].(map[string]any)["image"] = "registry.ops.quarterhill.com/roadside/next/management:0.1.9"
	require.NoError(t, unstructured.SetNestedSlice(pinnedSuite.Object, comps, "status", "components"))
	require.NoError(t, unstructured.SetNestedField(pinnedSuite.Object, "0.1.9", "spec", "services", "management", "image", "tag"))
	pinnedCfg := riteSuiteKind()
	pinnedCfg.PinnedVersionsPath = "{.spec.services.*.image.tag}"
	convPinned, err := k8s.NewOperatorApplicationConverter(pinnedCfg)
	require.NoError(t, err)
	res, err = convPinned.Convert(pinnedSuite)
	require.NoError(t, err)
	a.Equal("0.2.0", res.SyncRevision, "the pinned tag does not break agreement")
	a.Equal([]string{"0.1.9"}, res.PinnedVersions)

	// healthy suite, environment from a path, component from the name label
	healthy := &unstructured.Unstructured{}
	require.NoError(t, json.Unmarshal([]byte(riteSuite), &healthy.Object))
	require.NoError(t, unstructured.SetNestedSlice(healthy.Object, []any{
		map[string]any{"type": "Ready", "status": "True"},
	}, "status", "conditions"))
	healthy.SetLabels(map[string]string{"app.kubernetes.io/name": "suite", "env": "qa"})
	conv2, err := k8s.NewOperatorApplicationConverter(config.ApplicationKindConfig{
		Group: "platform.ritesuite.com", Version: "v1alpha1", Resource: "ritesuites", Kind: "RiteSuite",
		EnvironmentPath: ".metadata.labels.env",
	})
	require.NoError(t, err)
	res, err = conv2.Convert(healthy)
	require.NoError(t, err)
	a.Equal(repository.HealthHealthy, res.Health)
	a.Equal("suite", res.DefaultComponent)
	a.Equal("qa", res.DefaultEnvironment)

	_, err = k8s.NewOperatorApplicationConverter(config.ApplicationKindConfig{
		Group: "g", Version: "v", Resource: "r", Kind: "K", DesiredVersionPath: "{.spec[",
	})
	require.Error(t, err, "bad JSONPath is rejected up front")
}

func TestParentOf_OwnerReference(t *testing.T) {
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: "dev-management", Namespace: "ritesuite",
		Labels: map[string]string{"app.kubernetes.io/managed-by": "ritesuite-operator", "app.kubernetes.io/instance": "dev", "app.kubernetes.io/version": "0.2.0"},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "platform.ritesuite.com/v1alpha1", Kind: "RiteSuite", Name: "dev", UID: "x", Controller: ptr.To(true),
		}},
	}}
	res := k8s.DeploymentResource(dep)
	assert.Equal(t, "customresources/platform.ritesuite.com/RiteSuite/ritesuite/dev", res.Parent)
	assert.Equal(t, "0.2.0", res.StampedVersion)

	// a ReplicaSet owner says nothing; the Helm label still applies
	helm := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: "app", Namespace: "dev-app-a",
		Labels:          map[string]string{"app.kubernetes.io/managed-by": "Helm", "app.kubernetes.io/instance": "dev-app-a"},
		OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "rs", Controller: ptr.To(true)}},
	}}
	assert.Equal(t, "applications/dev-app-a", k8s.DeploymentResource(helm).Parent)
}
