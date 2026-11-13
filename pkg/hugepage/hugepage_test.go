package hugepage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"

	"github.com/stretchr/testify/require"
)

func Test_ConfigGeneration(t *testing.T) {
	assert := require.New(t)
	mgr, err := NewHugepageManager(context.TODO(), "./testdata/transparent", "./unused-in-this-test")
	assert.NoError(err, "expected to find no error")
	cfg := mgr.GetDefaultTHPConfig()
	assert.Equal(v1beta1.THPEnabled("madvise"), cfg.Enabled, "expected to find madvise")
	assert.Equal(v1beta1.THPShmemEnabled("never"), cfg.ShmemEnabled, "expected to find never")
	assert.Equal(v1beta1.THPDefrag("madvise"), cfg.Defrag, "expected to find madvise")
}

func Test_DefaultConfigGeneration(t *testing.T) {
	assert := require.New(t)
	mgr, err := NewHugepageManager(context.TODO(), "./nonexistentPath", "./unused-in-this-test")
	assert.NoError(err, "expected to find no error")
	cfg := mgr.GetDefaultTHPConfig()
	assert.Equal(v1beta1.THPEnabled("always"), cfg.Enabled, "expected to find always")
	assert.Equal(v1beta1.THPShmemEnabled("never"), cfg.ShmemEnabled, "expected to find never")
	assert.Equal(v1beta1.THPDefrag("madvise"), cfg.Defrag, "expected to find madvise")
}

func Test_GenerateStatus(t *testing.T) {
	assert := require.New(t)
	mgr, err := NewHugepageManager(context.TODO(), "./testdata/transparent", "./testdata/static")
	assert.NoError(err, "NewHugepageManager: expected to find no error")
	status, err := mgr.GenerateStatus()
	assert.NoError(err, "GenerateStatus: expected to find no error")
	assert.Equal(uint64(1024), status.Static.Hugepages, "expected to find 1024")
	assert.Equal(v1beta1.THPEnabled("madvise"), status.Transparent.Enabled, "expected to find madvise")
	assert.Equal(v1beta1.THPShmemEnabled("never"), status.Transparent.ShmemEnabled, "expected to find never")
	assert.Equal(v1beta1.THPDefrag("madvise"), status.Transparent.Defrag, "expected to find madvise")
}

func Test_ApplyConfig(t *testing.T) {
	assert := require.New(t)
	tmpDir := t.TempDir()
	for _, file := range []string{HugepagesFile, THPEnabledFile, THPShmemEnabledFile, THPDefragFile} {
		path := filepath.Join(tmpDir, file)
		f, err := os.Create(path)
		if err != nil {
			t.Errorf("Unable to create %s", path)
		}
		f.Close()
	}
	mgr, err := NewHugepageManager(context.TODO(), tmpDir, tmpDir)
	assert.NoError(err, "NewHugepageManager: expected to find no error")
	err = mgr.ApplyConfig(&v1beta1.HugepageSpec{
		Transparent: v1beta1.THPConfig{
			Enabled:      "always",
			ShmemEnabled: "never",
			Defrag:       "madvise",
		},
		Static: v1beta1.SHPConfig{
			Hugepages: 1024,
		},
	})
	assert.NoError(err, "ApplyConfig: expected to find no error")
}
