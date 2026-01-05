# Kube elasti helm chart
KubeElasti is a Kubernetes-native solution that offers scale-to-zero functionality when there is no traffic and automatic scale up to 1 when traffic arrives. Most Kubernetes autoscaling solutions like HPA or Keda can scale from 1 to n replicas based on cpu utilization or memory usage

## Parameters

### Global parameters for elasti helm chart

| Name                                | Description                                  | Value           |
| ----------------------------------- | -------------------------------------------- | --------------- |
| `global.kubernetesClusterDomain`    | domain of the Kubernetes cluster             | `cluster.local` |
| `global.nameOverride`               | name of the deployment                       | `""`            |
| `global.fullnameOverride`           | full name of the deployment                  | `""`            |
| `global.enableMonitoring`           | whether to enable monitoring                 | `false`         |
| `global.secretName`                 | name of the secret to use for the deployment | `elasti-secret` |
| `global.image.registry`             | registry to use for the deployment           | `tfy.jfrog.io`  |
| `global.imagePullSecrets`           | image pull secrets to use for the deployment | `[]`            |
| `global.labels`                     | labels to apply to all resources             | `{}`            |
| `global.annotations`                | annotations to apply to all resources        | `{}`            |
| `global.podLabels`                  | labels to apply to all pods                  | `{}`            |
| `global.podAnnotations`             | annotations to apply to all pods             | `{}`            |
| `global.serviceLabels`              | labels to apply to all services              | `{}`            |
| `global.serviceAnnotations`         | annotations to apply to all services         | `{}`            |
| `global.deploymentLabels`           | labels to apply to all deployments           | `{}`            |
| `global.deploymentAnnotations`      | annotations to apply to all deployments      | `{}`            |
| `global.serviceAccount`             | service account configuration                | `{}`            |
| `global.serviceAccount.annotations` | annotations to apply to all service accounts | `{}`            |
| `global.serviceAccount.labels`      | labels to apply to all service accounts      | `{}`            |

### Elasti controller parameters

| Name                                                | Description                                            | Value                        |
| --------------------------------------------------- | ------------------------------------------------------ | ---------------------------- |
| `elastiController.manager.args`                     | arguments to pass to the manager                       | `[]`                         |
| `elastiController.manager.containerSecurityContext` | container security context                             | `{}`                         |
| `elastiController.manager.podSecurityContext`       | pod security context                                   | `{}`                         |
| `elastiController.manager.image.registry`           | registry to use for the deployment                     | `""`                         |
| `elastiController.manager.image.repository`         | repository to use for the deployment                   | `tfy-images/elasti-operator` |
| `elastiController.manager.image.tag`                | tag to use for the deployment                          | `0.1.19`                     |
| `elastiController.manager.imagePullPolicy`          | image pull policy                                      | `IfNotPresent`               |
| `elastiController.manager.resources`                | resources to use for the deployment                    | `{}`                         |
| `elastiController.manager.sentry.enabled`           | whether to enable sentry                               | `false`                      |
| `elastiController.manager.sentry.environment`       | environment to use for the deployment                  | `""`                         |
| `elastiController.manager.env`                      | environment to use for the deployment                  | `{}`                         |
| `elastiController.replicas`                         | number of replicas to use for the deployment           | `1`                          |
| `elastiController.commonLabels`                     | labels to apply to all elastiController resources      | `{}`                         |
| `elastiController.commonAnnotations`                | annotations to apply to all elastiController resources | `{}`                         |
| `elastiController.podLabels`                        | labels to apply to elastiController pods               | `{}`                         |
| `elastiController.podAnnotations`                   | annotations to apply to elastiController pods          | `{}`                         |
| `elastiController.deploymentLabels`                 | labels to apply to elastiController deployment         | `{}`                         |
| `elastiController.deploymentAnnotations`            | annotations to apply to elastiController deployment    | `{}`                         |
| `elastiController.serviceAccount.annotations`       | annotations to use for the deployment                  | `{}`                         |
| `elastiController.serviceAccount.labels`            | labels to use for the service account                  | `{}`                         |
| `elastiController.metricsService`                   | metrics service to use for the deployment              | `{}`                         |
| `elastiController.metricsService.labels`            | labels to apply to metricsService                      | `{}`                         |
| `elastiController.metricsService.annotations`       | annotations to apply to metricsService                 | `{}`                         |
| `elastiController.service`                          | service to use for the deployment                      | `{}`                         |
| `elastiController.service.labels`                   | labels to apply to service                             | `{}`                         |
| `elastiController.service.annotations`              | annotations to apply to service                        | `{}`                         |
| `elastiController.serviceMonitor`                   | serviceMonitor configuration                           | `{}`                         |
| `elastiController.serviceMonitor.labels`            | labels to apply to serviceMonitor                      | `{}`                         |
| `elastiController.serviceMonitor.annotations`       | annotations to apply to serviceMonitor                 | `{}`                         |

### Elasti resolver parameters

| Name                                                        | Description                                                 | Value                        |
| ----------------------------------------------------------- | ----------------------------------------------------------- | ---------------------------- |
| `elastiResolver.proxy.env`                                  | environment to use for the deployment                       | `{}`                         |
| `elastiResolver.proxy.image.registry`                       | registry to use for the deployment                          | `""`                         |
| `elastiResolver.proxy.image.repository`                     | repository to use for the deployment                        | `tfy-images/elasti-resolver` |
| `elastiResolver.proxy.image.tag`                            | tag to use for the deployment                               | `0.1.19`                     |
| `elastiResolver.proxy.imagePullPolicy`                      | image pull policy                                           | `IfNotPresent`               |
| `elastiResolver.proxy.resources`                            | resources to use for the deployment                         | `{}`                         |
| `elastiResolver.proxy.containerSecurityContext`             | container security context                                  | `{}`                         |
| `elastiResolver.proxy.podSecurityContext`                   | pod security context                                        | `{}`                         |
| `elastiResolver.proxy.sentry.enabled`                       | whether to enable sentry                                    | `false`                      |
| `elastiResolver.proxy.sentry.environment`                   | environment to use for the deployment                       | `""`                         |
| `elastiResolver.replicas`                                   | number of replicas to use for the deployment                | `1`                          |
| `elastiResolver.commonLabels`                               | labels to apply to all elastiResolver resources             | `{}`                         |
| `elastiResolver.commonAnnotations`                          | annotations to apply to all elastiResolver resources        | `{}`                         |
| `elastiResolver.podLabels`                                  | labels to apply to elastiResolver pods                      | `{}`                         |
| `elastiResolver.podAnnotations`                             | annotations to apply to elastiResolver pods                 | `{}`                         |
| `elastiResolver.deploymentLabels`                           | labels to apply to elastiResolver deployment                | `{}`                         |
| `elastiResolver.deploymentAnnotations`                      | annotations to apply to elastiResolver deployment           | `{}`                         |
| `elastiResolver.serviceAccount.annotations`                 | annotations to use for the deployment                       | `{}`                         |
| `elastiResolver.serviceAccount.labels`                      | labels to use for the service account                       | `{}`                         |
| `elastiResolver.autoscaling.enabled`                        | whether to enable autoscaling                               | `false`                      |
| `elastiResolver.autoscaling.minReplicas`                    | minimum number of replicas to use for the deployment        | `1`                          |
| `elastiResolver.autoscaling.maxReplicas`                    | maximum number of replicas to use for the deployment        | `4`                          |
| `elastiResolver.autoscaling.targetCPUUtilizationPercentage` | target CPU utilization percentage to use for the deployment | `70`                         |
| `elastiResolver.reverseProxyService`                        | reverse proxy service to use for the deployment             | `{}`                         |
| `elastiResolver.service`                                    | service to use for the deployment                           | `{}`                         |
| `elastiResolver.service.labels`                             | labels to apply to service                                  | `{}`                         |
| `elastiResolver.service.annotations`                        | annotations to apply to service                             | `{}`                         |
| `elastiResolver.serviceMonitor`                             | serviceMonitor configuration                                | `{}`                         |
| `elastiResolver.serviceMonitor.labels`                      | labels to apply to serviceMonitor                           | `{}`                         |
| `elastiResolver.serviceMonitor.annotations`                 | annotations to apply to serviceMonitor                      | `{}`                         |
