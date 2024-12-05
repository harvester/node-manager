#!/bin/bash -e

TOP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/" &> /dev/null && pwd )"
source $TOP_DIR/helper.sh

pushd $TOP_DIR

cluster_nodes=$(yq -e e '.cluster_size' $VAGRANT_RANCHERD_DIR/settings.yaml)
echo "cluster nodes: $cluster_nodes"

$HELM pull harvester-node-manager --repo https://charts.harvesterhci.io --untar
$HELM install harvester-node-manager ./harvester-node-manager --create-namespace -n harvester-system

if wait_nm_ready; then
  echo "harvester-node-manager is ready"
else
  echo "harvester-node-manager pods failed to become ready within 10 minutes."
  exit 1
fi

# cleanup downloaded files
rm -rf harvester-node-manager*

popd
