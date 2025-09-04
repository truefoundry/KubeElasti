#!/bin/sh

echo "Current directory: $(pwd)"

# Parse arguments - only use flags
MANIFEST_DIR=""
NAMESPACE=""

while [ $# -gt 0 ]; do
  case $1 in
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --manifest-dir)
      MANIFEST_DIR="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 --manifest-dir <path> --namespace <namespace>"
      exit 1
      ;;
  esac
done

# Validate required arguments
if [ -z "$MANIFEST_DIR" ]; then
  echo "Error: --manifest-dir is required"
  exit 1
fi

if [ -z "$NAMESPACE" ]; then
  echo "Error: --namespace is required"
  exit 1
fi

echo "Using manifest directory: $MANIFEST_DIR"
echo "Using namespace: $NAMESPACE"

# Function to substitute template variables and apply
apply_template() {
  local template_file="$1"
  local target_namespace="$2"
  
  echo "Applying template: $(basename "$template_file")"
  
  # Substitute all variables and apply
  sed -e "s/\${NAMESPACE}/$target_namespace/g" "$template_file" | kubectl apply -f -
}

# 1. Apply target deployment
apply_template "$MANIFEST_DIR/test-template/target-deployment.yaml" "$NAMESPACE"
kubectl wait --for=condition=Ready pods -l app=target-deployment -n $NAMESPACE

# 2. Apply keda ScaledObject in KEDA for Target  
apply_template "$MANIFEST_DIR/test-template/keda-scaledObject-Target.yaml" "$NAMESPACE"
kubectl wait --for=condition=Ready scaledobject/target-scaled-object -n $NAMESPACE

# 3. Apply ElastiService
apply_template "$MANIFEST_DIR/test-template/target-elastiservice.yaml" "$NAMESPACE"
kubectl wait --for=jsonpath='{.status}' elastiservice/target-elastiservice -n $NAMESPACE 

# 4. Add virtual service (goes to istio-system but references our namespace)
# apply_template "$MANIFEST_DIR/test-template/target-virtualService.yaml" "$NAMESPACE"

# Label namespace for istio injection
kubectl label namespace $NAMESPACE istio-injection=enabled --overwrite
