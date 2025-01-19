<p align="center">
<img src="./docs/logo/banner.png" alt="elasti icon">
</p>

<p align="center">
 <a>
    <img src="https://img.shields.io/badge/license-MIT-blue" align="center">
 </a>

</p>

> This project is in Alpha right now.

# Why use Elasti?

Kubernetes clusters can become costly, especially when running multiple services continuously. Elasti addresses this issue by giving you the confidence to scale down services during periods of low or no traffic, as it can bring them back up when demand increases. This optimization minimizes resource usage without compromising on service availability. Additionally, Elasti ensures reliability by acting as a proxy that queues incoming requests for scaled-down services. Once these services are reactivated, Elasti processes the queued requests, so that no request is lost. This combination of cost savings and dependable performance makes Elasti an invaluable tool for efficient Kubernetes service management.

> The name Elasti comes from a superhero "Elasti-Girl" from DC Comics. Her supower is to expand or shrink her body at will—from hundreds of feet tall to mere inches in height.

<div align="center"> <b> Demo </b></div>
<div align="center">
    <a href="https://www.loom.com/share/6dae33a27a5847f081f7381f8d9510e6">
      <img style="max-width:640px;" src="https://cdn.loom.com/sessions/thumbnails/6dae33a27a5847f081f7381f8d9510e6-adf9e85a899f85fd-full-play.gif">
    </a>
  </div>

# Contents

- [Why use Elasti?](#why-use-elasti)
- [Contents](#contents)
- [Introduction](#introduction)
  - [Key Features](#key-features)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Install](#install)
    - [1. Add the Elasti Helm Repository](#1-add-the-elasti-helm-repository)
    - [2. Install Elasti](#2-install-elasti)
    - [3. Verify the Installation](#3-verify-the-installation)
  - [Configuration](#configuration)
    - [1. Define a ElastiService](#1-define-a-elastiservice)
    - [2. Apply the configuration](#2-apply-the-configuration)
    - [3. Check Logs](#3-check-logs)
  - [Monitoring](#monitoring)
  - [Uninstall](#uninstall)
- [Development](#development)
- [Contribution](#contribution)
  - [Getting Started](#getting-started-1)
  - [Getting Help](#getting-help)
  - [Acknowledgements](#acknowledgements)
- [Future Developments](#future-developments)

# Introduction

Elasti monitors the target service for which you want to enable scale-to-zero. When the target service is scaled down to zero, Elasti automatically switches to Proxy mode, redirecting all incoming traffic to itself. In this mode, Elasti queues the incoming requests and scales up the target service. Once the service is back online, Elasti processes the queued requests, sending them to the now-active service. After the target service is scaled up, Elasti switches to Serve mode, where traffic is directly handled by the service, removing any redirection. This seamless transition between modes ensures efficient handling of requests while optimizing resource usage.

<div align="center">
<img src="./docs/assets/modes.png" width="400px">
</div>

## Key Features

- **Seamless Integration:** Elasti integrates effortlessly with your existing Kubernetes setup. It takes just a few steps to enable scale to zero for any service.

- **Development and Argo Rollouts Support:** Elasti supports two target references: Deployment and Argo Rollouts, making it versatile for various deployment scenarios.

- **HTTP API Support:** Currently, Elasti supports only HTTP API types, ensuring straightforward and efficient handling of web traffic.

- **Prometheus Metrics Export:** Elasti exports Prometheus metrics for easy out-of-the-box monitoring. You can also import a pre-built dashboard into Grafana for comprehensive visualization.

- **Istio Support:** Elasti is compatible with Istio. It also supports East-West traffic using cluster-local service DNS, ensuring robust and flexible traffic management across your services.

# Getting Started

With Elasti, you can easily manage and scale your Kubernetes services by using a proxy mechanism that queues and holds requests for scaled-down services, bringing them up only when needed. Get started by follwing below steps:

## Prerequisites

- **Kubernetes Cluster:** You should have a running Kubernetes cluster. You can use any cloud-based or on-premises Kubernetes distribution.
- **kubectl:** Installed and configured to interact with your Kubernetes cluster.
- **Helm:** Installed for managing Kubernetes applications.

## Install

### 1. Install Elasti using helm

Use Helm to install elasti into your Kubernetes cluster. Replace `<release-name>` with your desired release name and `<namespace>` with the Kubernetes namespace you want to use:

```bash
helm install <release-name> oci://tfy.jfrog.io/tfy-helm/elasti --namespace <namespace> --create-namespace
```
Check out [values.yaml](./charts/elasti/values.yaml) to see config in the helm value file.

### 2. Verify the Installation

Check the status of your Helm release and ensure that the elasti components are running:

```bash
helm status <release-name> --namespace <namespace>
kubectl get pods -n <namespace>
```

You will see 2 components running.

1.  **Controller/Operator:** `elasti-operator-controller-manager-...` is to switch the traffic, watch resources, scale etc.
2.  **Resolver:** `elasti-resolver-...` is to proxy the requests.

Refer to the [Docs](./docs/README.md) to know how it works.

## Configuration

To configure a service to handle its traffic via elasti, you'll need to create and apply a `ElastiService` custom resource:

### 1. Define a ElastiService

```yaml
apiVersion: elasti.truefoundry.com/v1alpha1
kind: ElastiService
metadata:
  name: <service-name>
  namespace: <service-namespace>
spec:
  minTargetReplicas: <min-target-replicas>
  service: <service-name>
  scaleTargetRef:
    apiVersion: <apiVersion>
    kind: <kind>
    name: <deployment-or-rollout-name>
```

- `<service-name>`: Replace it with the service you want managed by elasti.
- `<min-target-replicas>`: Min replicas to bring up when first request arrives.
- `<service-namespace>`: Replace by namespace of the service.
- `<scaleTargetRef>`: Reference to the scale target similar to the one used in HorizontalPodAutoscaler.
- `<kind>`: Replace by `rollouts` or `deployments`
- `<apiVersion>`: Replace with `argoproj.io/v1alpha1` or `apps/v1`
- `<deployment-or-rollout-name>`: Replace with name of the rollout or the deployment for the service. This will be scaled up to min-target-replicas when first request comes

### 2. Apply the configuration

Apply the configuration to your Kubernetes cluster:

```bash
kubectl apply -f <service-name>-elasti-CRD.yaml
```

### 3. Check Logs

You can view logs from the controller to watchout for any errors.

```bash
kubectl logs -f deployment/elasti-operator-controller-manager -n <namespace>
```

## Monitoring

During installation, two ServiceMonitor custom resources are created to enable Prometheus to discover the Elasti components. To verify this, you can open your Prometheus interface and search for metrics prefixed with elasti-, or navigate to the Targets section to check if Elasti is listed.

Once verification is complete, you can use the [provided Grafana dashboard](./playground/infra/elasti-dashboard.yaml) to monitor the internal metrics and performance of Elasti.

<div align="center">
<img src="./docs/assets/grafana-dashboard.png" width="800px">
</div>

## Uninstall

To uninstall Elasti, **you will need to remove all the installed ElastiServices first.** Then, simply delete the installation file.

```bash
kubectl delete elastiservices --all
helm uninstall <release-name> -n <namespace>
kubectl delete namespace <namespace>
```

# Development

Refer to [DEVELOPMENT.md](./DEVELOPMENT.md) for more details.

# Contribution

We welcome contributions from the community to improve Elasti. Whether you're fixing bugs, adding new features, improving documentation, or providing feedback, your efforts are appreciated. Follow the steps below to contribute effectively to the project.

## Getting Started

Follows the steps mentioned in [development](#development) section. Post that follow:

1. **Fork the Repository:**
   Fork the Elasti repository to your own GitHub account:

   ```bash
   git clone https://github.com/your-org/elasti.git
   cd elasti
   ```

2. **Create a New Branch:**
   Create a new branch for your changes:

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Code Changes:**
   Make your code changes or additions in the appropriate files and directories. Ensure you follow the project's coding standards and best practices.

4. **Write Tests:**
   Add unit or integration tests to cover your changes. This helps maintain code quality and prevents future bugs.

5. **Update Documentation:**
   If your changes affect the usage of Elasti, update the relevant documentation in README.md or other documentation files.

6. **Sign Your Commits & Push:**
   Sign your commits to certify that you wrote the code and have the right to pass it on as an open-source contribution:

   ```bash
   git commit -s -m "Your commit message"
   git push origin feature/your-feature-name
   ```

7. **Create a Pull Request:**
   Navigate to the original Elasti repository and submit a pull request from your branch. Provide a clear description of your changes and the motivation behind them. If your pull request addresses any open issues, link them in the description. Use keywords like fixes #issue_number to automatically close the issue when the pull request is merged.

8. **Review Process:**
   Your pull request will be reviewed by project maintainers. Be responsive to feedback and make necessary changes. Post review, it will be merged!

<div align="center"> <b> You just contributed to Elasti! </b></div>
<div align="center">

<img src="./docs/assets/awesome.gif" width="400px">
</div>

## Getting Help

If you need help or have questions, feel free to reach out to the community. You can:

- Open an issue for discussion or help.
- Join our community chat or mailing list.
- Refer to the FAQ and Troubleshooting Guide.

## Acknowledgements

Thank you for contributing to Elasti! Your contributions make the project better for everyone. We look forward to collaborating with you.

# Future Developments

- Support GRPC, Websockets.
- Test multiple ports in same service.
- Seperate queue for different services.
- Unit test coverage.
