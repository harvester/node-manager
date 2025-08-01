/*
Copyright 2025 Rancher Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1beta1

import (
	context "context"

	nodeharvesterhciiov1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	scheme "github.com/harvester/node-manager/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// NodeConfigsGetter has a method to return a NodeConfigInterface.
// A group's client should implement this interface.
type NodeConfigsGetter interface {
	NodeConfigs(namespace string) NodeConfigInterface
}

// NodeConfigInterface has methods to work with NodeConfig resources.
type NodeConfigInterface interface {
	Create(ctx context.Context, nodeConfig *nodeharvesterhciiov1beta1.NodeConfig, opts v1.CreateOptions) (*nodeharvesterhciiov1beta1.NodeConfig, error)
	Update(ctx context.Context, nodeConfig *nodeharvesterhciiov1beta1.NodeConfig, opts v1.UpdateOptions) (*nodeharvesterhciiov1beta1.NodeConfig, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, nodeConfig *nodeharvesterhciiov1beta1.NodeConfig, opts v1.UpdateOptions) (*nodeharvesterhciiov1beta1.NodeConfig, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*nodeharvesterhciiov1beta1.NodeConfig, error)
	List(ctx context.Context, opts v1.ListOptions) (*nodeharvesterhciiov1beta1.NodeConfigList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *nodeharvesterhciiov1beta1.NodeConfig, err error)
	NodeConfigExpansion
}

// nodeConfigs implements NodeConfigInterface
type nodeConfigs struct {
	*gentype.ClientWithList[*nodeharvesterhciiov1beta1.NodeConfig, *nodeharvesterhciiov1beta1.NodeConfigList]
}

// newNodeConfigs returns a NodeConfigs
func newNodeConfigs(c *NodeV1beta1Client, namespace string) *nodeConfigs {
	return &nodeConfigs{
		gentype.NewClientWithList[*nodeharvesterhciiov1beta1.NodeConfig, *nodeharvesterhciiov1beta1.NodeConfigList](
			"nodeconfigs",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *nodeharvesterhciiov1beta1.NodeConfig { return &nodeharvesterhciiov1beta1.NodeConfig{} },
			func() *nodeharvesterhciiov1beta1.NodeConfigList { return &nodeharvesterhciiov1beta1.NodeConfigList{} },
		),
	}
}
