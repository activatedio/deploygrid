package k8s

import "fmt"

func ApplicationName(namespace, name string) string {
	return fmt.Sprintf("applications/%s/%s", namespace, name)
}

func DeploymentName(namespace, name string) string {
	return fmt.Sprintf("deployments/%s/%s", namespace, name)
}
