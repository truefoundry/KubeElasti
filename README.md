

<div align="center">
<img src="./docs/assets/images/logo/rounded_black_white_bg.png" alt="KubeElasti Logo" width="200px">
<h1> KubeElasti</h1>
<h3> Confidently Scale-to-zero with no downtime</h3>
</div>

<div align="center">



![Lint and Test](https://github.com/KubeElasti/KubeElasti/actions/workflows/lint-and-test.yaml/badge.svg)
[![codecov](https://codecov.io/gh/KubeElasti/KubeElasti/branch/main/graph/badge.svg)](https://codecov.io/gh/KubeElasti/KubeElasti)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/kubeelasti)](https://artifacthub.io/packages/search?repo=kubeelasti)
[![CNCF Landscape](https://img.shields.io/badge/CNCF%20Landscape-5699C6)](https://landscape.cncf.io/?item=serverless--framework--kubeelasti)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/12491/badge)](https://www.bestpractices.dev/projects/12491)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/KubeElasti/KubeElasti/badge)](https://scorecard.dev/viewer/?uri=github.com/KubeElasti/KubeElasti)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FKubeElasti%2FKubeElasti.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2FKubeElasti%2FKubeElasti?ref=badge_shield&issueType=license)
![License](https://img.shields.io/badge/license-MIT-blue)
[![Docs](https://img.shields.io/badge/Docs-kubeelasti.dev-blue)](https://kubeelasti.dev)
[![Static Badge](https://img.shields.io/badge/slack-@cloud_native/kubeelasti-blue.svg?logo=slack )](https://cloud-native.slack.com/archives/C0AMUFC5Y3D)
</div>

# Why use KubeElasti?

Kubernetes clusters can become costly, especially when running multiple services continuously. KubeElasti addresses this issue by giving you the confidence to scale down services during periods of low or no traffic, as it can bring them back up when demand increases. This optimization minimizes resource usage without compromising on service availability. Additionally, KubeElasti ensures reliability by acting as a proxy that queues incoming requests for scaled-down services. Once these services are reactivated, KubeElasti processes the queued requests, so that no request is lost. This combination of cost savings and dependable performance makes KubeElasti an invaluable tool for efficient Kubernetes service management.

> The name Elasti comes from a superhero "Elasti-Girl" from DC Comics. Her superpower is to expand or shrink her body at will—from hundreds of feet tall to mere inches in height. Kube just refers to kubernetes. Elasti powers in kubernetes! 

> KubeElasti(Sometimes referred to as just "Elasti").

<div align="center">
  <img src="./docs/assets/images/intro.png" alt="Illustration of KubeElasti's active vs. serverless modes">
</div>

# Contents

- [Why use KubeElasti?](#why-use-kubeelasti)
- [Contents](#contents)
- [Introduction](#introduction)
  - [Key Features](#key-features)
- [Getting Started](#getting-started)
- [Configure KubeElasti](#configure-kubeelasti)
- [Monitoring](#monitoring)
- [Development](#development)
- [Contribution](#contribution)
- [Governance](#governance)
- [Adopters](#adopters)
- [Getting Help](#getting-help)
- [Roadmap](#roadmap)
- [Contributors](#contributors)
- [Project Supporters](#project-supporters)

# Introduction

KubeElasti is a Kubernetes-native solution that offers scale-to-zero functionality when there is no traffic and automatic scale up to 1 when traffic arrives. Most Kubernetes autoscaling solutions like HPA or Keda can scale from 1 to n replicas based on cpu utilization or memory usage. However, these solutions do not offer a way to scale to 0 when there is no traffic. KubeElasti solves this problem by dynamically managing service replicas based on real-time traffic conditions. It only handles scaling the application down to 0 replicas and scaling it back up to 1 replica when traffic is detected again. The scaling after 1 replica is handled by the autoscaler like HPA or Keda.

KubeElasti uses a proxy mechanism that queues and holds requests for scaled-down services, bringing them up only when needed. The proxy is used only when the service is scaled down to 0. When the service is scaled up to 1, the proxy is disabled and the requests are processed directly by the pods of the service.

<div align="center">
<img src="./docs/assets/images/modes.png" width="400px">
</div>

## Key Features

- **Seamless Integration:** KubeElasti integrates effortlessly with your existing Kubernetes setup - whether you are using HPA or Keda. It takes just a few steps to enable scale to zero for any service.

- **Deployment, StatefulSet, Argo Rollouts Support:** KubeElasti supports three scale target references: Deployment, StatefulSet and Argo Rollouts, making it versatile for various deployment scenarios.

- **Prometheus Metrics Export:** KubeElasti exports Prometheus metrics for easy out-of-the-box monitoring. You can also import a pre-built dashboard into Grafana for comprehensive visualization.

- **Generic Service Support:** KubeElasti works at the kubernetes service level. It also supports East-West traffic using cluster-local service DNS, ensuring robust and flexible traffic management across your services. So any ingress or service mesh solution can be used with KubeElasti.

- **Autoscaler Integration:** KubeElasti can work seamlessly with [HPA](./docs/src/documentation/get-started/scalers.md#scaling-with-hpa) and [Keda](./docs/src/documentation/get-started/scalers.md#scaling-with-keda).

# Getting Started

Details on how to install and configure KubeElasti can be found in the [installation and demo](./docs/src/install/demo-setup.md) guide.

# Configure KubeElasti

Check out the different ways to configure KubeElasti in the [Configuration](./docs/src/install/configure-elastiservice.md) guide.

# Monitoring

Monitoring details can be found in the [Monitoring](./docs/src/documentation/get-started/monitoring.md) guide.

# Development

Refer to [DEVELOPMENT.md](./DEVELOPMENT.md) for more details.

# Contribution

Contribution details can be found in the [Contribution](./CONTRIBUTING.md) guide.

# Governance

Project governance details, including roles, decision-making, and maintainer responsibilities, are documented in [GOVERNANCE.md](./GOVERNANCE.md). The formal voting process is in [VOTING.md](./VOTING.md).

# Adopters

See the list of organizations who are using KubeElasti in Production or in Staging in the [Adopters](./docs/src/about/adopters.md) guide.

# Getting Help

We have a dedicated [Discussions](https://github.com/KubeElasti/KubeElasti/discussions) section for getting help and discussing ideas.

**KubeElasti Community Day** is a public community call on the second Thursday of each month, 19:00-19:30 Asia/Kolkata (13:30-14:00 UTC). Join on [Google Meet](https://meet.google.com/wns-mqdn-veb). Details live in [GOVERNANCE.md](./GOVERNANCE.md#communication).

# Roadmap

We are maintaining the future roadmap using the [issues](https://github.com/KubeElasti/KubeElasti/issues) and [milestones](https://github.com/KubeElasti/KubeElasti/milestones). You can also suggest ideas and vote for them by adding a 👍 reaction to the issue.

# Contributors

KubeElasti is built by a growing community of contributors. Thank you to everyone who has helped shape the project!

<a href="https://github.com/KubeElasti/KubeElasti/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=KubeElasti/KubeElasti" alt="KubeElasti contributors" />
</a>

We are actively looking for additional reviewers and maintainers to help grow and steer the project. If you are interested in taking on a larger role, apply with the [role application form](https://github.com/KubeElasti/KubeElasti/issues/new?template=role_application.yml). The bar is 2 merged PRs and a month of activity for maintainer, 1 merged PR and a month for reviewer. Not there yet? Say hi on [Slack](https://cloud-native.slack.com/archives/C0AMUFC5Y3D) or start a thread in our [Discussions](https://github.com/KubeElasti/KubeElasti/discussions). See the [Contribution](#contribution) and [Governance](#governance) guides to get started.

# Project Supporters

KubeElasti is supported by [TrueFoundry](https://www.truefoundry.com/), which contributed the project to the [Cloud Native Computing Foundation (CNCF)](https://www.cncf.io/).
