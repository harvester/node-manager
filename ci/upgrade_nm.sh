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
node_manager_container_img=$(kubectl get ds harvester-node-manager -n harvester-system -o yaml | yq -e '.spec.template.spec.containers[] | select(.name == "node-manager") | .image')
node_manager_yaml_img=$(yq -e '.image.repository + ":" + .image.tag' nm-override.yaml)
if [[ "${node_manager_container_img}" == "${node_manager_yaml_img}" ]]; then
  echo "node-manager image is equal: ${node_manager_yaml_img}"
else
  echo "node-manager image is non-equal, container: ${node_manager_container_img}, yaml file: ${node_manager_yaml_img}"
  exit 1
fi

webhook_container_img=$(kubectl get deployment harvester-node-manager-webhook -n harvester-system -o yaml | yq -e '.spec.template.spec.containers[] | select(.name == "harvester-node-manager-webhook") | .image')
webhook_yaml_img=$(yq -e '.webhook.image.repository + ":" + .webhook.image.tag' nm-override.yaml)
if [[ "${webhook_container_img}" == "${webhook_yaml_img}" ]]; then
  echo "webhook image is equal: ${webhook_yaml_img}"
else
  echo "webhook image is non-equal, container: ${webhook_container_img}, yaml file: ${webhook_yaml_img}"
  exit 1
fi
echo "harvester-node-manager upgrade successfully!"

# cleanup
rm -f nm-override.yaml

popd
