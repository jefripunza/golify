package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// ---------------------------------------------------------------------------
// Kubernetes (kubectl) helpers — "full K8s" mode.
//
// Golify acts as a control plane that shells out to `kubectl` for everything
// container-related when a service is in K8s mode (LoadBalancer == "k8s"):
//   - deploy   → kubectl apply -f <Deployment + Service manifest>
//   - scale    → kubectl scale deployment/<name> --replicas=N
//   - stop     → kubectl scale --replicas=0   / start → --replicas=1
//   - restart  → kubectl rollout restart deployment/<name>
//   - list     → kubectl get pods -l app=<name>
//   - logs     → kubectl logs -f <pod>
//   - terminal → kubectl exec -it <pod> -- /bin/sh
//
// Domain/ingress routing stays in proxy.go (Golify's own nginx/traefik
// replacement) — that is the part we keep custom by design.
// ---------------------------------------------------------------------------

// k8sEnabled reports whether a service runs in full-K8s mode.
func k8sEnabled(svc Service) bool {
	return svc.LoadBalancer == "k8s"
}

// k8sServiceName returns the stable K8s resource name for a service
// (Deployment, Service, pods label). DNS-safe, lowercase, dashes.
func k8sServiceName(svc Service) string {
	base := "golify-" + strings.ToLower(strings.ReplaceAll(svc.Name, " ", "-"))
	// keep it short & DNS-safe (<63 chars)
	if len(base) > 60 {
		base = base[:60]
	}
	return base
}

// k8sNamespace is where Golify-managed workloads live.
const k8sNamespace = "default"

// k8sEnv returns the environment for kubectl: it needs the kind cluster's
// kubeconfig. If KUBECONFIG is set in the environment we honor it; otherwise
// we rely on the default ~/.kube/config (kubectl's own default).
func k8sEnv() []string {
	env := os.Environ()
	if os.Getenv("KUBECONFIG") == "" {
		// default config is fine; no need to inject anything
	}
	return env
}

// kubectl runs a kubectl command and returns its combined output.
func kubectl(args ...string) (string, error) {
	cmd := exec.Command("kubectl", append([]string{"--namespace", k8sNamespace}, args...)...)
	cmd.Env = k8sEnv()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// k8sDeployYAML renders the Deployment + Service manifest for a service.
// image = svc.Image + ":" + tag; replicas = svc.Replicas; container port 80.
func k8sDeployYAML(svc Service) string {
	name := k8sServiceName(svc)
	img := svc.Image
	if img == "" {
		img = svc.Name
	}
	if !strings.Contains(img, ":") {
		img += ":" + firstNonEmpty(svc.ImageTag, "latest")
	}
	replicas := svc.Replicas
	if replicas < 1 {
		replicas = 1
	}
	labels := fmt.Sprintf("app: %s", name)
	var b strings.Builder
	b.WriteString("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n")
	fmt.Fprintf(&b, "  name: %s\n  labels:\n    %s\n", name, labels)
	b.WriteString("spec:\n")
	fmt.Fprintf(&b, "  replicas: %d\n", replicas)
	b.WriteString("  selector:\n    matchLabels:\n")
	fmt.Fprintf(&b, "      app: %s\n", name)
	b.WriteString("  template:\n    metadata:\n      labels:\n")
	fmt.Fprintf(&b, "        app: %s\n", name)
	b.WriteString("    spec:\n      containers:\n")
	fmt.Fprintf(&b, "      - name: %s\n        image: %s\n", name, img)
	b.WriteString("        ports:\n        - containerPort: 80\n")
	b.WriteString("---\napiVersion: v1\nkind: Service\nmetadata:\n")
	fmt.Fprintf(&b, "  name: %s\n", name)
	b.WriteString("spec:\n  selector:\n")
	fmt.Fprintf(&b, "    app: %s\n", name)
	b.WriteString("  ports:\n  - port: 80\n    targetPort: 80\n")
	return b.String()
}

// k8sApply deploys the service manifest (Deployment + Service) via kubectl apply.
func k8sApply(svc Service) error {
	yaml := k8sDeployYAML(svc)
	cmd := exec.Command("kubectl", "apply", "--namespace", k8sNamespace, "-f", "-")
	cmd.Env = k8sEnv()
	cmd.Stdin = strings.NewReader(yaml)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	log.Printf("[k8s] applied %s: %s", k8sServiceName(svc), strings.TrimSpace(string(out)))
	return nil
}

// k8sScale sets the deployment replicas to n.
func k8sScale(svc Service, n int) error {
	name := k8sServiceName(svc)
	out, err := kubectl("scale", "deployment/"+name, "--replicas", fmt.Sprintf("%d", n))
	if err != nil {
		return fmt.Errorf("kubectl scale: %v (%s)", err, strings.TrimSpace(out))
	}
	log.Printf("[k8s] scaled %s → %d (%s)", name, n, strings.TrimSpace(out))
	return nil
}

// k8sRestart does a rolling restart of the deployment.
func k8sRestart(svc Service) error {
	name := k8sServiceName(svc)
	out, err := kubectl("rollout", "restart", "deployment/"+name)
	if err != nil {
		return fmt.Errorf("kubectl rollout restart: %v (%s)", err, strings.TrimSpace(out))
	}
	log.Printf("[k8s] restarted %s (%s)", name, strings.TrimSpace(out))
	return nil
}

// k8sPods lists running pods for the service. Returns replica entries with
// id (pod name), name (pod name), replica_id (12-char pod uid hash),
// status, running, ports ("" for pods).
func k8sPods(svc Service) []fiber.Map {
	name := k8sServiceName(svc)
	out, err := kubectl("get", "pods", "-l", "app="+name, "-o", "json")
	if err != nil {
		log.Printf("[k8s] get pods %s: %v", name, err)
		return []fiber.Map{}
	}
	var parsed struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
				UID  string `json:"uid"`
			} `json:"metadata"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		log.Printf("[k8s] parse pods %s: %v", name, err)
		return []fiber.Map{}
	}
	rows := []fiber.Map{}
	for _, p := range parsed.Items {
		running := p.Status.Phase == "Running"
		rid := p.Metadata.UID
		if len(rid) > 12 {
			rid = rid[:12]
		}
		status := p.Status.Phase
		if !running {
			status = "NotReady"
		}
		rows = append(rows, fiber.Map{
			"id":         p.Metadata.Name,
			"name":       p.Metadata.Name,
			"replica_id": rid,
			"status":     status,
			"running":    running,
			"ports":      "",
		})
	}
	return rows
}

// k8sLogStream returns the kubectl logs -f command for a pod, or nil if the
// pod doesn't exist.
func k8sLogStream(pod string) (*exec.Cmd, error) {
	// sanity: pod exists?
	out, err := kubectl("get", "pod", pod, "-o", "name")
	if err != nil || !strings.Contains(out, "pod/") {
		return nil, fmt.Errorf("pod %q not found", pod)
	}
	cmd := exec.Command("kubectl", "logs", "-f", "--namespace", k8sNamespace, pod)
	cmd.Env = k8sEnv()
	return cmd, nil
}

// k8sExec returns a kubectl exec -it <pod> -- /bin/sh command.
func k8sExec(pod string) *exec.Cmd {
	cmd := exec.Command("kubectl", "exec", "-it", "--namespace", k8sNamespace, pod, "--", "/bin/sh")
	cmd.Env = k8sEnv()
	return cmd
}

// k8sWaitReady waits up to timeout for the deployment to reach n ready replicas.
func k8sWaitReady(svc Service, n int, timeout time.Duration) error {
	name := k8sServiceName(svc)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := kubectl("get", "deployment/"+name, "-o", "jsonpath={.status.readyReplicas}")
		if err == nil && strings.TrimSpace(out) == fmt.Sprintf("%d", n) {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("deployment %s not ready after %v", name, timeout)
}
