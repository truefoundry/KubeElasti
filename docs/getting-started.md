# Getting Started

With Elasti, you can easily manage and scale your Kubernetes services by using a proxy mechanism that queues and holds requests for scaled-down services, bringing them up only when needed. Get started by following below steps:

## Prerequisites

- **Kubernetes Cluster:** You should have a running Kubernetes cluster. You can use any cloud-based or on-premises Kubernetes distribution.
- **kubectl:** Installed and configured to interact with your Kubernetes cluster.
- **Helm:** Installed for managing Kubernetes applications.

## Install

### 1. Install Elasti using helm

Use Helm to install elasti into your Kubernetes cluster. Replace `<release-name>` with your desired release name and `<namespace>` with the Kubernetes namespace you want to use:

```bash
helm install elasti oci://tfy.jfrog.io/tfy-helm/elasti --namespace elasti --create-namespace
```
Check out [values.yaml](./charts/elasti/values.yaml) to see config in the helm value file.

### 2. Verify the Installation

Check the status of your Helm release and ensure that the elasti components are running:

```bash
helm status elasti --namespace elasti
kubectl get pods -n elasti
```

You will see 2 components running.

1.  **Controller/Operator:** `elasti-operator-controller-manager-...` is to switch the traffic, watch resources, scale etc.
2.  **Resolver:** `elasti-resolver-...` is to proxy the requests.

### 3. Setup Prometheus

We will setup a sample prometheus to read metrics from the nginx ingress controller.

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set alertmanager.enabled=false \
  --set grafana.enabled=false \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

### 4. Setup nginx ingress controller

We will setup a nginx ingress controller to route the traffic to the httpbin service.

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install nginx-ingress ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --set controller.metrics.enabled=true \
  --set controller.metrics.serviceMonitor.enabled=true \
  --create-namespace
```

This will deploy a nginx ingress controller in the `ingress-nginx` namespace.

### 5. Setup a Service

We will use a sample httpbin service to demonstrate how to configure a service to handle its traffic via elasti.

```bash
kubectl create namespace elasti-demo
kubectl apply -n elasti-demo -f ../playground/config/demo-application.yaml
```

This will deploy a httpbin service in the `elasti-demo` namespace.

### 6. Define an ElastiService

To configure a service to handle its traffic via elasti, you'll need to create and apply a `ElastiService` custom resource:
```yaml
apiVersion: elasti.truefoundry.com/v1alpha1
kind: ElastiService
metadata:
  name: <service-name>-elasti
  namespace: <service-namespace>
spec:
  minTargetReplicas: 1
  service: <service-name>
  cooldownPeriod: 300 
  scaleTargetRef:
    apiVersion: <api-version>
    kind: <kind>
    name: <deployment-or-rollout-name>
  triggers:
    - type: <trigger-type>
      metadata:
        query: <prometheus-query>
        serverAddress: <prometheus-server-address>
        threshold: <threshold>
  autoscaler:
    type: <autoscaler-type>
    name: <autoscaler-object-name>
```

- `<service-name>`: Replace it with the service you want managed by elasti.
- `<min-target-replicas>`: Min replicas to bring up when first request arrives.
- `<service-namespace>`: Replace by namespace of the service.
- `<scaleTargetRef>`: Reference to the scale target similar to the one used in HorizontalPodAutoscaler.
- `<kind>`: Replace by `rollouts` or `deployments`
- `<apiVersion>`: Replace with `argoproj.io/v1alpha1` or `apps/v1`
- `<deployment-or-rollout-name>`: Replace with name of the rollout or the deployment for the service. This will be scaled up to min-target-replicas when first request comes
- `cooldownPeriod`: Minimum time (in seconds) to wait after scaling up before considering scale down
- `triggers`: List of conditions that determine when to scale down (currently supports only Prometheus metrics)
- `autoscaler`: **Optional** integration with an external autoscaler (HPA/KEDA) if needed
  - `<autoscaler-type>`: hpa/keda
  - `<autoscaler-object-name>`: name of the KEDA ScaledObject or HPA HorizontalPodAutoscaler object
  
Create a file named `httpbin-elasti.yaml` and apply the configuration.
```yaml
apiVersion: elasti.truefoundry.com/v1alpha1
kind: ElastiService
metadata:
  name: httpbin-elasti
  namespace: elasti-demo
spec:
  minTargetReplicas: 1
  service: httpbin
  cooldownPeriod: 5
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployments
    name: httpbin
  triggers:
    - type: prometheus
      metadata:
        query: sum(rate(nginx_ingress_controller_nginx_process_requests_total[1m])) or vector(0)
        serverAddress: http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090
        threshold: "0.5"
```

### 7. Apply the elasti service configuration

Apply the configuration to your Kubernetes cluster:

```bash
kubectl apply -f httpbin-elasti.yaml -n elasti-demo
```

The pod will be scaled down to 0 replicas if there is no traffic.

### 8. Test the setup

You can test the setup by sending requests to the nginx load balancer service.

```bash
kubectl port-forward svc/nginx-ingress-nginx-controller -n ingress-nginx 8080:80
```

Start a watch on the httpbin service.

```bash
kubectl get pods -n elasti-demo -w
```

Send a request to the httpbin service.

```bash
curl -v http://localhost:8080/httpbin
```

You should see the pods being created and scaled up to 1 replica. A response from the httpbin service should be visible for the curl command.
The service should be scaled down to 0 replicas if there is no traffic for `cooldownPeriod` seconds.

## Uninstall

To uninstall Elasti, **you will need to remove all the installed ElastiServices first.** Then, simply delete the installation file.

```bash
kubectl delete elastiservices --all
helm uninstall elasti -n elasti
kubectl delete namespace elasti
```