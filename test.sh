#!/bin/bash
set -euo pipefail

k3d delete --keep-registry-volume || :
k3d create --enable-registry --wait 300

KUBECONFIG="$(k3d get-kubeconfig --name='k3s-default')"
export KUBECONFIG

docker build -t registry.local:5000/proxy:latest .
docker push registry.local:5000/proxy:latest
kubectl apply --wait=true -f setup.yml
kubectl -n test-zipkin rollout status deploy/zipkin
kubectl apply --wait=true -f test.yml
