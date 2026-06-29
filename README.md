# go-proxy-operator

A Kubernetes Operator that automates the deployment, configuration, and lifecycle management of **go-reverse-proxy** instances. Built with **Kubebuilder** and **controller-runtime**, it acts as the control plane for managing Layer 7 reverse proxy data planes.

---

# Features

* **Declarative Management** using the `ProxyService` Custom Resource.
* **Automatic Configuration** through generated ConfigMaps.
* **Self-Healing** by continuously reconciling Deployments, Services, and ConfigMaps.
* **Separation of Concerns** where the operator manages infrastructure while proxy instances handle traffic.

---

# Architecture

```
ProxyService (CRD)
        │
        ▼
go-proxy-operator (Controller)
        │
        ├── Deployment
        ├── Service
        └── ConfigMap
                │
                ▼
        go-reverse-proxy Pods
```

The operator watches `ProxyService` resources and automatically creates or updates the required Kubernetes objects. Any manual changes to managed resources are reverted during reconciliation.

---

# Prerequisites

* Go 1.24+
* Docker or Podman
* Kubernetes cluster (Kind or Minikube)
* kubectl
* Kustomize

---

# Local Development

### Generate manifests

```sh
make manifests
```

### Install the CRD

```sh
kubectl apply -f config/crd/bases/
```

### Verify installation

```sh
kubectl get crds | grep krish
```

### Run the controller locally

```sh
make run
```

---

# Deploy the Operator

### Build and push the image

```sh
make docker-build docker-push IMG=<your-registry>/go-proxy-operator:v1.0.0
```

### Deploy to Kubernetes

```sh
make deploy IMG=<your-registry>/go-proxy-operator:v1.0.0
```

---

# Create a ProxyService

```sh
kubectl apply -f config/samples/networking_v1alpha1_proxyservice.yaml
```

The operator will automatically create and manage the required Deployment, Service, and ConfigMap.

---

# Build a Single Installer

Generate a combined installation manifest:

```sh
make build-installer IMG=<your-registry>/go-proxy-operator:v1.0.0
```

Install it using:

```sh
kubectl apply -f dist/install.yaml
```

---

# Project Structure

```
api/
├── v1alpha1/
│   └── proxyservice_types.go

internal/
└── controller/
    └── proxyservice_controller.go

config/
├── crd/
├── manager/
├── rbac/
└── samples/
```

---

# Contributing

1. Update the API in `api/v1alpha1/proxyservice_types.go`.
2. Modify reconciliation logic in `internal/controller/proxyservice_controller.go`.
3. Regenerate manifests:

```sh
make manifests
go mod tidy
```

---

# License

Licensed under the Apache License 2.0.
