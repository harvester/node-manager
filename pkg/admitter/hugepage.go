package admitter

import (
	"context"
	"fmt"

	"github.com/harvester/webhook/pkg/server/admission"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/harvester/go-common/common"
	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	clientset "github.com/harvester/node-manager/pkg/generated/clientset/versioned"
	nodev1beta1 "github.com/harvester/node-manager/pkg/generated/clientset/versioned/typed/node.harvesterhci.io/v1beta1"
	werror "github.com/harvester/webhook/pkg/error"
)

type Hugepage struct {
	admission.DefaultValidator

	nodeconfigs nodev1beta1.NodeConfigInterface
}

func NewHugepageValidator(config *rest.Config) (*Hugepage, error) {
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	nodeconfigs := client.NodeV1beta1().NodeConfigs(common.HarvesterSystemNamespaceName)

	return &Hugepage{
		nodeconfigs: nodeconfigs,
	}, nil
}

func (v *Hugepage) Create(_ *admission.Request, newObj runtime.Object) error {
	newHugepage := newObj.(*v1beta1.Hugepage)
	return v.validate(newHugepage)
}

func (v *Hugepage) Update(_ *admission.Request, _ runtime.Object, newObj runtime.Object) error {
	newHugepage := newObj.(*v1beta1.Hugepage)
	return v.validate(newHugepage)
}

func (v *Hugepage) validate(hugepage *v1beta1.Hugepage) error {
	nodeconfig, err := v.nodeconfigs.Get(context.TODO(), hugepage.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return werror.NewInternalError(fmt.Sprintf("failed to find nodeconfig for %s: %v", hugepage.Name, err))
	}
	if nodeconfig.Spec.LonghornConfig != nil && nodeconfig.Spec.LonghornConfig.EnableV2DataEngine && hugepage.Spec.Static.NrHugepages < uint64(nodeconfig.Spec.LonghornConfig.HugepagesToAllocate) {
		return werror.NewInvalidError(fmt.Sprintf("spec.static.nrHugepages must be at least %d (required by Longhorn V2 Data Engine)", nodeconfig.Spec.LonghornConfig.HugepagesToAllocate), "spec.static.nrHugepages")
	}
	return nil
}

func (v *Hugepage) Resource() admission.Resource {
	return admission.Resource{
		Names:      []string{v1beta1.HugepageResourceName},
		Scope:      admissionregv1.ClusterScope,
		APIGroup:   v1beta1.SchemeGroupVersion.Group,
		APIVersion: v1beta1.SchemeGroupVersion.Version,
		ObjectType: &v1beta1.Hugepage{},
		OperationTypes: []admissionregv1.OperationType{
			admissionregv1.Create,
			admissionregv1.Update,
		},
	}
}
