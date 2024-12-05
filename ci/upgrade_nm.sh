#!/bin/bash -e

TOP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/" &> /dev/null && pwd )"
source $TOP_DIR/../scripts/version
source $TOP_DIR/helper.sh

pushd $TOP_DIR

cluster_nodes=$(yq -e e '.cluster_size' $VAGRANT_RANCHERD_DIR/settings.yaml)
echo "cluster nodes: $cluster_nodes"

OVERRIDE_CONTENT=$(cat <<EOF
image:
  repository: "ttl.sh/node-manager-${COMMIT}"
  tag: "1h"
webhook:
  image:
    repository: "ttl.sh/node-manager-webhook-${COMMIT}"
    tag: "1h"
EOF
)
echo "node-manager override content:"
echo "$OVERRIDE_CONTENT"
echo "$OVERRIDE_CONTENT" > nm-override.yaml

$HELM upgrade -f nm-override.yaml harvester-node-manager ../charts/harvester-node-manager/ -n harvester-system

sleep 30 # wait 30 seconds for node manager respawn pods
if wait_nm_ready; then
  echo "harvester-node-manager is ready"
else
  echo "harvester-node-manager pods failed to become ready within 10 minutes."
  exit 1
fi

# check image
pod_name=$(kubectl get pods -n harvester-system |grep Running |grep ^harvester-node-manager|head -n1 |awk '{print $1}')
container_img=$(kubectl get pods ${pod_name} -n harvester-system -o yaml |yq -e .spec.containers[0].image |tr ":" \n)
yaml_img=$(yq -e .image.repository nm-override.yaml)
if grep -q ${yaml_img} <<< ${container_img}; then
  echo "Image is equal: ${yaml_img}"
else
  echo "Image is non-equal, container: ${container_img}, yaml file: ${yaml_img}"
  exit 1
fi
echo "harvester-node-manager upgrade successfully!"

# cleanup
rm -f nm-override.yaml

popd
