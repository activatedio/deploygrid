package k8s_test

import (
	"github.com/activatedio/deploygrid/pkg/apiinfra/util"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"testing"
)

var (
	opsCl   dynamic.Interface
	apps1Cl dynamic.Interface
)

func TestMain(m *testing.M) {

	makeClient := func(path string) dynamic.Interface {
		cfg, err := clientcmd.BuildConfigFromFlags("", path)
		util.Check(err)
		cl, err := dynamic.NewForConfig(cfg)
		util.Check(err)
		return cl
	}

	opsCl = makeClient("../../../.kind/kubeconfig-ops-cluster-1.yaml")
	apps1Cl = makeClient("../../../.kind/kubeconfig-app-cluster-1.yaml")

	os.Exit(m.Run())

}
