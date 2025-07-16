package hugepage

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func (c *Controller) NodeOnChange(_ string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil || node.Name != c.Name {
		return node, nil
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
			},
			Status: nodev1beta1.HugepageStatus{},
		}
	}

	if _, err := c.HugepageCache.Get(node.Name); err != nil {
		if !apierrors.IsNotFound(err) {
			return node, err
		}
		if _, err := c.HugepageClient.Create(defaultHugepage(node)); err != nil {
			return node, fmt.Errorf("failed to create Hugepage: %s", err)
		}
	}

	return node, nil
}
