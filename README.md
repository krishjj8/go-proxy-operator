# go-proxy-operator

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go&logoColor=white)
![Kubebuilder](https://img.shields.io/badge/Kubebuilder-v4-326ce5?style=flat&logo=kubernetes)
![License](https://img.shields.io/badge/license-Apache%202.0-blue?style=flat)

A Kubernetes Operator built with **Kubebuilder** and **controller-runtime** that manages the full lifecycle of [go-reverse-proxy](https://github.com/krishjj8/go-reverse-proxy) instances. Submit a single `ProxyService` custom resource; the operator provisions the Deployment, Service, and ConfigMap — and continuously heals any configuration drift.

---

## How it works

The operator implements the Kubernetes control-plane / data-plane split cleanly:

- **Control plane (this operator):** watches `ProxyService` resources and reconciles the required Kubernetes objects. It is completely unaware of live HTTP traffic.
- **Data plane (go-reverse-proxy):** runs independently, processing requests, routing via Host header, enforcing the circuit breaker and rate limiter. It has no knowledge of the operator loop.

```
ProxyService CR  ──▶  go-proxy-operator (reconciler)
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
          Deployment       Service        ConfigMap
              │
              ▼
      go-reverse-proxy pods
      (round-robin, circuit breaker, rate limiter)
```

Any manual edit to a managed resource (Deployment, Service, ConfigMap) is reverted on the next reconcile loop — the CR is the single source of truth.

---

## The ProxyService resource

```yaml
apiVersion: networking.krish.platform/v1alpha1
kind: ProxyService
metadata:
  name: edge-gateway
  namespace: default
  labels:
    app.kubernetes.io/name: go-proxy-operator
    app.kubernetes.io/managed-by: kustomize
spec:
  replicas: 2
  listenPort: 8080
  rateLimit: 25
  upstreams:
    - "payment-svc-stable:8001"
    - "payment-svc-canary:8001"
```

Applying this one file causes the operator to create and own:
- a `Deployment` running the proxy image
- a `ClusterIP` `Service` with a dynamically allocated virtual IP (never hardcoded)
- a `ConfigMap` containing the generated `config.yaml`

All three resources carry the label `proxy-instance: edge-gateway`, so you can query them together:

```bash
kubectl get deployments,services,configmaps -l proxy-instance=edge-gateway
```

---

## Local development

```bash
# Generate deepcopy hooks and CRD manifests from API types
make manifests

# Install the CRDs into the cluster
make install

# Verify
kubectl get crds | grep krish

# Run the controller loop locally (watches live cluster events)
make run
```

---

## Full demo (two-terminal split)

**Terminal 1 — operator**
```bash
cd ~/Documents/go-proxy-operator
make manifests && make install && make run
```

**Terminal 2 — data plane + traffic**
```bash
# Build and load the proxy image
cd ~/Documents/go-reverse-proxy
docker build -t go-reverse-proxy:latest .
kind load docker-image go-reverse-proxy:latest

# Apply backends and the ProxyService CR
kubectl apply -f backends.yaml
cd ~/Documents/go-proxy-operator
kubectl apply -f config/samples/networking_v1alpha1_proxyservice.yaml

# Confirm the operator created all three child resources
kubectl get deployments,services,configmaps -l proxy-instance=edge-gateway

# Open the tunnel
kubectl port-forward svc/edge-gateway-service 8080:8080

# Route a request through the proxy
curl -H "Host: api.proxy" http://localhost:8080/
```

---

## Deploy to a cluster

```bash
# Build and push the operator image
make docker-build docker-push IMG=<your-registry>/go-proxy-operator:v1.0.0

# Deploy CRDs + controller
make deploy IMG=<your-registry>/go-proxy-operator:v1.0.0

# Or generate a single combined manifest
make build-installer IMG=<your-registry>/go-proxy-operator:v1.0.0
kubectl apply -f dist/install.yaml
```

---

## Key implementation notes

**Top-level ObjectMeta labels**
The label `proxy-instance: <name>` is set on the `ObjectMeta` of the Deployment, Service, and ConfigMap — not just inside the pod template or selector. Without this, `kubectl get -l proxy-instance=...` returns nothing, because the query matches resource metadata, not pod specs.

**Dynamic ClusterIP allocation**
The Service spec sets `Type: corev1.ServiceTypeClusterIP` but leaves the `ClusterIP` field empty. The API server assigns an available address from the cluster's Service CIDR at creation time. Hardcoding a static IP risks collisions and breaks portability across clusters.

---

## Project layout

```
go-proxy-operator/
├── api/
│   └── v1alpha1/
│       └── proxyservice_types.go      # CRD schema and spec/status types
├── internal/
│   └── controller/
│       └── proxyservice_controller.go # reconcile loop + child resource factories
├── config/
│   ├── crd/                           # generated CRD manifests
│   ├── manager/                       # controller-manager deployment
│   ├── rbac/                          # ClusterRole / binding
│   └── samples/                       # example ProxyService CR
└── Makefile
```

---

## Contributing

1. Edit the API types in `api/v1alpha1/proxyservice_types.go`.
2. Update reconciliation logic in `internal/controller/proxyservice_controller.go`.
3. Regenerate manifests and tidy deps:

```bash
make manifests
go mod tidy
```

---

## Prerequisites

- Go 1.26+
- Docker or Podman
- kind cluster with Cilium installed (see [go-reverse-proxy](https://github.com/krishjj8/go-reverse-proxy) setup)
- kubectl + Kustomize

---

## License

Apache 2.0
