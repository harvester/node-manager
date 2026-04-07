package admitter

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/harvester/webhook/pkg/server/admission"
	"go.yaml.in/yaml/v4"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	clientset "github.com/harvester/node-manager/pkg/generated/clientset/versioned"
	cloudinitv1beta1 "github.com/harvester/node-manager/pkg/generated/clientset/versioned/typed/node.harvesterhci.io/v1beta1"
)

var (
	errFilenameTaken     = errors.New("filename already in use")
	errProtectedFilename = errors.New("filename conflicts with a critical system-owned file")
	errMissingExt        = errors.New("filename does not end in .yaml or .yml")
	errNotYAML           = errors.New("could not parse document as yaml")
)

var builtinFilenameDenyList = []string{
	"90_custom.yaml",
	"99_settings.yaml",
	"elemental.config",
	"grubenv",
	"harvester.config",
	"install",
}

type CloudInit struct {
	admission.DefaultValidator

	cloudinits cloudinitv1beta1.CloudInitInterface
}

func NewCloudInitValidator(config *rest.Config) (*CloudInit, error) {
	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cloudinits := client.NodeV1beta1().CloudInits()

	return &CloudInit{
		cloudinits: cloudinits,
	}, nil
}

func (v *CloudInit) Create(_ *admission.Request, newObj runtime.Object) error {
	newCloudInit := newObj.(*v1beta1.CloudInit)
	return v.validate(newCloudInit)
}

func (v *CloudInit) Update(_ *admission.Request, _ runtime.Object, newObj runtime.Object) error {
	newCloudInit := newObj.(*v1beta1.CloudInit)
	return v.validate(newCloudInit)
}

func (v *CloudInit) validate(cloudinit *v1beta1.CloudInit) error {
	if v.missingExtension(cloudinit.Spec.Filename) {
		return errMissingExt
	}

	if v.isProtectedFilename(cloudinit.Spec.Filename) {
		return errProtectedFilename
	}

	taken, err := v.isFilenameTaken(cloudinit)
	if err != nil {
		return fmt.Errorf("check for duplicate filename: %w", err)
	}

	if taken {
		return errFilenameTaken
	}

	if err := isYaml(cloudinit.Spec.Contents); err != nil {
		return err
	}

	return nil
}

func (v *CloudInit) missingExtension(name string) bool {
	ext := filepath.Ext(name)
	return ext != ".yaml" && ext != ".yml"
}

func (v *CloudInit) isFilenameTaken(cloudinit *v1beta1.CloudInit) (bool, error) {
	cloudinits, err := v.cloudinits.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return true, err
	}

	for _, existing := range cloudinits.Items {
		if existing.Name == cloudinit.Name {
			continue
		}
		if existing.Spec.Filename == cloudinit.Spec.Filename {
			return true, nil
		}
	}

	return false, nil
}

func (v *CloudInit) isProtectedFilename(name string) bool {
	for _, protected := range builtinFilenameDenyList {
		if name == protected {
			return true
		}
	}
	return false
}

// isYaml checks whether content is empty or a valid YAML mapping document,
// which is the only structure cloud-init accepts.
func isYaml(contents string) error {
	var obj map[string]any
	var typeErr *yaml.LoadErrors
	err := yaml.Unmarshal([]byte(contents), &obj)
	if errors.As(err, &typeErr) {
		return errNotYAML
	}
	if err != nil {
		return err
	}
	return nil
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
