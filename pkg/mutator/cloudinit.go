package mutator

import (
	"fmt"
	"path/filepath"

	"github.com/harvester/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

type CloudInit struct {
	admission.DefaultMutator
}

func NewCloudInitMutator() *CloudInit {
	return &CloudInit{}
}

func (m *CloudInit) Create(_ *admission.Request, newObj runtime.Object) (admission.Patch, error) {
	newCloudInit := newObj.(*v1beta1.CloudInit)
	return patchFilenameIfNecessary(newCloudInit)
}

func (m *CloudInit) Update(_ *admission.Request, _ runtime.Object, newObj runtime.Object) (admission.Patch, error) {
	newCloudInit := newObj.(*v1beta1.CloudInit)
	return patchFilenameIfNecessary(newCloudInit)
}

func patchFilenameIfNecessary(newCloudInit *v1beta1.CloudInit) (admission.Patch, error) {
	var patch admission.Patch

	filename := ensureFileExtension(filepath.Base(newCloudInit.Spec.Filename))
	if filename == newCloudInit.Spec.Filename {
		return patch, nil
	}

	p := admission.PatchOp{
		Op:    admission.PatchOpReplace,
		Path:  "/spec/filename",
		Value: filename,
	}
	patch = append(patch, p)

	return patch, nil
}

func ensureFileExtension(s string) string {
	accept := func(extension string) bool {
		extensions := []string{".yaml", ".yml"}
		for _, ext := range extensions {
			if ext == extension {
				return true
			}
		}
		return false
	}
	if accept(filepath.Ext(s)) {
		return s
	}
	return fmt.Sprintf("%s.yaml", s)
}

func (m *CloudInit) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{v1beta1.CloudInitResourceName},
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   v1beta1.SchemeGroupVersion.Group,
		APIVersion: v1beta1.SchemeGroupVersion.Version,
		ObjectType: &v1beta1.CloudInit{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
