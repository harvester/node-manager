/*
Copyright 2022 Rancher Labs, Inc.

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
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type KsmtunedHandler func(string, *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error)

type KsmtunedController interface {
	generic.ControllerMeta
	KsmtunedClient

	OnChange(ctx context.Context, name string, sync KsmtunedHandler)
	OnRemove(ctx context.Context, name string, sync KsmtunedHandler)
	Enqueue(name string)
	EnqueueAfter(name string, duration time.Duration)

	Cache() KsmtunedCache
}

type KsmtunedClient interface {
	Create(*v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error)
	Update(*v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error)
	UpdateStatus(*v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Get(name string, options metav1.GetOptions) (*v1beta1.Ksmtuned, error)
	List(opts metav1.ListOptions) (*v1beta1.KsmtunedList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.Ksmtuned, err error)
}

type KsmtunedCache interface {
	Get(name string) (*v1beta1.Ksmtuned, error)
	List(selector labels.Selector) ([]*v1beta1.Ksmtuned, error)

	AddIndexer(indexName string, indexer KsmtunedIndexer)
	GetByIndex(indexName, key string) ([]*v1beta1.Ksmtuned, error)
}

type KsmtunedIndexer func(obj *v1beta1.Ksmtuned) ([]string, error)

type ksmtunedController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewKsmtunedController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) KsmtunedController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &ksmtunedController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromKsmtunedHandlerToHandler(sync KsmtunedHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1beta1.Ksmtuned
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1beta1.Ksmtuned))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *ksmtunedController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1beta1.Ksmtuned))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateKsmtunedDeepCopyOnChange(client KsmtunedClient, obj *v1beta1.Ksmtuned, handler func(obj *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error)) (*v1beta1.Ksmtuned, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *ksmtunedController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *ksmtunedController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *ksmtunedController) OnChange(ctx context.Context, name string, sync KsmtunedHandler) {
	c.AddGenericHandler(ctx, name, FromKsmtunedHandlerToHandler(sync))
}

func (c *ksmtunedController) OnRemove(ctx context.Context, name string, sync KsmtunedHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromKsmtunedHandlerToHandler(sync)))
}

func (c *ksmtunedController) Enqueue(name string) {
	c.controller.Enqueue("", name)
}

func (c *ksmtunedController) EnqueueAfter(name string, duration time.Duration) {
	c.controller.EnqueueAfter("", name, duration)
}

func (c *ksmtunedController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *ksmtunedController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *ksmtunedController) Cache() KsmtunedCache {
	return &ksmtunedCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *ksmtunedController) Create(obj *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error) {
	result := &v1beta1.Ksmtuned{}
	return result, c.client.Create(context.TODO(), "", obj, result, metav1.CreateOptions{})
}

func (c *ksmtunedController) Update(obj *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error) {
	result := &v1beta1.Ksmtuned{}
	return result, c.client.Update(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *ksmtunedController) UpdateStatus(obj *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error) {
	result := &v1beta1.Ksmtuned{}
	return result, c.client.UpdateStatus(context.TODO(), "", obj, result, metav1.UpdateOptions{})
}

func (c *ksmtunedController) Delete(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), "", name, *options)
}

func (c *ksmtunedController) Get(name string, options metav1.GetOptions) (*v1beta1.Ksmtuned, error) {
	result := &v1beta1.Ksmtuned{}
	return result, c.client.Get(context.TODO(), "", name, result, options)
}

func (c *ksmtunedController) List(opts metav1.ListOptions) (*v1beta1.KsmtunedList, error) {
	result := &v1beta1.KsmtunedList{}
	return result, c.client.List(context.TODO(), "", result, opts)
}

func (c *ksmtunedController) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), "", opts)
}

func (c *ksmtunedController) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*v1beta1.Ksmtuned, error) {
	result := &v1beta1.Ksmtuned{}
	return result, c.client.Patch(context.TODO(), "", name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type ksmtunedCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *ksmtunedCache) Get(name string) (*v1beta1.Ksmtuned, error) {
	obj, exists, err := c.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v1beta1.Ksmtuned), nil
}

func (c *ksmtunedCache) List(selector labels.Selector) (ret []*v1beta1.Ksmtuned, err error) {

	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.Ksmtuned))
	})

	return ret, err
}

func (c *ksmtunedCache) AddIndexer(indexName string, indexer KsmtunedIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1beta1.Ksmtuned))
		},
	}))
}

func (c *ksmtunedCache) GetByIndex(indexName, key string) (result []*v1beta1.Ksmtuned, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1beta1.Ksmtuned, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1beta1.Ksmtuned))
	}
	return result, nil
}

type KsmtunedStatusHandler func(obj *v1beta1.Ksmtuned, status v1beta1.KsmtunedStatus) (v1beta1.KsmtunedStatus, error)

type KsmtunedGeneratingHandler func(obj *v1beta1.Ksmtuned, status v1beta1.KsmtunedStatus) ([]runtime.Object, v1beta1.KsmtunedStatus, error)

func RegisterKsmtunedStatusHandler(ctx context.Context, controller KsmtunedController, condition condition.Cond, name string, handler KsmtunedStatusHandler) {
	statusHandler := &ksmtunedStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromKsmtunedHandlerToHandler(statusHandler.sync))
}

func RegisterKsmtunedGeneratingHandler(ctx context.Context, controller KsmtunedController, apply apply.Apply,
	condition condition.Cond, name string, handler KsmtunedGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &ksmtunedGeneratingHandler{
		KsmtunedGeneratingHandler: handler,
		apply:                     apply,
		name:                      name,
		gvk:                       controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterKsmtunedStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type ksmtunedStatusHandler struct {
	client    KsmtunedClient
	condition condition.Cond
	handler   KsmtunedStatusHandler
}

func (a *ksmtunedStatusHandler) sync(key string, obj *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type ksmtunedGeneratingHandler struct {
	KsmtunedGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *ksmtunedGeneratingHandler) Remove(key string, obj *v1beta1.Ksmtuned) (*v1beta1.Ksmtuned, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1beta1.Ksmtuned{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *ksmtunedGeneratingHandler) Handle(obj *v1beta1.Ksmtuned, status v1beta1.KsmtunedStatus) (v1beta1.KsmtunedStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.KsmtunedGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
