package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	apiinfraconfig "github.com/activatedio/deploygrid/pkg/apiinfra/config"
	"github.com/activatedio/deploygrid/pkg/apis/deploygrid.activated.io/v1alpha1"
	"github.com/activatedio/deploygrid/pkg/collector"
	"github.com/activatedio/deploygrid/pkg/config"
	deploygridfx "github.com/activatedio/deploygrid/pkg/fx"
	"github.com/activatedio/deploygrid/pkg/repository/k8s"
	"github.com/activatedio/deploygrid/pkg/runner"
	"github.com/activatedio/deploygrid/pkg/service"
)

func json(r *resty.Request) *resty.Request {
	return r.SetHeader("Content-Type", "application/json")
}

func waitForHealth(url string) {

	for i := 0; i < 30; i++ {
		resp, err := resty.New().SetBaseURL(url).R().Get("/api/healthz")
		if err == nil && resp.IsSuccess() {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	panic("health check timed out")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func doTest(t *testing.T, configPath string, callback func(t *testing.T, baseURL string)) {

	fxctx := context.Background()

	var baseURL string

	m := config.NewMainConfig(apiinfraconfig.NewConfig(configPath))

	app := fx.New(deploygridfx.Index(m),
		fx.Invoke(func(server *runner.RunningServer) {
			baseURL = fmt.Sprintf("http://%s:%d", server.Host, server.Port)
			log.Info().Str("host", server.Host).Int("port", server.Port).Msg("Starting server")
		}))
	check(app.Start(fxctx))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = app.Stop(ctx)
	})

	waitForHealth(baseURL)

	callback(t, baseURL)

}

type ErrorResponse struct {
	Error string `json:"error"`
}

func kubeClients(t *testing.T, kubeconfig string) (kubernetes.Interface, dynamic.Interface) {
	t.Helper()
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)
	cs, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)
	dc, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
	return cs, dc
}

// waitForToken reads the collector token the server generates for an
// agent-mode Cluster.
func waitForToken(t *testing.T, kubeconfig, namespace, secretName string) string {
	t.Helper()
	cs, _ := kubeClients(t, kubeconfig)
	var token string
	require.Eventually(t, func() bool {
		s, err := cs.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		token = string(s.Data[service.DefaultTokenKey])
		return token != ""
	}, 30*time.Second, 500*time.Millisecond, "token secret %s/%s", namespace, secretName)
	return token
}

func getCluster(t *testing.T, kubeconfig, namespace, name string) *v1alpha1.Cluster {
	t.Helper()
	_, dc := kubeClients(t, kubeconfig)
	u, err := dc.Resource(v1alpha1.SchemeGroupVersion.WithResource(v1alpha1.ClusterResourceName)).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	out := &v1alpha1.Cluster{}
	require.NoError(t, k8s.DecodeMap(u.Object, out))
	return out
}

// startCollector runs an in-process collector against the given cluster for
// the duration of the test.
func startCollector(t *testing.T, server, cluster, token, kubeconfig string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		err := collector.RunFromConfig(ctx, &config.CollectorConfig{
			Server:           server,
			Cluster:          cluster,
			Token:            token,
			KubeConfigPath:   kubeconfig,
			FlushSeconds:     1,
			HeartbeatSeconds: 5,
		})
		if err != nil {
			log.Error().Err(err).Msg("collector stopped")
		}
	}()
}
