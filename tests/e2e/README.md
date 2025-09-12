# Elasti E2E Testing Framework

This directory contains an End-to-End (E2E) testing framework for the Elasti Kubernetes operator system using KUTTL (KUbernetes Test TooL). The framework provides a comprehensive way to validate Elasti's core functionality in a realistic Kubernetes environment.

## Requirements

### System Requirements

- Docker (for Kind)
- Go 1.20+
- Make

### Tools

- [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker) v0.17.0+
- [kubectl](https://kubernetes.io/docs/tasks/tools/) v1.25.0+
- [kuttl](https://kuttl.dev/) v0.15.0+
- [jq](https://jqlang.org/) v1.7+

  ```bash
    # Using Krew (Kubernetes plugin manager)
    kubectl krew install kuttl

    # Or using Go
    go install github.com/kudobuilder/kuttl/cmd/kubectl-kuttl@latest
  ```

### Development Environment

- Linux, macOS, or WSL2 on Windows
- At least 4GB of free memory for the Kind cluster
- At least 10GB of free disk space

## Directory Structure

The E2E testing framework is organized into the following directories:

```bash
elasti/tests/e2e/
├── tests/                     # KUTTL test definitions
│   └── 00-<TEST-NAME>/      # Individual test case
│       └── 00-<STEP-NAME>.yaml    # Test step (assertion)
├── manifest/                  # Kubernetes manifests and values files
│   ├── global/                # Global values files
│   ├── test-template/         # Test setup template files
│   └── setup.sh               # Test setup script
├── kind-config.yaml           # Kind cluster configuration
├── Makefile                   # Test automation commands
├── kuttl-test.yaml            # KUTTL test suite configuration
└── README.md                  # This file
```

- **`manifest/`**: Contains Kubernetes manifest files and Helm values files
- **`tests/`**: Contains the actual KUTTL tests
  - Each directory represents an individual test.
  - Each file within a test directory represents a step in that test.
  - Each step follows the naming convention with prefix 00-, 01-, etc. for execution ordering.
- **`kind-config.yaml`**: Configuration for the Kind cluster used in testing
- **`kuttl-test.yaml`**: Configuration file for KUTTL tests
  - Contains commands that run before test execution

## Test Scenarios

The framework includes the following test scenarios:

1. **Setup Check**: Test if setup is in desired state.
2. **Enable Proxy Mode**: Tests Elasti's ability to switch to proxy mode when scaling a deployment to zero.
3. **Enable Serve Mode**: Tests Elasti's ability to switch back to serve mode after receiving traffic.

## Running Tests

### Quick Start

> Note: Docker daemon should be running for the tests to work.

To run the complete test suite:

```bash
// Build images for operator and resolver
// Always run this step after making changes to operator or resolver
make build-images

// Setup the environment
// Run this only first time
make setup

// Run all tests
make test

or

// Run specific test
make test T=00-elasti-setup
```

### Individual Commands

You can also run specific parts of the testing process:

| Command                  | Description                                                                                                                                                                                                    |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `make build-images`      | Build and push Elasti operator and resolver images to local registry |
| `make setup`             | Sets up the environment (registry and Kind cluster with dependencies) |
| `make test`              | Run the KUTTL E2E tests |
| `make reload-images`     | Use the newly built image of Elasti operator and resolver images |
| `make destroy`           | Delete Kind cluster and stop registry |
| `make kind-up`           | Create a Kind cluster with the name `elasti-e2e`|
| `make kind-down`         | Delete the Kind cluster |
| `make apply-deps`        | Install all dependencies (Istio, Prometheus, Elasti) |
| `make apply-elasti`      | Install only the Elasti operator and CRDs |
| `make apply-prometheus`  | Install only Prometheus (with Grafana) |
| `make apply-ingress`     | Install only Istio ingress gateway |
| `make apply-keda`        | Install only KEDA |
| `make uninstall-ingress` | Uninstall Istio components |
| `make uninstall-keda`    | Uninstall KEDA components |
| `make pf-prom`           | Port-forward the Prometheus service to localhost:9090 |
| `make pf-graf`           | Port-forward the Grafana service to localhost:9001 |
| `make pf-ingress`        | Port-forward the ingress gateway service to localhost:8080 |
