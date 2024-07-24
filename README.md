<p align="center">
<img src="./docs/logo/logo_512.png" alt="elasti icon" width="100">
</p>

<h1 align="center">Elasti</h1>
<h5 align="center">Serverless capabilities in Kubernetes :twisted_rightwards_arrows: :arrow_double_up:</h5>

<p align="center">
 <a>
    <img src="https://goreportcard.com/badge/github.com/truefoundry/elasti" align="center">
 </a>
 <a>
    <img src="https://img.shields.io/badge/godoc-reference-green" align="center">
 </a>
 <a>
    <img src="https://img.shields.io/badge/license-MIT-blue" align="center">
 </a>

</p>


# Elasti

Elasti is a cloud-native tool to facilitate serverless capabilities within Kubernetes services. It intercepts and queues requests directed to a service that has been scaled down to zero instances, then scales the service up, and subsequently forwards the queued requests to the now active service.


>  The name Elasti comes from a superhero "Elasti-Girl" from DC Comics. Her supower is to expand or shrink her body at will—from hundreds of feet tall to mere inches in height. 

![Arch](./docs/assets/elasti-hld.png)

# Problem Statement
TBA

# Installation / Deployment on K8s

You can install the Elasti Tool by running the following command:
```bash
make deploy

or 

helm install elasti ./charts/elasti -n elasti
```

After this, you can start creating elastiService. You can find a sample at `demo-elastiService.yaml`.

## Uninstallation 

To uninstall Elasti, **you will need to remove all the CRDs first.** Then, simply delete the installation file. 
```bash
make undeploy

or 

helm uninstall elasti -n elasti
```

# Development

Here are the steps to deploy Elasti in your Kubernetes cluster. For now, we will need to deploy the Elasti-Operator and Elasti-Resolver separately.

### Setup Docker

This is required to publish our Docker images and pull them inside our manifest files. 

1. Login to the Docker Hub via CLI.
```bash
docker login -u ramantehlan
# When asked for password, paste below text.
dckr_pat_WgMJEsO0nMNp10Bf7rLQ_FcVzLU
``` 

> That's it, now you will be able to push changes to images. Since the images are public, pulling it doesn't require any changes.

### Build Resolver

We will build and publish our resolver changes.

1. Go into the resolver directory. 
2. Run the build and publish command.
```bash
make docker-buildx IMG=ramantehlan/elasti-resolver:v1alpha1
```

### Build Operator

We will build and publish our Operator changes.

1. Go into the operator directory.
2. Run the build and publish command.
```bash
make docker-buildx IMG=ramantehlan/elasti-operator:v1alpha1
```

> Once your changes are published, you can re-deploy them in your cluster.

# Configuration
TBA

# Playground 

```
make docker-build docker-publish IMG=localhost:5001/elasti-operator:v1alpha1
make docker-build docker-publish IMG=localhost:5001/elasti-resolver:v1alpha1
```

# Icon 

The icon is <a href="https://www.flaticon.com/free-icons/full-screen" title="full-screen icons">Full-screen icon created by Uniconlabs - Flaticon</a>. 



---

```
Get argo rollout type locally
kubectl apply -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

```

```
helm upgrade --install promstack prometheus-community/kube-prometheus-stack -f prom-stack.yaml


helm upgrade --install prometheus prometheus-community/prometheus --version 25.24.0 -f prometheus.yaml -n prometheus  

helm upgrade --install grafana grafana/grafana -n prometheus


kubectl get secret --namespace prometheus grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo

kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_alertmanagers.yaml
kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml
kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_probes.yaml
kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_prometheuses.yaml
kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml
kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
kubectl apply -f example/prometheus-operator-crd/monitoring.coreos.com_thanosrulers.yaml


```


