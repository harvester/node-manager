#!/bin/bash -e

TOP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/" &> /dev/null && pwd )"

wait_nm_ready() {
  local timeout=600 # 10 minutes in seconds
  local interval=10  # Check every 10 seconds
  local elapsed=0

  while [ $elapsed -lt $timeout ]; do
    running_num=$(kubectl get ds harvester-node-manager -n harvester-system -o 'jsonpath={.status.numberReady}')
    if [[ $running_num -eq ${cluster_nodes} ]]; then
      echo "harvester-node-manager pods are ready!"
      return 0
    fi
    echo "harvester-node-manager pods are not ready, sleeping for $interval seconds..."
    sleep $interval
    elapsed=$((elapsed + interval))
  done

  echo "Timeout reached: harvester-node-manager pods are not ready after 10 minutes."
  return 1
}

ensure_command() {
  local cmd=$1
  if ! which $cmd &> /dev/null; then
    echo 1
    return
  fi
  echo 0
}

if [ -z $1 ]; then
  echo "vagrant-rancherd dir is not set."
  exit 1
fi
VAGRANT_RANCHERD_DIR=$1
export KUBECONFIG=$VAGRANT_RANCHERD_DIR/kubeconfig

if [[ $(ensure_command helm) -eq 1 ]]; then
  echo "no helm, try to curl..."
  curl -O https://get.helm.sh/helm-v3.9.4-linux-amd64.tar.gz
  tar -zxvf helm-v3.9.4-linux-amd64.tar.gz
  HELM=$TOP_DIR/linux-amd64/helm
  $HELM version
else
  echo "Get helm, version info as below"
  HELM=$(which helm)
  $HELM version
fi
