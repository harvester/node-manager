package admitter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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
	return v.validate(newCloudInit, "")
}

func (v *CloudInit) Update(_ *admission.Request, oldObj runtime.Object, newObj runtime.Object) error {
	oldCloudInit := oldObj.(*v1beta1.CloudInit)
	newCloudInit := newObj.(*v1beta1.CloudInit)
	return v.validate(newCloudInit, oldCloudInit.Name)
}

func (v *CloudInit) validate(cloudinit *v1beta1.CloudInit, ignoreName string) error {
	if v.missingExtension(cloudinit.Spec.Filename) {
		return errMissingExt
	}

	if v.isProtectedFilename(cloudinit.Spec.Filename) {
		return errProtectedFilename
	}

	taken, err := v.isFilenameTaken(cloudinit.Spec.Filename, ignoreName)
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

func (v *CloudInit) isFilenameTaken(name, ignoreName string) (bool, error) {
	cloudinits, err := v.cloudinits.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return true, err
	}

	for _, cloudinit := range cloudinits.Items {
		if cloudinit.Name == ignoreName {
			continue
		}
		if cloudinit.Spec.Filename == name {
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

// isYaml checks whether the specified content is empty or a valid YAML document
// whose root value is a mapping, as expected by CloudInit.
func isYaml(contents string) error {
	// Use `NewLoader` for more control over the parsing process.
	loader, err := yaml.NewLoader(strings.NewReader(contents))
	if err != nil {
		return err
	}

	var node yaml.Node
	if err := loader.Load(&node); err != nil {
		if errors.Is(err, io.EOF) {
			// Keep backward compatibility: empty content has historically
			// been accepted.
			return nil
		}
		return err
	}

	// Keep backward compatibility: empty content has historically been accepted.
	if len(node.Content) == 0 {
		return nil
	}

	if node.Kind != yaml.DocumentNode || len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return errNotYAML
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
