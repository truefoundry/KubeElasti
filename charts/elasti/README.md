# Kube elasti helm chart
KubeElasti is a Kubernetes-native solution that offers scale-to-zero functionality when there is no traffic and automatic scale up to 1 when traffic arrives. Most Kubernetes autoscaling solutions like HPA or Keda can scale from 1 to n replicas based on cpu utilization or memory usage

## Parameters

### Global parameters for elasti helm chart

| Name                             | Description                                  | Value           |
| -------------------------------- | -------------------------------------------- | --------------- |
| `global.kubernetesClusterDomain` | domain of the Kubernetes cluster             | `cluster.local` |
| `global.nameOverride`            | name of the deployment                       | `""`            |
| `global.fullnameOverride`        | full name of the deployment                  | `""`            |
| `global.enableMonitoring`        | whether to enable monitoring                 | `false`         |
| `global.secretName`              | name of the secret to use for the deployment | `elasti-secret` |
| `global.image.registry`          | registry to use for the deployment           | `tfy.jfrog.io`  |
| `global.imagePullSecrets`        | image pull secrets to use for the deployment | `[]`            |

### Elasti controller parameters

| Name                                                | Description                                  | Value                        |
| --------------------------------------------------- | -------------------------------------------- | ---------------------------- |
| `elastiController.manager.args`                     | arguments to pass to the manager             | `[]`                         |
| `elastiController.manager.containerSecurityContext` | container security context                   | `{}`                         |
| `elastiController.manager.podSecurityContext`       | pod security context                         | `{}`                         |
| `elastiController.manager.image.registry`           | registry to use for the deployment           | `""`                         |
| `elastiController.manager.image.repository`         | repository to use for the deployment         | `tfy-images/elasti-operator` |
| `elastiController.manager.image.tag`                | tag to use for the deployment                | `0.1.18`                     |
| `elastiController.manager.imagePullPolicy`          | image pull policy                            | `IfNotPresent`               |
| `elastiController.manager.resources`                | resources to use for the deployment          | `{}`                         |
| `elastiController.manager.sentry.enabled`           | whether to enable sentry                     | `false`                      |
| `elastiController.manager.sentry.environment`       | environment to use for the deployment        | `""`                         |
| `elastiController.manager.env`                      | environment to use for the deployment        | `{}`                         |
| `elastiController.replicas`                         | number of replicas to use for the deployment | `1`                          |
| `elastiController.serviceAccount.annotations`       | annotations to use for the deployment        | `{}`                         |
| `elastiController.metricsService`                   | metrics service to use for the deployment    | `{}`                         |
| `elastiController.service`                          | service to use for the deployment            | `{}`                         |

### Elasti resolver parameters

| Name                                                        | Description                                                 | Value                        |
| ----------------------------------------------------------- | ----------------------------------------------------------- | ---------------------------- |
| `elastiResolver.proxy.env`                                  | environment to use for the deployment                       | `{}`                         |
| `elastiResolver.proxy.image.registry`                       | registry to use for the deployment                          | `test.jfrog.io`              |
| `elastiResolver.proxy.image.repository`                     | repository to use for the deployment                        | `tfy-images/elasti-resolver` |
| `elastiResolver.proxy.image.tag`                            | tag to use for the deployment                               | `0.1.18`                     |
| `elastiResolver.proxy.imagePullPolicy`                      | image pull policy                                           | `IfNotPresent`               |
| `elastiResolver.proxy.resources`                            | resources to use for the deployment                         | `{}`                         |
| `elastiResolver.proxy.containerSecurityContext`             | container security context                                  | `{}`                         |
| `elastiResolver.proxy.podSecurityContext`                   | pod security context                                        | `{}`                         |
| `elastiResolver.proxy.sentry.enabled`                       | whether to enable sentry                                    | `false`                      |
| `elastiResolver.proxy.sentry.environment`                   | environment to use for the deployment                       | `""`                         |
| `elastiResolver.replicas`                                   | number of replicas to use for the deployment                | `1`                          |
| `elastiResolver.serviceAccount.annotations`                 | annotations to use for the deployment                       | `{}`                         |
| `elastiResolver.autoscaling.enabled`                        | whether to enable autoscaling                               | `false`                      |
| `elastiResolver.autoscaling.minReplicas`                    | minimum number of replicas to use for the deployment        | `1`                          |
| `elastiResolver.autoscaling.maxReplicas`                    | maximum number of replicas to use for the deployment        | `4`                          |
| `elastiResolver.autoscaling.targetCPUUtilizationPercentage` | target CPU utilization percentage to use for the deployment | `70`                         |
| `elastiResolver.reverseProxyService`                        | reverse proxy service to use for the deployment             | `{}`                         |
| `elastiResolver.service`                                    | service to use for the deployment                           | `{}`                         |
