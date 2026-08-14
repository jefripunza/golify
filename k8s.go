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

// envNamespace returns the Kubernetes namespace for an environment.
// The shared cluster hosts one namespace per environment.
func envNamespace(envID string) string {
	// DNS-safe: namespaces must be lowercase alphanumeric + '-'.
	ns := "env-" + strings.ToLower(strings.ReplaceAll(envID, "_", "-"))
	if len(ns) > 63 {
		ns = ns[:63]
	}
	return ns
}

// k8sNamespaceForService resolves the namespace for a service's environment.
func k8sNamespaceForService(svc Service) string {
	return envNamespace(string(svc.EnvironmentID))
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
// DEPRECATED: namespaces are per-environment now — use envNamespace(envID).
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

// kubectl runs a kubectl command with an explicit namespace and returns its
// combined output.
func kubectlNS(ns string, args ...string) (string, error) {
	cmd := exec.Command("kubectl", append([]string{"--namespace", ns}, args...)...)
	cmd.Env = k8sEnv()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// kubectl runs a kubectl command in the default namespace (legacy callers).
func kubectl(args ...string) (string, error) {
	return kubectlNS(k8sNamespace, args...)
}

// ensureNamespace creates the env namespace if it doesn't exist.
func ensureNamespace(ns string) error {
	out, err := kubectl("get", "namespace", ns, "-o", "name")
	if err == nil && strings.Contains(out, "namespace/") {
		return nil
	}
	out, err = kubectl("create", "namespace", ns)
	if err != nil {
		return fmt.Errorf("create namespace %s: %v (%s)", ns, err, strings.TrimSpace(out))
	}
	log.Printf("[k8s] namespace %s ready", ns)
	return nil
}

// k8sDeployYAML renders the Deployment + Service manifest for a service.
// image = svc.Image + ":" + tag; replicas = svc.Replicas; container port 80.
func k8sDeployYAML(svc Service) string {
	name := k8sServiceName(svc)
	ns := k8sNamespaceForService(svc)
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
	fmt.Fprintf(&b, "  name: %s\n  namespace: %s\n  labels:\n    %s\n", name, ns, labels)
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
	fmt.Fprintf(&b, "  name: %s\n  namespace: %s\n", name, ns)
	b.WriteString("spec:\n  selector:\n")
	fmt.Fprintf(&b, "    app: %s\n", name)
	b.WriteString("  ports:\n  - port: 80\n    targetPort: 80\n")
	return b.String()
}

// k8sApplyIngress creates/updates the Ingress for a service's domains.
// One Ingress per service, with a host rule per ServiceDomain. The Ingress
// routes via the shared ingress-nginx controller (installed once per cluster).
func k8sApplyIngress(svc Service) error {
	domains := svc.Domains
	if len(domains) == 0 {
		return nil // no domains → no Ingress needed
	}
	name := k8sServiceName(svc)
	ns := k8sNamespaceForService(svc)
	var b strings.Builder
	b.WriteString("apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n")
	fmt.Fprintf(&b, "  name: %s\n  namespace: %s\n", name, ns)
	b.WriteString("  annotations:\n    nginx.ingress.kubernetes.io/rewrite-target: /\n")
	b.WriteString("spec:\n  ingressClassName: nginx\n  rules:\n")
	for _, d := range domains {
		host := strings.TrimSpace(d.Host)
		if host == "" {
			continue
		}
		port := d.Port
		if port == "" {
			port = "80"
		}
		fmt.Fprintf(&b, "  - host: %s\n    http:\n      paths:\n      - path: /\n        pathType: Prefix\n        backend:\n          service:\n            name: %s\n            port:\n              number: %s\n", host, name, port)
	}
	cmd := exec.Command("kubectl", "apply", "--namespace", ns, "-f", "-")
	cmd.Env = k8sEnv()
	cmd.Stdin = strings.NewReader(b.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply ingress: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	log.Printf("[k8s] applied Ingress %s in %s: %s", name, ns, strings.TrimSpace(string(out)))
	return nil
}

// k8sApply deploys the service manifest (Deployment + Service) via kubectl apply.
func k8sApply(svc Service) error {
	yaml := k8sDeployYAML(svc)
	ns := k8sNamespaceForService(svc)
	if err := ensureNamespace(ns); err != nil {
		return err
	}
	cmd := exec.Command("kubectl", "apply", "--namespace", ns, "-f", "-")
	cmd.Env = k8sEnv()
	cmd.Stdin = strings.NewReader(yaml)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	log.Printf("[k8s] applied %s in %s: %s", k8sServiceName(svc), ns, strings.TrimSpace(string(out)))
	return nil
}

// k8sScale sets the deployment replicas to n.
func k8sScale(svc Service, n int) error {
	name := k8sServiceName(svc)
	ns := k8sNamespaceForService(svc)
	out, err := kubectlNS(ns, "scale", "deployment/"+name, "--replicas", fmt.Sprintf("%d", n))
	if err != nil {
		return fmt.Errorf("kubectl scale: %v (%s)", err, strings.TrimSpace(out))
	}
	log.Printf("[k8s] scaled %s in %s → %d (%s)", name, ns, n, strings.TrimSpace(out))
	return nil
}

// k8sRestart does a rolling restart of the deployment.
func k8sRestart(svc Service) error {
	name := k8sServiceName(svc)
	ns := k8sNamespaceForService(svc)
	out, err := kubectlNS(ns, "rollout", "restart", "deployment/"+name)
	if err != nil {
		return fmt.Errorf("kubectl rollout restart: %v (%s)", err, strings.TrimSpace(out))
	}
	log.Printf("[k8s] restarted %s in %s (%s)", name, ns, strings.TrimSpace(out))
	return nil
}

// k8sPods lists running pods for the service. Returns replica entries with
// id (pod name), name (pod name), replica_id (12-char pod uid hash),
// status, running, ports ("" for pods).
func k8sPods(svc Service) []fiber.Map {
	name := k8sServiceName(svc)
	ns := k8sNamespaceForService(svc)
	out, err := kubectlNS(ns, "get", "pods", "-l", "app="+name, "-o", "json")
	if err != nil {
		log.Printf("[k8s] get pods %s in %s: %v", name, ns, err)
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

// k8sLogStream returns the kubectl logs -f command for a pod in a namespace,
// or an error if the pod doesn't exist.
func k8sLogStream(ns, pod string) (*exec.Cmd, error) {
	// sanity: pod exists?
	out, err := kubectlNS(ns, "get", "pod", pod, "-o", "name")
	if err != nil || !strings.Contains(out, "pod/") {
		return nil, fmt.Errorf("pod %q not found in %s", pod, ns)
	}
	cmd := exec.Command("kubectl", "logs", "-f", "--namespace", ns, pod)
	cmd.Env = k8sEnv()
	return cmd, nil
}

// k8sExec returns a command that shells into a pod in a namespace.
// kubectl exec needs a TTY for interactive use, but our WS path pipes stdio
// (no TTY). We wrap kubectl in `script -qec` (util-linux) which allocates a
// PTY for the child, so `-it` works and the WS pipe stays interactive.
// Falls back to plain `kubectl exec -i` if `script` is unavailable.
func k8sExec(ns, pod string) *exec.Cmd {
	inner := "kubectl exec -it --namespace " + shellQuote(ns) + " " + shellQuote(pod) + " -- /bin/sh"
	if _, err := exec.LookPath("script"); err == nil {
		return exec.Command("script", "-qec", inner, "/dev/null")
	}
	cmd := exec.Command("kubectl", "exec", "-i", "--namespace", ns, pod, "--", "/bin/sh")
	cmd.Env = k8sEnv()
	return cmd
}

// shellQuote wraps s in single quotes for use inside a shell -c string.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// k8sWaitReady waits up to timeout for the deployment to reach n ready replicas.
func k8sWaitReady(svc Service, n int, timeout time.Duration) error {
	name := k8sServiceName(svc)
	ns := k8sNamespaceForService(svc)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := kubectlNS(ns, "get", "deployment/"+name, "-o", "jsonpath={.status.readyReplicas}")
		if err == nil && strings.TrimSpace(out) == fmt.Sprintf("%d", n) {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("deployment %s in %s not ready after %v", name, ns, timeout)
}
