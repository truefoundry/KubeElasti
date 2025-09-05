# Variables
CLUSTER_NAME := elasti-e2e
NAMESPACE := test
ELASTI_NAMESPACE := elasti
T ?= ./tests
REGISTRY_HOST := localhost:5002
REGISTRY_NAME := kind-registry
ISTIO_VERSION := 1.26.1
PROMETHEUS_VERSION := 73.2.3
PROMETHEUS_PUSHGATEWAY_VERSION := 3.4.1
KEDA_VERSION := 2.17.1
TIMEOUT := 300s

.PHONY: all setup destroy test
all: build-images setup test
setup: kind-up chmod-scripts apply-deps
destroy: kind-down
test:
	@echo "Running KUTTL tests..."
	@kubectl kuttl test --start-kind=false --test $(T)
	@echo "E2E Tests completed!"

.PHONY: chmod-scripts
chmod-scripts:
	@echo "Creating reset executable...
	@chmod +x ./manifest/setup.sh

.PHONY: build-images
build-images: ## Build and push images
	$(MAKE) -C ../../operator docker-build IMG=elasti-operator:v1alpha1
	$(MAKE) -C ../../resolver docker-build IMG=elasti-resolver:v1alpha1

.PHONY: kind-up
kind-up: ## Create Kind cluster
	@echo "Creating Kind cluster '$(CLUSTER_NAME)'..."
	@if ! kind get clusters | grep -q $(CLUSTER_NAME); then \
		kind create cluster --config ./kind-config.yaml; \
		kubectl config use-context kind-$(CLUSTER_NAME); \
		echo "Kind cluster '$(CLUSTER_NAME)' created successfully."; \
	else \
		echo "Kind cluster '$(CLUSTER_NAME)' already exists."; \
		kubectl config use-context kind-$(CLUSTER_NAME); \
	fi
	kind load docker-image elasti-operator:v1alpha1 --name elasti-e2e
	kind load docker-image elasti-resolver:v1alpha1 --name elasti-e2e

.PHONY: kind-down
kind-down:
	@echo "Deleting kind cluster '$(CLUSTER_NAME)'..."
	@if kind get clusters | grep -q $(CLUSTER_NAME); then \
		kind delete cluster --name $(CLUSTER_NAME); \
		echo "Kind cluster '$(CLUSTER_NAME)' deleted successfully."; \
	else \
		echo "Kind cluster '$(CLUSTER_NAME)' does not exist."; \
	fi

.PHONY: apply-deps
apply-deps: setup-repos apply-deps-parallel apply-elasti

.PHONY: apply-deps-parallel
apply-deps-parallel:
	@$(MAKE) apply-keda &
	@$(MAKE) apply-ingress &
	@$(MAKE) apply-prometheus
	@echo "All dependencies installed successfully."

.PHONY: setup-repos
setup-repos:
	@echo "Setting up Helm repositories..."
	@helm repo add istio https://istio-release.storage.googleapis.com/charts
	@helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	@helm repo add kedacore https://kedacore.github.io/charts
	@helm repo update

.PHONY: apply-elasti
apply-elasti:
	@echo "Installing Elasti Operator..."
	@helm upgrade --install --create-namespace elasti ../../charts/elasti -n $(ELASTI_NAMESPACE) -f ./manifest/global/values-elasti.yaml

.PHONY: apply-prometheus
apply-prometheus:
	@echo "Installing Prometheus Stack..."
	@kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack --version $(PROMETHEUS_VERSION) \
	  --namespace monitoring \
	  --create-namespace \
	  --set alertmanager.enabled=false \
	  --set grafana.enabled=true \
	  --set kubeStateMetrics.enabled=false \
	  --set nodeExporter.enabled=false \
	  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false --wait --timeout $(TIMEOUT)
	@kubectl apply -f ./manifest/global/istio-servicemonitor.yaml
	@helm upgrade --install prometheus-pushgateway prometheus-community/prometheus-pushgateway --version $(PROMETHEUS_PUSHGATEWAY_VERSION) \
	  --namespace monitoring \
	  --set serviceMonitor.enabled=true \
	  --set serviceMonitor.interval=5s \
	  --wait --timeout $(TIMEOUT)
	@echo "Prometheus Stack installed successfully."

.PHONY: apply-ingress
apply-ingress:
	@echo "Setting up Istio..."
	@kubectl create namespace target --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install istio-base istio/base -n istio-system --create-namespace --version $(ISTIO_VERSION) --wait --timeout $(TIMEOUT)
	@helm upgrade --install istiod istio/istiod -n istio-system --version $(ISTIO_VERSION) --wait --timeout $(TIMEOUT)
	@helm upgrade --install istio-ingressgateway istio/gateway -n istio-system -f ./manifest/global/values-istio-ingress.yaml --version $(ISTIO_VERSION) --wait --timeout $(TIMEOUT)
	@kubectl apply -f ./manifest/global/istio-gateway.yaml -n istio-system
	@echo "Istio installed successfully."

.PHONY: uninstall-ingress
uninstall-ingress:
	@echo "Uninstalling Istio..."
	@helm uninstall istio-base istiod istio-ingressgateway -n istio-system
	@kubectl delete namespace istio-system
	@echo "Istio uninstalled successfully."

.PHONY: apply-keda
apply-keda: 
	@echo "Installing KEDA..."
	@kubectl create namespace keda --dry-run=client -o yaml | kubectl apply -f -
	@helm upgrade --install keda kedacore/keda --namespace keda --create-namespace --version $(KEDA_VERSION) --wait --timeout $(TIMEOUT)
	@echo "KEDA installed successfully."

# This is for locally testing, not used in CI
.PHONY: uninstall-keda
uninstall-keda:
	@echo "Uninstalling KEDA..."
	@helm uninstall keda -n keda
	@kubectl delete namespace keda
	@echo "KEDA uninstalled successfully."

.PHONY: pf-prom
pf-prom:
	@echo "Portforwarding prometheus on port 9090..."
	@kubectl --namespace monitoring port-forward service/prometheus-operated 9090:9090

.PHONY: pf-graf
pf-graf: 
	@echo "Portforwarding grafana on port 9001..."
	@kubectl --namespace monitoring port-forward service/kube-prometheus-stack-grafana 9001:80

.PHONY: pf-ingress
pf-ingress: 
	@echo "Portforwarding ingress gateway on port 8080..."
	@kubectl --namespace istio-system port-forward service/istio-ingressgateway 8080:80

.PHONY: lint-kube
lint-kube:
	@echo "Linting k8s manifests and kuttl tests..."
	@kube-linter lint ./manifest
	@kube-linter lint ./tests

.PHONY: lint-yaml
lint-yaml:
	@echo "Linting k8s manifests and kuttl tests..."
	@yamllint ./manifest
	@yamllint ./tests
