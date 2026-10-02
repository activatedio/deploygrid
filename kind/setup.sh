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

# deploygrid custom resources live on the ops (control) cluster
kubectl --context kind-ops-cluster-1 apply -f ${dir}/../crds
sleep 2
kubectl --context kind-ops-cluster-1 apply -f ${dir}/systems.yaml

# Operator-installed application on app-cluster-1
kubectl --context kind-app-cluster-1 apply -f ${dir}/operator-crd.yaml
kubectl --context kind-app-cluster-1 wait --for condition=established --timeout=60s crd/ritesuites.platform.ritesuite.com
kubectl --context kind-app-cluster-1 apply -f ${dir}/operator.yaml
uid=$(kubectl --context kind-app-cluster-1 get ritesuite dev -n ritesuite -o jsonpath='{.metadata.uid}')
for d in dev-management dev-console; do
  kubectl --context kind-app-cluster-1 patch deployment $d -n ritesuite --type=merge -p \
    "{\"metadata\":{\"ownerReferences\":[{\"apiVersion\":\"platform.ritesuite.com/v1alpha1\",\"kind\":\"RiteSuite\",\"name\":\"dev\",\"uid\":\"$uid\",\"controller\":true}]}}"
done

# Applications
kubectl --context kind-app-cluster-1 apply -f ${dir}/deployments-dev.yaml
kubectl --context kind-app-cluster-1 apply -f ${dir}/deployments-qa.yaml
kubectl --context kind-app-cluster-2 apply -f ${dir}/deployments-stage.yaml
