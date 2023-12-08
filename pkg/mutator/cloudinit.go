package mutator

import (
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

func (m *CloudInit) Create(_ *admission.Request, _ runtime.Object) (admission.Patch, error) {
	var patch admission.Patch
	// Not implemented, validator will fail request
	return patch, nil
}

func (m *CloudInit) Update(_ *admission.Request, _ runtime.Object, _ runtime.Object) (admission.Patch, error) {
	var patch admission.Patch
	// Not implemented, validator will fail request
	return patch, nil
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
