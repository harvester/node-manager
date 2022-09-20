package ksmtuned

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ksmtunedv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func (c *Controller) NodeOnChange(_ string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil || node.Name != c.NodeName {
		return node, nil
	}

	if _, err := c.KsmtunedCache.Get(node.Name); err != nil {
		if !apierrors.IsNotFound(err) {
			return node, err
		}
		if _, err := c.Ksmtuneds.Create(defaultKsmtuned(node)); err != nil {
			return node, fmt.Errorf("failed to create Ksmtuned: %s", err)
		}
	}

	return node, nil
}

func defaultKsmtuned(node *corev1.Node) *ksmtunedv1.Ksmtuned {
	return &ksmtunedv1.Ksmtuned{
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
		Spec: ksmtunedv1.KsmtunedSpec{
			Run:                ksmtunedv1.Stop,
			Mode:               ksmtunedv1.StandardMode,
			ThresCoef:          20,
			KsmtunedParameters: modes[ksmtunedv1.StandardMode],
		},
		Status: ksmtunedv1.KsmtunedStatus{
			KsmdPhase: ksmtunedv1.KsmdStopped,
		},
	}
}
