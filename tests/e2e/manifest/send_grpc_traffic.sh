#!/bin/bash

# Enhanced gRPC traffic testing script with comprehensive debugging
set -u

# --- Color Definitions ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# --- Default Values ---
ADDR=""
TEST_TYPE="both"
CLIENT_NAMESPACE=""
TARGET_RESOURCE=""
TARGET_NAME=""
CLIENT_POD_NAME="grpc-client-pod"
MAX_RETRIES=5
RETRY_SLEEP=10
# MAX_FAILURES: number of failed requests that are tolerated before declaring the
# test as failed.  The default (0) means all requests must succeed.  Use 1 for
# proxy-trigger steps where the very first request may arrive before the target
# pod's Istio sidecar is ready (cold-start window of ~25–45 s).
MAX_FAILURES=0

# --- Argument Parsing ---
while [ "$#" -gt 0 ]; do
    case "$1" in
        --addr)
            ADDR="$2"
            shift 2
            ;;
        --test)
            TEST_TYPE="$2"
            shift 2
            ;;
        --namespace)
            CLIENT_NAMESPACE="$2"
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
        --max-failures)
            MAX_FAILURES="$2"
            shift 2
            ;;
        *)
            echo "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 --addr <host:port> --namespace <ns> --target-resource <type> --target-name <name> [--test <unary|stream|both>] [--max-failures <n>]"
            exit 1
            ;;
    esac
done

# --- Validate Required Arguments ---
if [ -z "$ADDR" ]; then
    echo "${RED}ERROR: --addr flag is required.${NC}"
    exit 1
fi

if [ -z "$CLIENT_NAMESPACE" ]; then
    echo "${RED}ERROR: --namespace flag is required.${NC}"
    exit 1
fi

if [ -z "$TARGET_RESOURCE" ] || [ -z "$TARGET_NAME" ]; then
    echo "${RED}ERROR: --target-resource and --target-name flags are required.${NC}"
    exit 1
fi

# --- Helper Functions ---

# log_failure_details: Centralized function to log detailed debugging information on failure
log_failure_details() {
    start_time="${1}"

    echo "${RED}--- DETAILED FAILURE ANALYSIS ---${NC}"

    # Target EndpointSlice Status
    echo "${CYAN}  Status of target Private EndpointSlice in ${CLIENT_NAMESPACE}:${NC}"
    kubectl get endpointslice -n "$CLIENT_NAMESPACE" -l endpointslice.kubernetes.io/managed-by=endpointslice-controller.k8s.io -o yaml | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve EndpointSlice status${NC}"

    # Resolver Logs
    echo "${CYAN}  Logs from elasti-resolver:${NC}"
    kubectl logs -n elasti services/elasti-resolver-service --since-time="${start_time}" --all-pods=true | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve resolver logs${NC}"

    # Controller Logs
    echo "${CYAN}  Logs from elasti-controller:${NC}"
    kubectl logs -n elasti services/elasti-operator-controller-service --since-time="${start_time}" | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve controller logs${NC}"

    # Target Logs
    echo "${CYAN}  Logs from target (${TARGET_RESOURCE}/${TARGET_NAME}):${NC}"
    kubectl logs -n "$CLIENT_NAMESPACE" "${TARGET_RESOURCE}/${TARGET_NAME}" --since-time="${start_time}" | sed 's/^/    /' || echo "${YELLOW}    - Could not retrieve target pod logs${NC}"

    echo "${RED}-----------------------------------${NC}"
}

# --- Configuration Banner ---
printf "${CYAN}=== gRPC Traffic Test Configuration ===${NC}\n"
printf "  ${CYAN}Target Address:${NC}  %s\n" "$ADDR"
printf "  ${CYAN}Test Type:${NC}       %s\n" "$TEST_TYPE"
printf "  ${CYAN}Client Pod:${NC}      %s (in %s namespace)\n" "$CLIENT_POD_NAME" "$CLIENT_NAMESPACE"
printf "  ${CYAN}Target:${NC}          %s/%s (in %s namespace)\n" "${TARGET_RESOURCE}" "${TARGET_NAME}" "$CLIENT_NAMESPACE"
printf "  ${CYAN}Retries:${NC}         %s\n" "$MAX_RETRIES"
printf "  ${CYAN}Retry Sleep:${NC}     %ss\n" "${RETRY_SLEEP}"
printf "  ${CYAN}Max Failures:${NC}    %s\n" "$MAX_FAILURES"
printf "  ${CYAN}Timestamp:${NC}       %s\n" "$(date)"
printf "${CYAN}========================================${NC}\n"

# --- Check Client Pod ---
echo "Checking gRPC client pod status..."
if ! kubectl get pod "$CLIENT_POD_NAME" -n "$CLIENT_NAMESPACE" >/dev/null 2>&1; then
    echo "${RED}ERROR: gRPC client pod $CLIENT_POD_NAME not found in namespace $CLIENT_NAMESPACE${NC}"
    kubectl get pods -n "$CLIENT_NAMESPACE" || true
    exit 4
fi

POD_STATUS=$(kubectl get pod "$CLIENT_POD_NAME" -n "$CLIENT_NAMESPACE" -o jsonpath='{.status.phase}')
echo "${GREEN}gRPC client pod status: $POD_STATUS${NC}"

if [ "$POD_STATUS" != "Running" ]; then
    echo "${YELLOW}WARNING: gRPC client pod is not in Running state. Describing pod...${NC}"
    kubectl describe pod "$CLIENT_POD_NAME" -n "$CLIENT_NAMESPACE" || true
fi

echo ""
printf "${CYAN}--- Starting gRPC Traffic Test ---${NC}\n"

failure_count=0

for i in $(seq 1 $MAX_RETRIES); do
    printf "\n${CYAN}--- Request %d/%d ---${NC}\n" "$i" "$MAX_RETRIES"
    printf "  ${CYAN}Time:${NC} %s\n" "$(date)"

    printf "  ${CYAN}Executing: kubectl exec -n %s %s -- /grpc-client --addr=%s --test=%s${NC}\n" "$CLIENT_NAMESPACE" "$CLIENT_POD_NAME" "$ADDR" "$TEST_TYPE"
    start_time_rfc=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    start_time=$(date +%s)

    if kubectl exec -n "$CLIENT_NAMESPACE" "$CLIENT_POD_NAME" -- /grpc-client --addr="$ADDR" --test="$TEST_TYPE" 2>&1; then
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        printf "${GREEN}SUCCESS: Request %d completed successfully (%ds)${NC}\n" "$i" "$duration"
    else
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        printf "${RED}FAILED: Request %d failed (%ds)${NC}\n" "$i" "$duration"

        log_failure_details "${start_time_rfc}"
        failure_count=$((failure_count + 1))
    fi

    if [ $i -lt $MAX_RETRIES ]; then
        printf "  ${CYAN}Sleeping %ss before next request...${NC}\n" "${RETRY_SLEEP}"
        sleep $RETRY_SLEEP
    fi
    echo ""
done

printf "\n${CYAN}=== Test Summary ===${NC}\n"
printf "  ${CYAN}Failures:${NC}     %d / %d (max tolerated: %d)\n" "$failure_count" "$MAX_RETRIES" "$MAX_FAILURES"
printf "  ${CYAN}Target:${NC}       %s\n" "$ADDR"
printf "  ${CYAN}Completed at:${NC} %s\n" "$(date)"

if [ "$failure_count" -gt "$MAX_FAILURES" ]; then
    printf "${RED}Test FAILED with %d failed requests out of %d (max tolerated: %d).${NC}\n" "$failure_count" "$MAX_RETRIES" "$MAX_FAILURES"
    printf "${CYAN}====================${NC}\n"
    exit 1
else
    if [ "$failure_count" -gt 0 ]; then
        printf "${YELLOW}%d request(s) failed but within the tolerated limit of %d — test PASSED.${NC}\n" "$failure_count" "$MAX_FAILURES"
    else
        printf "${GREEN}All %d requests completed successfully.${NC}\n" "$MAX_RETRIES"
    fi
    printf "${CYAN}====================${NC}\n"
    exit 0
fi
