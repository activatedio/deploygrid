#!/bin/bash

set -e


dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

cfgpath=$dir/../.kind

mkdir $cfgpath || true

makecluster() {
  name=$1
	kind create cluster --name $name  --config ${dir}/$name.yaml
	kind get kubeconfig --name $name > $cfgpath/kubeconfig-$name.yaml
}

makecluster ops-cluster-1
makecluster app-cluster-1
makecluster app-cluster-2

# Setop ops cluster
kubectl --context kind-ops-cluster-1 apply -f ${dir}/argocd-application-crds.yaml 
sleep 2
kubectl --context kind-ops-cluster-1 apply -f ${dir}/applications.yaml

# Applications
kubectl --context kind-app-cluster-1 apply -f ${dir}/deployments-dev.yaml
kubectl --context kind-app-cluster-1 apply -f ${dir}/deployments-qa.yaml
kubectl --context kind-app-cluster-2 apply -f ${dir}/deployments-stage.yaml
