#!/bin/sh

# Enhanced traffic testing script with comprehensive debugging
set -u

# Validate input
if [ -z "$1" ]; then
    echo "ERROR: No URL provided. Usage: $0 <url>"
    exit 3
fi

URL="$1"
CURL_POD_NAME="curl-target-gw"
CURL_NAMESPACE="default"
TARGET_NAMESPACE=""
TARGET_RESOURCE=""
TARGET_NAME=""
MAX_RETRIES=5
TIMEOUT=160

# --- Argument Parsing ---
shift # Shift past the URL argument
while [ "$#" -gt 0 ]; do
    case "$1" in
        --namespace)
            TARGET_NAMESPACE="$2"
            shift 2
            ;;
        --target-resource)
            TARGET_RESOURCE="$2"
            shift 2
            ;;
        --target-name)
            TARGET_NAME="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

if [ -z "$TARGET_NAMESPACE" ]; then
    echo "${RED}ERROR: --namespace flag is required.${NC}"
    exit 5
fi

if [ -z "$TARGET_RESOURCE" ] || [ -z "$TARGET_NAME" ]; then
    echo "${RED}ERROR: --target-resource and --target-name flags are required.${NC}"
    exit 5
fi

# --- Color Definitions ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# --- Helper Functions ---

# log_failure_details: Centralized function to log detailed debugging information on failure
log_failure_details() {
    start_time="${1}"

    echo "${RED}--- DETAILED FAILURE ANALYSIS ---${NC}"

    # Target EndpointSlice Status
    echo "${CYAN}  Status of target Private EndpointSlice in ${TARGET_NAMESPACE}:${NC}"
    kubectl get endpointslice -n "$TARGET_NAMESPACE"   -l endpointslice.kubernetes.io/managed-by=endpointslice-controller.k8s.io -o yaml | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve EndpointSlice status${NC}"

    # Ingress Logs
    echo "${CYAN}  Logs from ingress:${NC}"
    kubectl logs -n istio-system deployments/istiod --since-time="${start_time}" | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve ingress logs${NC}"

    # Resolver Logs
    echo "${CYAN}  Logs from elasti-resolver:${NC}"
    kubectl logs -n elasti services/elasti-resolver-service --since-time="${start_time}" --all-pods=true | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve resolver logs${NC}"

    # Controller Logs
    echo "${CYAN}  Logs from elasti-controller:${NC}"
    kubectl logs -n elasti services/elasti-operator-controller-service --since-time="${start_time}" | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve controller logs${NC}"

    # Target Logs
    echo "${CYAN}  Logs from target (${TARGET_RESOURCE}/${TARGET_NAME}):${NC}"
    kubectl logs -n "$TARGET_NAMESPACE" "${TARGET_RESOURCE}/${TARGET_NAME}"  --since-time="${start_time}" | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve target pod logs${NC}"

    # Verbose Curl Request
    echo "${CYAN}  Attempting verbose request for more details...${NC}"
    kubectl exec -n "$CURL_NAMESPACE" "$CURL_POD_NAME" -- curl --max-time 10 -v -s "$URL" 2>&1 | head -20 | sed 's/^/    /' || echo "${YELLOW}  - Verbose request failed${NC}"

    echo "${RED}-----------------------------------${NC}"
}

echo "${CYAN}=== Traffic Test Configuration ===${NC}"
echo "  ${CYAN}Target URL:${NC}  $URL"
echo "  ${CYAN}Curl Pod:${NC}    $CURL_POD_NAME (in $CURL_NAMESPACE namespace)"
echo "  ${CYAN}Target App:${NC}  app=httpbin (in $TARGET_NAMESPACE namespace)"
echo "  ${CYAN}Retries:${NC}     $MAX_RETRIES"
echo "  ${CYAN}Timeout:${NC}     ${TIMEOUT}s"
echo "  ${CYAN}Timestamp:${NC}   $(date)"
echo "${CYAN}==================================${NC}"

# Check if CURL pod exists and is ready
echo "Checking CURL pod status..."
if ! kubectl get pod "$CURL_POD_NAME" -n "$CURL_NAMESPACE" >/dev/null 2>&1; then
    echo "${RED}ERROR: CURL Pod $CURL_POD_NAME not found in namespace $CURL_NAMESPACE${NC}"
    kubectl get pods -n "$CURL_NAMESPACE" || true
    exit 4
fi

POD_STATUS=$(kubectl get pod "$CURL_POD_NAME" -n "$CURL_NAMESPACE" -o jsonpath='{.status.phase}')
echo "${GREEN}CURL Pod status: $POD_STATUS${NC}"

if [ "$POD_STATUS" != "Running" ]; then
    echo "${YELLOW}WARNING: CURL Pod is not in Running state. Describing pod...${NC}"
    kubectl describe pod "$CURL_POD_NAME" -n "$CURL_NAMESPACE" || true
fi

echo ""
echo "${CYAN}--- Starting Traffic Test ---${NC}"

failure_count=0

for i in $(seq 1 $MAX_RETRIES); do
    echo "\n${CYAN}--- Request $i/$MAX_RETRIES ---${NC}"
    echo "  ${CYAN}Time:${NC} $(date)"

    echo "  ${CYAN}Executing: kubectl exec -n $CURL_NAMESPACE $CURL_POD_NAME -- curl --max-time $TIMEOUT -s -o /dev/null -w \"%{http_code}\" \"$URL\""
    start_time=$(date +%s)
    start_time_rfc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    code=$(kubectl exec -n "$CURL_NAMESPACE" "$CURL_POD_NAME" -- curl \
        --max-time "$TIMEOUT" \
        -s \
        -o /dev/null \
        -w "%{http_code}" \
        "$URL" 2>&1)
    result=$?
    end_time=$(date +%s)
    duration=$((end_time - start_time))

    echo "  ${CYAN}Curl exit code:${NC} $result"
    echo "  ${CYAN}HTTP status code:${NC} $code"
    echo "  ${CYAN}Request duration:${NC} ${duration}s"

    # Detailed error analysis
    if [ "$result" != "0" ]; then
        echo "ERROR: Curl command failed with exit code $result"
        case $result in
            1) echo "  - Unsupported protocol or failed to initialize" ;;
            2) echo "  - Failed to initialize" ;;
            3) echo "  - URL malformed" ;;
            6) echo "  - Couldn't resolve host" ;;
            7) echo "  - Failed to connect to host" ;;
            28) echo "  - Operation timeout" ;;
            *) echo "  - Unknown curl error" ;;
        esac

        log_failure_details "${start_time_rfc}"
        failure_count=$((failure_count + 1))
        continue
    fi

    if [ "$code" != "200" ]; then
        echo "ERROR: Expected HTTP 200, got $code"
        case $code in
            000) echo "  - No response received (connection failed)" ;;
            404) echo "  - Not Found - endpoint may not be available" ;;
            500) echo "  - Internal Server Error" ;;
            502) echo "  - Bad Gateway" ;;
            503) echo "  - Service Unavailable" ;;
            504) echo "  - Gateway Timeout" ;;
            *) echo "  - HTTP error response" ;;
        esac

        log_failure_details "${start_time_rfc}"
        failure_count=$((failure_count + 1))
        continue
    fi

    echo "${GREEN}SUCCESS: Request $i completed successfully (HTTP $code in ${duration}s)${NC}"

    if [ $i -lt $MAX_RETRIES ]; then
        sleep 1
    fi
    echo ""
done


echo "\n${CYAN}=== Test Summary ===${NC}"
if [ "$failure_count" -gt 0 ]; then
    echo "${RED}Test FAILED with $failure_count failed requests out of $MAX_RETRIES.${NC}"
    echo "  ${CYAN}Target:${NC}      $URL"
    echo "  ${CYAN}Completed at:${NC} $(date)"
    echo "${CYAN}====================${NC}"
    exit 1
else
    echo "${GREEN}All $MAX_RETRIES requests completed successfully.${NC}"
    echo "  ${CYAN}Target:${NC}      $URL"
    echo "  ${CYAN}Completed at:${NC} $(date)"
    echo "${CYAN}====================${NC}"
fi
