/*
Copyright 2023 Rancher Labs, Inc.

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
	"context"
	"time"

	v1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	scheme "github.com/harvester/node-manager/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CloudInitsGetter has a method to return a CloudInitInterface.
// A group's client should implement this interface.
type CloudInitsGetter interface {
	CloudInits() CloudInitInterface
}

// CloudInitInterface has methods to work with CloudInit resources.
type CloudInitInterface interface {
	Create(ctx context.Context, cloudInit *v1beta1.CloudInit, opts v1.CreateOptions) (*v1beta1.CloudInit, error)
	Update(ctx context.Context, cloudInit *v1beta1.CloudInit, opts v1.UpdateOptions) (*v1beta1.CloudInit, error)
	UpdateStatus(ctx context.Context, cloudInit *v1beta1.CloudInit, opts v1.UpdateOptions) (*v1beta1.CloudInit, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta1.CloudInit, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta1.CloudInitList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.CloudInit, err error)
	CloudInitExpansion
}

// cloudInits implements CloudInitInterface
type cloudInits struct {
	client rest.Interface
}

// newCloudInits returns a CloudInits
func newCloudInits(c *NodeV1beta1Client) *cloudInits {
	return &cloudInits{
		client: c.RESTClient(),
	}
}

// Get takes name of the cloudInit, and returns the corresponding cloudInit object, and an error if there is any.
func (c *cloudInits) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.CloudInit, err error) {
	result = &v1beta1.CloudInit{}
	err = c.client.Get().
		Resource("cloudinits").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CloudInits that match those selectors.
func (c *cloudInits) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.CloudInitList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta1.CloudInitList{}
	err = c.client.Get().
		Resource("cloudinits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cloudInits.
func (c *cloudInits) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("cloudinits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a cloudInit and creates it.  Returns the server's representation of the cloudInit, and an error, if there is any.
func (c *cloudInits) Create(ctx context.Context, cloudInit *v1beta1.CloudInit, opts v1.CreateOptions) (result *v1beta1.CloudInit, err error) {
	result = &v1beta1.CloudInit{}
	err = c.client.Post().
		Resource("cloudinits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cloudInit).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a cloudInit and updates it. Returns the server's representation of the cloudInit, and an error, if there is any.
func (c *cloudInits) Update(ctx context.Context, cloudInit *v1beta1.CloudInit, opts v1.UpdateOptions) (result *v1beta1.CloudInit, err error) {
	result = &v1beta1.CloudInit{}
	err = c.client.Put().
		Resource("cloudinits").
		Name(cloudInit.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cloudInit).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *cloudInits) UpdateStatus(ctx context.Context, cloudInit *v1beta1.CloudInit, opts v1.UpdateOptions) (result *v1beta1.CloudInit, err error) {
	result = &v1beta1.CloudInit{}
	err = c.client.Put().
		Resource("cloudinits").
		Name(cloudInit.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(cloudInit).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the cloudInit and deletes it. Returns an error if one occurs.
func (c *cloudInits) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("cloudinits").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *cloudInits) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("cloudinits").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched cloudInit.
func (c *cloudInits) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.CloudInit, err error) {
	result = &v1beta1.CloudInit{}
	err = c.client.Patch(pt).
		Resource("cloudinits").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
