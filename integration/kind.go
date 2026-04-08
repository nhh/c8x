//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/cluster"
)

const clusterName = "c8x-integration"

// CreateTestCluster creates a KinD cluster and returns the kubeconfig path.
func CreateTestCluster() (string, error) {
	provider := cluster.NewProvider()

	fmt.Println("Creating KinD cluster...")
	err := provider.Create(clusterName)
	if err != nil {
		return "", fmt.Errorf("creating kind cluster: %w", err)
	}

	kubeconfig, err := provider.KubeConfig(clusterName, false)
	if err != nil {
		return "", fmt.Errorf("getting kubeconfig: %w", err)
	}

	// Write kubeconfig to temp file
	kubeconfigPath := filepath.Join(os.TempDir(), "c8x-integration-kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0600); err != nil {
		return "", fmt.Errorf("writing kubeconfig: %w", err)
	}

	return kubeconfigPath, nil
}

// DeleteTestCluster removes the KinD cluster.
func DeleteTestCluster() error {
	fmt.Println("Deleting KinD cluster...")
	provider := cluster.NewProvider()
	return provider.Delete(clusterName, "")
}
