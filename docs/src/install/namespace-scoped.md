---
title: "Namespace-scoped install - run KubeElasti without cluster-wide RBAC"
description: "Install KubeElasti confined to an explicit list of namespaces using namespace-scoped Role/RoleBinding, with no cluster-wide list/watch. Opt-in via global.allowedNamespaces."
keywords:
  - KubeElasti namespace scoped
  - namespace scoped RBAC
  - namespaced operator RBAC
  - allowedNamespaces
  - WATCH_NAMESPACES
  - multi-tenant Kubernetes
icon: lucide/shield-check
---

# Namespace-scoped install

## What it is

By default KubeElasti installs cluster-scoped: a `ClusterRole` and `ClusterRoleBinding` let the operator and resolver watch every namespace.

Namespace-scoped mode is opt-in. You give KubeElasti an explicit list of namespaces. It then installs a `Role` and `RoleBinding` in each one instead of a `ClusterRole`, and the operator only watches those namespaces. There is no cluster-wide list/watch.

## When to use it

- Multi-tenant clusters where a team may only touch its own namespaces.
- Regulated or restricted environments that forbid cluster-wide RBAC.
- You want KubeElasti's blast radius limited to a known set of namespaces.

Use the default cluster-scoped install if none of these apply.

## The CRD is still cluster-scoped

Namespace-scoped mode makes the operator and resolver run with namespaced `Role`/`RoleBinding` only, with no `ClusterRole` at runtime. The one cluster-scoped object that remains is the `ElastiService` CRD. CRDs are always cluster-scoped in Kubernetes.

The chart installs the CRD as part of the release, so installing or upgrading the chart needs permission to manage that CRD. This is a one-time registration, not a standing grant to the operator or resolver.

!!! warning "Operator image version"
    Namespace-scoped mode needs an operator image that supports the `WATCH_NAMESPACES` env var. Use a release that ships this feature. On an older image the chart still renders scoped RBAC, but the operator falls back to cluster-scoped watching and its reads fail under the namespaced `Role`.

## Enable it

Set `global.allowedNamespaces` to the namespaces you want KubeElasti to manage:

```bash
helm install elasti oci://ghcr.io/kubeelasti/charts/elasti \
  --namespace elasti --create-namespace \
  --set 'global.allowedNamespaces={team-a,team-b}'
```

The release namespace is always added to the list automatically. The operator needs it for the leader-election lease and to read the resolver's `EndpointSlices`.

Leaving `global.allowedNamespaces` empty (the default) keeps the cluster-scoped install unchanged.

## What changes

| | Default (`[]`) | Scoped (list set) |
|---|---|---|
| Operator/resolver RBAC | `ClusterRole` + `ClusterRoleBinding` | `Role` + `RoleBinding` per namespace (+ release namespace) |
| Watched scope | all namespaces | only the listed namespaces |
| Operator env | none | `WATCH_NAMESPACES` |
| ClusterRoles created | yes | none |

```mermaid
flowchart TD
    A[helm install] --> B{global.allowedNamespaces}
    B -->|empty, default| C[ClusterRole + ClusterRoleBinding]
    C --> C2[Operator watches ALL namespaces]
    B -->|"team-a, team-b"| D[Role + RoleBinding in team-a, team-b, release-ns]
    D --> D2[WATCH_NAMESPACES env set on operator]
    D2 --> D3[Operator watches ONLY listed namespaces, no ClusterRoles]
```

## Verify

No `ClusterRole` or `ClusterRoleBinding` is created in scoped mode:

```bash
kubectl get clusterrole,clusterrolebinding -l app.kubernetes.io/instance=elasti
# expect: no resources found
```

A `Role` and `RoleBinding` exist in each listed namespace:

```bash
kubectl get role,rolebinding -n team-a -l app.kubernetes.io/instance=elasti
```

The operator has the namespace list:

```bash
kubectl get deploy -n elasti -l app.kubernetes.io/component=operator \
  -o jsonpath='{.items[0].spec.template.spec.containers[0].env}' | grep WATCH_NAMESPACES
```

An `ElastiService` created in a namespace outside the list is ignored by the operator.

## Uninstall

Same as the default install. Remove ElastiServices first, then the release:

```bash
kubectl delete elastiservices --all
helm uninstall elasti -n elasti
```

The CRD is part of the release, so `helm uninstall` removes it too.
