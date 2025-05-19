#!/bin/bash

set -e

kind delete cluster --name ops-cluster-1
kind delete cluster --name app-cluster-1
kind delete cluster --name app-cluster-2
