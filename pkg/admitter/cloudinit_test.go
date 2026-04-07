package admitter

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/harvester/webhook/pkg/server/admission"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func TestProtectedFilenames(t *testing.T) {
	want := map[string]struct{}{
		"90_custom.yaml":   {},
		"99_settings.yaml": {},
		"elemental.config": {},
		"grubenv":          {},
		"harvester.config": {},
		"install":          {},
	}

	got := make(map[string]struct{})
	for _, f := range builtinFilenameDenyList {
		got[f] = struct{}{}
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestCreate(t *testing.T) {
	origDenyList := builtinFilenameDenyList
	defer func() { builtinFilenameDenyList = origDenyList }()
	builtinFilenameDenyList = []string{
		"helloworld.yaml",
	}

	existing := []v1beta1.CloudInit{
		{ObjectMeta: v1.ObjectMeta{Name: "ssh-access"}, Spec: v1beta1.CloudInitSpec{Filename: "99_ssh.yaml"}},
	}

	tests := []struct {
		name  string
		input v1beta1.CloudInitSpec
		want  error
	}{
		{"allow yaml", v1beta1.CloudInitSpec{Filename: "hi.yaml"}, nil},
		{"allow yml", v1beta1.CloudInitSpec{Filename: "hi.yml"}, nil},
		{"filename collision", v1beta1.CloudInitSpec{Filename: "99_ssh.yaml"}, errFilenameTaken},
		{"conflicts with protected file", v1beta1.CloudInitSpec{Filename: "helloworld.yaml"}, errProtectedFilename},
		{"not yaml or yml file ext", v1beta1.CloudInitSpec{Filename: "a"}, errMissingExt},
		{"not yaml contents", v1beta1.CloudInitSpec{Filename: "not.yaml", Contents: "hello, there"}, errNotYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl := &CloudInit{cloudinits: &mockClient{list: existing}}

			cloudinit := &v1beta1.CloudInit{
				ObjectMeta: v1.ObjectMeta{Name: "test-cloudinit"},
				Spec:       tt.input,
			}

			got := ctl.Create(new(admission.Request), cloudinit)
			if !errors.Is(got, tt.want) {
				t.Errorf("want err=%v, got err=%v", tt.want, got)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	origDenyList := builtinFilenameDenyList
	defer func() { builtinFilenameDenyList = origDenyList }()
	builtinFilenameDenyList = []string{
		"helloworld.yaml",
	}

	existing := []v1beta1.CloudInit{
		{ObjectMeta: v1.ObjectMeta{Name: "ssh-access"}, Spec: v1beta1.CloudInitSpec{Filename: "99_ssh.yaml"}},
	}

	tests := []struct {
		name  string
		input v1beta1.CloudInitSpec
		want  error
	}{
		{"allow yaml", v1beta1.CloudInitSpec{Filename: "hi.yaml"}, nil},
		{"allow yml", v1beta1.CloudInitSpec{Filename: "hi.yml"}, nil},
		{"filename collision", v1beta1.CloudInitSpec{Filename: "99_ssh.yaml"}, errFilenameTaken},
		{"conflicts with protected file", v1beta1.CloudInitSpec{Filename: "helloworld.yaml"}, errProtectedFilename},
		{"not yaml or yml file ext", v1beta1.CloudInitSpec{Filename: "a"}, errMissingExt},
		{"not yaml contents", v1beta1.CloudInitSpec{Filename: "not.yaml", Contents: "hello, there"}, errNotYAML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl := &CloudInit{cloudinits: &mockClient{list: existing}}

			cloudinit := &v1beta1.CloudInit{
				ObjectMeta: v1.ObjectMeta{Name: "test-cloudinit"},
				Spec:       tt.input,
			}

			old := &v1beta1.CloudInit{
				ObjectMeta: v1.ObjectMeta{Name: "test-cloudinit"},
				Spec:       v1beta1.CloudInitSpec{Filename: "specifically-not-in-use.yaml"},
			}

			got := ctl.Update(new(admission.Request), old, cloudinit)
			if !errors.Is(got, tt.want) {
				t.Errorf("want err=%v, got err=%v", tt.want, got)
			}
		})
	}
}

func TestUpdateRegressionCases(t *testing.T) {
	tests := []struct {
		name      string
		existing  []v1beta1.CloudInit
		oldSpec   v1beta1.CloudInitSpec
		newSpec   v1beta1.CloudInitSpec
		wantError error
	}{
		{
			name:      "unchanged filename still validates contents",
			oldSpec:   v1beta1.CloudInitSpec{Filename: "same.yaml", Contents: "a: b"},
			newSpec:   v1beta1.CloudInitSpec{Filename: "same.yaml", Contents: "hello, there"},
			wantError: errNotYAML,
		},
		{
			name: "ignore current object for duplicate filename",
			existing: []v1beta1.CloudInit{{
				ObjectMeta: v1.ObjectMeta{Name: "test-cloudinit"},
				Spec:       v1beta1.CloudInitSpec{Filename: "same.yaml"},
			}},
			oldSpec: v1beta1.CloudInitSpec{Filename: "same.yaml", Contents: "k: v"},
			newSpec: v1beta1.CloudInitSpec{Filename: "same.yaml", Contents: "k: new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctl := &CloudInit{cloudinits: &mockClient{list: tt.existing}}
			old := &v1beta1.CloudInit{
				ObjectMeta: v1.ObjectMeta{Name: "test-cloudinit"},
				Spec:       tt.oldSpec,
			}
			newObj := &v1beta1.CloudInit{
				ObjectMeta: v1.ObjectMeta{Name: "test-cloudinit"},
				Spec:       tt.newSpec,
			}
			got := ctl.Update(new(admission.Request), old, newObj)
			if tt.wantError != nil {
				assert.ErrorIs(t, got, tt.wantError)
			} else {
				assert.NoError(t, got)
			}
		})
	}
}

func TestIsYaml(t *testing.T) {
	tests := []struct {
		name           string
		contents       string
		wantErrNotYAML bool
		wantOtherErr   bool
	}{
		{name: "empty content is allowed", contents: ""},
		{name: "whitespace content is allowed", contents: "   \n"},
		{name: "mapping YAML is allowed", contents: "a: b"},
		{name: "flow sequence YAML is allowed", contents: "a: [1, 2]"},
		{name: "block sequence YAML is allowed", contents: "a:\n  - 1\n  - 2"},
		{name: "nested mapping YAML is allowed", contents: "a:\n  b: c"},
		{name: "scalar YAML is rejected", contents: "hello, there", wantErrNotYAML: true},
		{name: "sequence YAML is rejected", contents: "- a\n- b", wantErrNotYAML: true},
		{name: "invalid YAML syntax returns parser error", contents: "a: [1, 2", wantOtherErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isYaml(tt.contents)
			switch {
			case tt.wantErrNotYAML:
				assert.ErrorContains(t, err, errNotYAML.Error())
			case tt.wantOtherErr:
				assert.Error(t, err)
			default:
				assert.NoError(t, err)
			}
		})
	}
}

// Sadly, github.com/rancher/wrangler/pkg/generic/fake package generates mock clients that lack
// the ctx parameter that is required by the CloudInitInterface.

type mockClient struct {
	list []v1beta1.CloudInit
}

func (m *mockClient) Create(_ context.Context, _ *v1beta1.CloudInit, _ v1.CreateOptions) (*v1beta1.CloudInit, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) Update(_ context.Context, _ *v1beta1.CloudInit, _ v1.UpdateOptions) (*v1beta1.CloudInit, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) UpdateStatus(_ context.Context, _ *v1beta1.CloudInit, _ v1.UpdateOptions) (*v1beta1.CloudInit, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) Delete(_ context.Context, _ string, _ v1.DeleteOptions) error {
	return errors.New("not implemented")
}

func (m *mockClient) DeleteCollection(_ context.Context, _ v1.DeleteOptions, _ v1.ListOptions) error {
	return errors.New("not implemented")
}

func (m *mockClient) Get(_ context.Context, _ string, _ v1.GetOptions) (*v1beta1.CloudInit, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) List(_ context.Context, _ v1.ListOptions) (*v1beta1.CloudInitList, error) {
	return &v1beta1.CloudInitList{Items: m.list}, nil
}

func (m *mockClient) Watch(_ context.Context, _ v1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("not implemented")
}

func (m *mockClient) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ v1.PatchOptions, _ ...string) (result *v1beta1.CloudInit, err error) {
	return nil, errors.New("not implemented")
}
