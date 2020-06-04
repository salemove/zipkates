#!/bin/bash
set -euo pipefail

K3D_NAME='zipkin-test'

# Start from a clean slate and ensure everything's cleaned up after the test
function clean_up() {
  echo "Removing k3d cluster if it exists .."
  k3d delete --name="$K3D_NAME" || :
}
trap clean_up EXIT
clean_up

# Start the cluster and set up access
k3d create --wait 300 --image='docker.io/rancher/k3s:v1.17.5-k3s1' --name="$K3D_NAME"
KUBECONFIG="$(k3d get-kubeconfig --name="$K3D_NAME")"
export KUBECONFIG

# Start a zipkin instance with the sidecar and wait for it to be ready
docker build -t salemove/zipkates:build .
k3d import-images --name="$K3D_NAME" salemove/zipkates:build
kubectl apply --wait=true -f test-setup.yml
kubectl -n test-zipkin rollout status deploy/zipkin

# Run the test
kubectl apply --wait=true -f test.yml
kubectl -n test-service wait --for=condition=complete --timeout=120s job/zipkin-client
