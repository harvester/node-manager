package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	nodeconfigv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
	"github.com/kevinburke/ssh_config"
	"github.com/melbahja/goph"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/harvester/node-manager/pkg/generated/clientset/versioned"
)

type NTPSuite struct {
	suite.Suite
	sshClient      *goph.Client
	clientSet      *clientset.Clientset
	targetNodeName string
	targetDiskName string
}

const defaultNTPServers = "0.opensuse.pool.ntp.org 1.opensuse.pool.ntp.org 2.opensuse.pool.ntp.org 3.opensuse.pool.ntp.org"

func (s *NTPSuite) SetupSuite() {
	vagrantRancherdHome := os.Getenv("VAGRANT_RANCHERD_HOME")
	require.NotEmpty(s.T(), vagrantRancherdHome, "VAGRANT_RANCHERD_HOME should not be empty")

	nodeName := ""

	cmd := exec.Command("vagrant", "ssh-config", "node1")
	cmd.Dir = vagrantRancherdHome

	stdout, err := cmd.Output()
	require.NoError(s.T(), err, "Failed to run vagrant ssh-config node1")

	cfg, err := ssh_config.DecodeBytes(stdout)
	require.NoError(s.T(), err, "Failed to decode ssh-config")

	// consider wildcard, so length shoule be 2
	require.Equal(s.T(), 2, len(cfg.Hosts), "number of Hosts on SSH-config should be 2")
	for _, host := range cfg.Hosts {
		if host.String() == "" {
			// wildcard, continue
			continue
		}
		nodeName = host.Patterns[0].String()
		break
	}
	require.True(s.T(), nodeName != "", "nodeName should not be empty.")
	s.targetNodeName = nodeName

	targetHost, _ := cfg.Get(nodeName, "HostName")
	targetUser, _ := cfg.Get(nodeName, "User")
	targetPrivateKey, _ := cfg.Get(nodeName, "IdentityFile")
	auth, err := goph.Key(targetPrivateKey, "")
	require.NoError(s.T(), err, "Failed to generate ssh auth key")

	s.sshClient, err = goph.NewUnknown(targetUser, targetHost, auth)
	require.NoError(s.T(), err, "Failed to create ssh connection")

	kubeconfig := filepath.Join(vagrantRancherdHome, "kubeconfig")
	require.FileExists(s.T(), kubeconfig, "kubeconfig should exist")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(s.T(), err, "Failed to generate clientset config")

	s.clientSet, err = clientset.NewForConfig(config)
	require.NoError(s.T(), err, "Failed to create clientset")
}

// setup default NTPServers
func (s *NTPSuite) BeforeTest(_, _ string) {
	newNodeConfig := &nodeconfigv1.NodeConfig{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      s.targetNodeName,
			Namespace: "harvester-system",
		},
		Spec: nodeconfigv1.NodeConfigSpec{
			NTPConfig: &nodeconfigv1.NTPConfig{
				NTPServers: defaultNTPServers,
			},
		},
	}

	nodeConfigs := s.clientSet.NodeV1beta1().NodeConfigs("harvester-system")
	nodeConfig, err := nodeConfigs.Get(context.TODO(), s.targetNodeName, k8smetav1.GetOptions{})
	if err != nil {
		_, err = nodeConfigs.Create(context.TODO(), newNodeConfig, k8smetav1.CreateOptions{})
		require.NoError(s.T(), err, "Failed to create NodeConfig")
	} else {
		updateNodeConfig := nodeConfig.DeepCopy()
		updateNodeConfig.Spec.NTPConfig = newNodeConfig.Spec.NTPConfig
		_, err = nodeConfigs.Update(context.TODO(), updateNodeConfig, k8smetav1.UpdateOptions{})
		require.NoError(s.T(), err, "Failed to update NodeConfig")
	}

	require.Eventually(s.T(), func() bool {
		out, _ := s.sshClient.Run("timedatectl show-timesync")
		return strings.Contains(string(out), fmt.Sprintf("SystemNTPServers=%s", defaultNTPServers))
	}, 10*time.Second, 1*time.Second, fmt.Sprintf("NTPServers should be %s", defaultNTPServers))
}

// restore default NTPServers
func (s *NTPSuite) AfterTest(_, _ string) {
	defer func() {
		if s.sshClient != nil {
			s.sshClient.Close()
		}
	}()

	nodeConfigs := s.clientSet.NodeV1beta1().NodeConfigs("harvester-system")

	nodeConfig, err := nodeConfigs.Get(context.TODO(), s.targetNodeName, k8smetav1.GetOptions{})
	require.NoError(s.T(), err, "Failed to get NodeConfig")
	require.NotNil(s.T(), nodeConfig, "NodeConfig should not be nil")

	updateNodeConfig := nodeConfig.DeepCopy()
	updateNodeConfig.Spec.NTPConfig = &nodeconfigv1.NTPConfig{NTPServers: defaultNTPServers}

	_, err = nodeConfigs.Update(context.TODO(), updateNodeConfig, k8smetav1.UpdateOptions{})
	require.NoError(s.T(), err, "Failed to update NodeConfig")
	require.Eventually(s.T(), func() bool {
		out, _ := s.sshClient.Run("timedatectl show-timesync")
		return strings.Contains(string(out), fmt.Sprintf("SystemNTPServers=%s", defaultNTPServers))
	}, 10*time.Second, 1*time.Second, fmt.Sprintf("NTPServers should be %s", defaultNTPServers))
}

func TestNTPServerConfig(t *testing.T) {
	suite.Run(t, new(NTPSuite))
}

func (s *NTPSuite) TestNTP() {
	nodeConfigs := s.clientSet.NodeV1beta1().NodeConfigs("harvester-system")

	googleNTPServers := "time1.google.com time2.google.com"

	nodeConfig, err := nodeConfigs.Get(context.TODO(), s.targetNodeName, k8smetav1.GetOptions{})
	require.NoError(s.T(), err, "Failed to get NodeConfig")
	require.NotNil(s.T(), nodeConfig, "NodeConfig should not be nil")

	updateNodeConfig := nodeConfig.DeepCopy()
	updateNodeConfig.Spec.NTPConfig = &nodeconfigv1.NTPConfig{NTPServers: googleNTPServers}

	_, err = nodeConfigs.Update(context.TODO(), updateNodeConfig, k8smetav1.UpdateOptions{})
	require.NoError(s.T(), err, "Failed to create NodeConfig")

	require.Eventually(s.T(), func() bool {
		out, _ := s.sshClient.Run("timedatectl show-timesync")
		return strings.Contains(string(out), fmt.Sprintf("SystemNTPServers=%s", googleNTPServers))
	}, 10*time.Second, 1*time.Second, fmt.Sprintf("NTPServers should be %s", googleNTPServers))
}
