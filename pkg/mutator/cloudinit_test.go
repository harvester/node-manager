package mutator

import (
	"reflect"
	"testing"

	"github.com/harvester/webhook/pkg/server/admission"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func TestCreate(t *testing.T) {
	patchFilename := func(want string) admission.Patch {
		return admission.Patch([]admission.PatchOp{
			{Op: admission.PatchOpReplace, Path: "/spec/filename", Value: want},
		})
	}

	var noPatch admission.Patch

	tests := []struct {
		input string
		want  admission.Patch
	}{
		{"/baseonly/a.yaml", patchFilename("a.yaml")},
		{"missing_suffix", patchFilename("missing_suffix.yaml")},
		{"/baseonly/andmissingsuffix", patchFilename("andmissingsuffix.yaml")},
		{"b.yml", noPatch},
		{"b.yaml", noPatch},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := NewCloudInitMutator()
			cloudinit := &v1beta1.CloudInit{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1beta1.CloudInitSpec{
					MatchSelector: map[string]string{},
					Filename:      tt.input,
					Contents:      "hello, world",
				},
			}
			got, err := m.Create(new(admission.Request), cloudinit)
			if err != nil {
				t.Errorf("want err=<nil>, got err=%v", err)
			}

			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("want patch %+v, got patch %+v", tt.want, got)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	patchFilename := func(want string) admission.Patch {
		return admission.Patch([]admission.PatchOp{
			{Op: admission.PatchOpReplace, Path: "/spec/filename", Value: want},
		})
	}

	var noPatch admission.Patch

	tests := []struct {
		input string
		want  admission.Patch
	}{
		{"/baseonly/a.yaml", patchFilename("a.yaml")},
		{"missing_suffix", patchFilename("missing_suffix.yaml")},
		{"/baseonly/andmissingsuffix", patchFilename("andmissingsuffix.yaml")},
		{"b.yml", noPatch},
		{"b.yaml", noPatch},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := NewCloudInitMutator()

			cloudinit := &v1beta1.CloudInit{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1beta1.CloudInitSpec{
					MatchSelector: map[string]string{},
					Filename:      tt.input,
					Contents:      "hello, world",
				},
			}

			old := &v1beta1.CloudInit{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: v1beta1.CloudInitSpec{
					MatchSelector: map[string]string{},
					Filename:      "specifically_not_in_use.yaml",
					Contents:      "hello, world",
				},
			}

			got, err := m.Update(new(admission.Request), old, cloudinit)
			if err != nil {
				t.Errorf("want err=<nil>, got err=%v", err)
			}

			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("want patch %+v, got patch %+v", tt.want, got)
			}
		})
	}
}
