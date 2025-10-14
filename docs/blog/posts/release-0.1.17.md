---
date: 2025-10-14
pin: true
title: Release 0.1.17
description: Release 0.1.17 of KubeElasti is now available. This release includes a number of new features and improvements.
keywords: 
    - KubeElasti Release 0.1.17
    - scale-to-zero
    - cost optimization
    - kubernetes scaling
author: 
    - KubeElasti Team
slug: release-0.1.17
---

# Release 0.1.17

**KubeElasti v0.1.17** is out, and we are excited! This release brings significant improvements to KubeElasti, and marks a major milestone in making KubeElasti more versatile and production-ready for diverse workloads.


## **Major New Feature: StatefulSet Support**

The highlight of this release is **native StatefulSet support** as a scale target. You can now enable scale-to-zero functionality for StatefulSets just as easily as you do for Deployments and Argo Rollouts.

### How to Use StatefulSet Scaling

Simply configure your ElastiService with a StatefulSet:

```yaml
apiVersion: elasti.truefoundry.com/v1alpha1
kind: ElastiService
metadata:
  name: my-stateful-service
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: StatefulSet  # ← Now supported!
    name: my-statefulset
  # ... rest of your configuration
```
!!! info "Behind the scenes"

    We've fundamentally changed how KubeElasti handles scaling by switching to Kubernetes' standard `/scale` subresource. This means we can support scale-to-zero on any resource that supports the `/scale` subresource.

    Want scale-to-zero support for other Kubernetes resources? Simply [open an issue on GitHub](https://github.com/truefoundry/KubeElasti/issues/new/choose) with your use case.

## **Other Improvements**
- **Environment variable forwarding**: Helm values can now be passed through environment variables for better configuration management.
- **Improved test stability**: Enhanced reliability of our automated testing suite.
- **New adopters page**: Added documentation showcasing organizations using KubeElasti in production.
- **Updated dependencies**: Upgraded Go modules and third-party dependencies for better security and performance.

Shoutout to [Shubham](https://github.com/shubhamrai1993) and [Rethil](https://github.com/rethil) for their contributions to this release.

## **Upgrade**

To upgrade to v0.1.17, update your Helm chart:

```bash
helm repo update
helm upgrade kubeelasti truefoundry/elasti --version 0.1.17
```

Ready to save costs with scale-to-zero? [Get started with KubeElasti today!](https://kubeelasti.dev/src/gs-setup/) 

[💬 Join our Discord](https://discord.gg/qFyN73htgE)

