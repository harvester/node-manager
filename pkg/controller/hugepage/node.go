package hugepage

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harvester/go-common/common"
	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func (c *Controller) NodeOnChange(_ string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil || node.Name != c.Name {
		return node, nil
	}

	if _, err := c.HugepageCache.Get(node.Name); err != nil {
		if !apierrors.IsNotFound(err) {
			return node, err
		}

		var hugepages uint64 = 0
		if observedStatus, err := c.HugepageManager.GenerateStatus(); err == nil {
			// In case there's already hugepages manually allocated somehow, pick that up
			// as the default for the new hugepage CR
			hugepages = observedStatus.Static.Hugepages
		}

		if nodeConfig, err := c.NodeConfigCache.Get(common.HarvesterSystemNamespaceName, node.Name); err == nil && nodeConfig.Spec.LonghornConfig != nil && nodeConfig.Spec.LonghornConfig.EnableV2DataEngine {
			// If we can get a NodeConfig, use this to adjust default hugepages if necessary
			// for the Longhorn V2 data engine.  If we can't get a NodeConfig (for any
			// reason), don't worry about it.  It will be sync'd eventually, e.g. via
			// setting.nodeOnChanged() in the Harvester repo.
			if hugepages < uint64(nodeConfig.Spec.LonghornConfig.HugepagesToAllocate) {
				hugepages = uint64(nodeConfig.Spec.LonghornConfig.HugepagesToAllocate)
			}
		}

		defaultHugepage := func(node *corev1.Node) *nodev1beta1.Hugepage {
			return &nodev1beta1.Hugepage{
				ObjectMeta: metav1.ObjectMeta{
					Name: node.Name,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "Node",
							Name:       node.Name,
							UID:        node.UID,
						},
					},
				},
				Spec: nodev1beta1.HugepageSpec{
					Transparent: *c.HugepageManager.GetDefaultTHPConfig(),
					Static: nodev1beta1.SHPConfig{
						Hugepages: hugepages,
					},
				},
				Status: nodev1beta1.HugepageStatus{},
			}
		}

		if _, err := c.HugepageClient.Create(defaultHugepage(node)); err != nil {
			return node, fmt.Errorf("failed to create Hugepage: %s", err)
		}
	}

	return node, nil
}
