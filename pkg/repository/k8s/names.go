package k8s

import "fmt"

func ApplicationName(name string) string {
	return fmt.Sprintf("applications/%s", name)
}

func DeploymentName(namespace, name string) string {
	return fmt.Sprintf("namespaces/%s/deployments/%s", namespace, name)
}
