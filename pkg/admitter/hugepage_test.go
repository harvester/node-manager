package admitter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/harvester/webhook/pkg/server/admission"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func Test_HugepageCreate(t *testing.T) {
	nodeconfigs := []v1beta1.NodeConfig{
		{ObjectMeta: v1.ObjectMeta{Name: "node-0"}, Spec: v1beta1.NodeConfigSpec{LonghornConfig: &v1beta1.LonghornConfig{HugepagesToAllocate: 1024}}},
	}

	tests := []struct {
		name  string
		input v1beta1.HugepageSpec
		valid bool
	}{
		{"nrHugepages < hugepagesToAllocate", v1beta1.HugepageSpec{Static: v1beta1.SHPConfig{NrHugepages: 0}}, false},
		{"nrHugepages == hugepagesToAllocate", v1beta1.HugepageSpec{Static: v1beta1.SHPConfig{NrHugepages: 1024}}, true},
		{"nrHugepages > hugepagesToAllocate", v1beta1.HugepageSpec{Static: v1beta1.SHPConfig{NrHugepages: 2048}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl := &Hugepage{nodeconfigs: &mockNodeConfigClient{list: nodeconfigs}}
			hugepage := &v1beta1.Hugepage{
				ObjectMeta: v1.ObjectMeta{Name: "node-0"},
				Spec:       tt.input,
			}
			got := ctl.Create(new(admission.Request), hugepage)
			if tt.valid {
				assert.NoError(t, got)
			} else {
				assert.Error(t, got)
			}
		})
	}
}

func Test_HugepageUpdate(t *testing.T) {
	nodeconfigs := []v1beta1.NodeConfig{
		{ObjectMeta: v1.ObjectMeta{Name: "node-0"}, Spec: v1beta1.NodeConfigSpec{LonghornConfig: &v1beta1.LonghornConfig{HugepagesToAllocate: 1024}}},
	}

	tests := []struct {
		name  string
		input v1beta1.HugepageSpec
		valid bool
	}{
		{"nrHugepages < hugepagesToAllocate", v1beta1.HugepageSpec{Static: v1beta1.SHPConfig{NrHugepages: 0}}, false},
		{"nrHugepages == hugepagesToAllocate", v1beta1.HugepageSpec{Static: v1beta1.SHPConfig{NrHugepages: 1024}}, true},
		{"nrHugepages > hugepagesToAllocate", v1beta1.HugepageSpec{Static: v1beta1.SHPConfig{NrHugepages: 2048}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl := &Hugepage{nodeconfigs: &mockNodeConfigClient{list: nodeconfigs}}
			hugepage := &v1beta1.Hugepage{
				ObjectMeta: v1.ObjectMeta{Name: "node-0"},
				Spec:       tt.input,
			}
			old := &v1beta1.Hugepage{
				ObjectMeta: v1.ObjectMeta{Name: "node-0"},
				Spec:       v1beta1.HugepageSpec{},
			}
			got := ctl.Update(new(admission.Request), old, hugepage)
			if tt.valid {
				assert.NoError(t, got)
			} else {
				assert.Error(t, got)
			}
		})
	}
}

type mockNodeConfigClient struct {
	list []v1beta1.NodeConfig
}

func (m *mockNodeConfigClient) Create(_ context.Context, _ *v1beta1.NodeConfig, _ v1.CreateOptions) (*v1beta1.NodeConfig, error) {
	return nil, errors.New("not implemented")
}

func (m *mockNodeConfigClient) Update(_ context.Context, _ *v1beta1.NodeConfig, _ v1.UpdateOptions) (*v1beta1.NodeConfig, error) {
	return nil, errors.New("not implemented")
}

func (m *mockNodeConfigClient) UpdateStatus(_ context.Context, _ *v1beta1.NodeConfig, _ v1.UpdateOptions) (*v1beta1.NodeConfig, error) {
	return nil, errors.New("not implemented")
}

func (m *mockNodeConfigClient) Delete(_ context.Context, _ string, _ v1.DeleteOptions) error {
	return errors.New("not implemented")
}

func (m *mockNodeConfigClient) DeleteCollection(_ context.Context, _ v1.DeleteOptions, _ v1.ListOptions) error {
	return errors.New("not implemented")
}

func (m *mockNodeConfigClient) Get(_ context.Context, name string, _ v1.GetOptions) (*v1beta1.NodeConfig, error) {
	for _, n := range m.list {
		if n.Name == name {
			return &n, nil
		}
	}
	return nil, errors.New("nodeconfig not found")
}

func (m *mockNodeConfigClient) List(_ context.Context, _ v1.ListOptions) (*v1beta1.NodeConfigList, error) {
	return &v1beta1.NodeConfigList{Items: m.list}, nil
}

func (m *mockNodeConfigClient) Watch(_ context.Context, _ v1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("not implemented")
}

func (m *mockNodeConfigClient) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ v1.PatchOptions, _ ...string) (result *v1beta1.NodeConfig, err error) {
	return nil, errors.New("not implemented")
}
