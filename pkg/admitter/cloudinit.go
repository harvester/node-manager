package admitter

import (
	"errors"

	"github.com/harvester/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

type CloudInit struct {
	admission.DefaultValidator
}

func NewCloudInitValidator() *CloudInit {
	return &CloudInit{}
}

func (v *CloudInit) Create(request *admission.Request, newObj runtime.Object) error {
	_, _ = request, newObj
	return errors.New("not implemented")
}

func (v *CloudInit) Update(request *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	_, _, _ = request, oldObj, newObj
	return errors.New("not implemented")
}

func (v *CloudInit) Resource() admission.Resource {
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
