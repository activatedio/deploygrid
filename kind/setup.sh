#!/bin/bash

set -e

dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

kind create cluster --name ops-cluster-1  --config ${dir}/ops-cluster-1.yaml
kind create cluster --name app-cluster-1  --config ${dir}/app-cluster-1.yaml
kind create cluster --name app-cluster-2  --config ${dir}/app-cluster-2.yaml

# Setop ops cluster
kubectl --context kind-ops-cluster-1 apply -f ${dir}/argocd-application-crds.yaml 
sleep 2
kubectl --context kind-ops-cluster-1 apply -f ${dir}/applications.yaml
