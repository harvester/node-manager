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

package fake

import (
	v1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	nodeharvesterhciiov1beta1 "github.com/harvester/node-manager/pkg/generated/clientset/versioned/typed/node.harvesterhci.io/v1beta1"
	gentype "k8s.io/client-go/gentype"
)

// fakeKsmtuneds implements KsmtunedInterface
type fakeKsmtuneds struct {
	*gentype.FakeClientWithList[*v1beta1.Ksmtuned, *v1beta1.KsmtunedList]
	Fake *FakeNodeV1beta1
}

func newFakeKsmtuneds(fake *FakeNodeV1beta1) nodeharvesterhciiov1beta1.KsmtunedInterface {
	return &fakeKsmtuneds{
		gentype.NewFakeClientWithList[*v1beta1.Ksmtuned, *v1beta1.KsmtunedList](
			fake.Fake,
			"",
			v1beta1.SchemeGroupVersion.WithResource("ksmtuneds"),
			v1beta1.SchemeGroupVersion.WithKind("Ksmtuned"),
			func() *v1beta1.Ksmtuned { return &v1beta1.Ksmtuned{} },
			func() *v1beta1.KsmtunedList { return &v1beta1.KsmtunedList{} },
			func(dst, src *v1beta1.KsmtunedList) { dst.ListMeta = src.ListMeta },
			func(list *v1beta1.KsmtunedList) []*v1beta1.Ksmtuned { return gentype.ToPointerSlice(list.Items) },
			func(list *v1beta1.KsmtunedList, items []*v1beta1.Ksmtuned) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
