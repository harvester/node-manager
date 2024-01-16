package cloudinit

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	cloudinitv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

const AnnotationHash = "node.harvesterhci.io/cloudinit-hash"

var Directory = "/host/oem"

// RequireLocal ensures that the Elemental cloud-init file described by
// the given `cloudinit` object is an exact copy of the `cloudinit` object's
// contents.
func RequireLocal(cloudinit *cloudinitv1.CloudInit) (bool, error) {
	absPath := filepath.Join(Directory, cloudinit.Spec.Filename)

	f, err := os.Open(absPath)
	var r io.Reader = f
	if err != nil {
		r = strings.NewReader("")
	} else {
		defer f.Close()
	}

	diskChecksum, err := Measure(r)
	if err != nil {
		return false, err
	}

	if fmt.Sprintf("%x", diskChecksum) == cloudinit.Annotations[AnnotationHash] {
		return false, nil
	}

	tempFile, err := os.CreateTemp(Directory, "node-manager")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, strings.NewReader(cloudinit.Spec.Contents))
	if err != nil {
		return false, err
	}

	err = os.Rename(tempFile.Name(), absPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

func Measure(r io.Reader) ([]byte, error) {
	h := sha256.New()
	_, err := io.Copy(h, r)
	return h.Sum(nil), err
}

func MatchesNode(node *corev1.Node, cloudinit *cloudinitv1.CloudInit) bool {
	selector := labels.SelectorFromSet(labels.Set(cloudinit.Spec.MatchSelector))
	return selector.Matches(labels.Set(node.GetLabels()))
}
