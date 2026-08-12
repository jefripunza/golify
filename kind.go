package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// kindProviderEnv returns the env for kind to use the Podman provider,
// matching the setup in example/golify.sh.
func kindProviderEnv() []string {
	return append(os.Environ(), "KIND_EXPERIMENTAL_PROVIDER=podman")
}

// kindCreateCluster creates a kind cluster named after the project UUID.
// Blocks until the cluster is ready (--wait 120s) so the API returns a
// real, running cluster.
func kindCreateCluster(name string) error {
	log.Printf("[kind] creating cluster %s", name)
	cmd := exec.Command("kind", "create", "cluster", "--name", name, "--wait", "120s")
	cmd.Env = kindProviderEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[kind] create cluster %s failed: %v\n%s", name, err, out)
		return err
	}
	log.Printf("[kind] cluster %s ready", name)
	return nil
}

// kindDeleteCluster deletes a kind cluster by name. Missing clusters are
// ignored (deleting an already-deleted cluster is a no-op).
func kindDeleteCluster(name string) {
	log.Printf("[kind] deleting cluster %s", name)
	cmd := exec.Command("kind", "delete", "cluster", "--name", name)
	cmd.Env = kindProviderEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[kind] delete cluster %s failed: %v\n%s", name, err, out)
	}
}

// kindClusterStatus returns "Running", "Stopped" or "Unknown" for a cluster.
func kindClusterStatus(name string) string {
	cmd := exec.Command("kind", "get", "clusters")
	cmd.Env = kindProviderEnv()
	out, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == name {
			return "Running"
		}
	}
	return "Stopped"
}
