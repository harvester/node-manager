package hugepage

import (
	"context"
	"testing"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_ConfigGeneration(t *testing.T) {
	assert := require.New(t)
	mgr, err := NewHugepageManager(context.TODO(), "./testdata")
	assert.NoError(err, "expected to find no error")
	cfg := mgr.GetDefaultTHPConfig()
	assert.Equal(v1beta1.THPEnabled("madvise"), cfg.Enabled, "expected to find madvise")
	assert.Equal(v1beta1.THPShmemEnabled("never"), cfg.ShmemEnabled, "expected to find never")
	assert.Equal(v1beta1.THPDefrag("madvise"), cfg.Defrag, "expected to find madvise")
}

func Test_DefaultConfigGeneration(t *testing.T) {
	assert := require.New(t)
	mgr, err := NewHugepageManager(context.TODO(), "./nonexistentPath")
	assert.NoError(err, "expected to find no error")
	cfg := mgr.GetDefaultTHPConfig()
	assert.Equal(v1beta1.THPEnabled("always"), cfg.Enabled, "expected to find always")
	assert.Equal(v1beta1.THPShmemEnabled("never"), cfg.ShmemEnabled, "expected to find never")
	assert.Equal(v1beta1.THPDefrag("madvise"), cfg.Defrag, "expected to find madvise")
}
