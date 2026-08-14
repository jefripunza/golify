package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// ---------------------------------------------------------------------------
// kind cluster management — SHARED control plane.
//
// Golify manages ONE kind cluster for ALL environments: "golify".
// kind appends "-control-plane" to the node container name automatically
// (kind create cluster --name golify → container "golify-control-plane"),
// so the cluster name itself stays short: "golify".
//
//   cluster golify (container: golify-control-plane)
//   ├── namespace env-<id> (environment "production")
//   │   ├── Deployment golify-<svc> (service)
//   │   │   └── Pod replica × N
//   │   └── Service golify-<svc>
//   └── namespace env-<id2> (environment "staging")
// ---------------------------------------------------------------------------

// k8sClusterName is the single shared kind cluster name (short on purpose —
// kind already appends "-control-plane" to the node container name).
const k8sClusterName = "golify"

// kindProviderEnv returns the env for kind to use the Podman provider,
// matching the setup in example/golify.sh.
func kindProviderEnv() []string {
	return append(os.Environ(), "KIND_EXPERIMENTAL_PROVIDER=podman")
}

// kindCreateCluster creates the shared kind cluster if it does not already
// exist. Blocks until the cluster is ready (--wait 120s). Missing clusters
// are created; existing clusters are left untouched (idempotent).
func kindCreateCluster() error {
	if kindClusterStatus() == "Running" {
		log.Printf("[kind] cluster %s already running", k8sClusterName)
		return nil
	}
	log.Printf("[kind] creating shared cluster %s", k8sClusterName)
	cmd := exec.Command("kind", "create", "cluster", "--name", k8sClusterName, "--wait", "120s")
	cmd.Env = kindProviderEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[kind] create cluster %s failed: %v\n%s", k8sClusterName, err, out)
		return err
	}
	log.Printf("[kind] cluster %s ready", k8sClusterName)
	return nil
}

// kindDeleteCluster deletes the shared kind cluster. Missing clusters are
// ignored (deleting an already-deleted cluster is a no-op). Used only by
// maintenance/admin flows — deleting an environment never deletes the cluster.
func kindDeleteCluster() {
	log.Printf("[kind] deleting shared cluster %s", k8sClusterName)
	cmd := exec.Command("kind", "delete", "cluster", "--name", k8sClusterName)
	cmd.Env = kindProviderEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[kind] delete cluster %s failed: %v\n%s", k8sClusterName, err, out)
	}
}

// kindClusterStatus returns "Running", "Stopped" or "Unknown" for the shared
// cluster.
func kindClusterStatus() string {
	cmd := exec.Command("kind", "get", "clusters")
	cmd.Env = kindProviderEnv()
	out, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == k8sClusterName {
			return "Running"
		}
	}
	return "Stopped"
}

// kindClusterIP returns the internal IP of the control-plane node for the
// shared cluster, or "" if the cluster is not reachable. Uses kubectl with
// the cluster's kubeconfig context (kind-golify-control-plane). NOTE: with
// the Podman provider the node name is "<name>-control-plane" (no "kind-"
// prefix), so we look it up by label instead of hardcoding the node name.
func kindClusterIP() string {
	cmd := exec.Command("kubectl", "get", "node",
		"-l", "node-role.kubernetes.io/control-plane",
		"-o", "jsonpath={.items[0].status.addresses[?(@.type==\"InternalIP\")].address}")
	cmd.Env = kindProviderEnv()
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[kind] get ip for %s failed: %v", k8sClusterName, err)
		return ""
	}
	ip := strings.TrimSpace(string(out))
	log.Printf("[kind] cluster %s internal IP = %s", k8sClusterName, ip)
	return ip
}
