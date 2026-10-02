package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/config"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
)

const (
	// DefaultTokenKey is the Secret key holding a collector token.
	DefaultTokenKey = "token"
	// tokenSecretSuffix names the Secret generated for an agent-mode Cluster
	// that declares no tokenSecretRef.
	tokenSecretSuffix = "-collector-token"

	labelTokenCluster = "deploygrid.activated.io/cluster"
)

// TokenSecretRef returns the Secret reference holding the collector token of
// an agent-mode Cluster, applying defaults.
func TokenSecretRef(c *v1alpha1.Cluster) (name, key string, ok bool) {
	if c.Spec.Collection.Mode != "" && c.Spec.Collection.Mode != v1alpha1.ClusterCollectionModeAgent {
		return "", "", false
	}
	if ref := c.Spec.Collection.TokenSecretRef; ref != nil && ref.Name != "" {
		key = ref.Key
		if key == "" {
			key = DefaultTokenKey
		}
		return ref.Name, key, true
	}
	return c.Name + tokenSecretSuffix, DefaultTokenKey, true
}

// controlTokenResolver matches bearer tokens against the Secrets referenced
// by agent-mode Cluster resources in the control namespace.
type controlTokenResolver struct {
	controllers *k8s.Controllers
	namespace   string
}

func (r *controlTokenResolver) ClusterForToken(_ context.Context, token string) (string, error) {
	clusters, err := r.controllers.ClustersCache.List(r.namespace, labels.Everything())
	if err != nil {
		return "", err
	}
	for _, c := range clusters {
		name, key, ok := TokenSecretRef(c)
		if !ok {
			continue
		}
		secret, err := r.controllers.SecretsCache.Get(r.namespace, name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return "", err
		}
		if string(secret.Data[key]) == token {
			return c.Name, nil
		}
	}
	return "", nil
}

func NewControlTokenResolver(controllers *k8s.Controllers, control *config.ControlConfig) TokenResolver {
	return &controlTokenResolver{controllers: controllers, namespace: control.Namespace}
}

// EnsureTokenSecrets creates the default token Secret for every agent-mode
// Cluster that references none and whose default Secret does not exist, so
// an operator can read the token back after applying the Cluster.
func EnsureTokenSecrets(controllers *k8s.Controllers, namespace string) error {
	clusters, err := controllers.ClustersCache.List(namespace, labels.Everything())
	if err != nil {
		return err
	}
	for _, c := range clusters {
		name, key, ok := TokenSecretRef(c)
		if !ok || c.Spec.Collection.TokenSecretRef != nil {
			continue
		}
		if _, err := controllers.SecretsCache.Get(namespace, name); err == nil {
			continue
		} else if !apierrors.IsNotFound(err) {
			return err
		}
		token, err := randomToken()
		if err != nil {
			return err
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    map[string]string{labelTokenCluster: c.Name},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "Cluster",
					Name:       c.Name,
					UID:        c.UID,
				}},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: map[string]string{key: token},
		}
		if _, err := controllers.Secrets.Create(secret); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create token secret for cluster %s: %w", c.Name, err)
		}
		log.Info().Str("cluster", c.Name).Str("secret", name).Msg("generated collector token")
	}
	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
