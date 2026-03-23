#!/bin/sh
# Verify that custom request headers are forwarded by the resolver to the target service.
#
# Scenario:
#   - Deployment is scaled to 0 (proxy mode).
#   - A request with custom headers is sent to the httpbin /headers endpoint.
#   - The resolver holds the request while scaling the deployment back up, then forwards it.
#   - The /headers endpoint echoes all received headers as JSON.
#   - This script asserts the custom header is present in that response.

set -u

if [ -z "$1" ]; then
    echo "ERROR: No URL provided."
    echo "Usage: $0 <url> --namespace <ns> --target-resource <resource> --target-name <name>"
    exit 3
fi

URL="$1"
CURL_POD_NAME="curl-target-gw"
CURL_NAMESPACE="default"
TARGET_NAMESPACE=""
TARGET_RESOURCE=""
TARGET_NAME=""
# Long timeout: the resolver must hold the connection while the deployment scales from 0.
TIMEOUT=300

shift
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
    echo "ERROR: --namespace is required."
    exit 5
fi
if [ -z "$TARGET_RESOURCE" ] || [ -z "$TARGET_NAME" ]; then
    echo "ERROR: --target-resource and --target-name are required."
    exit 5
fi

HEADER_NAME="X-Elasti-Test"
HEADER_VALUE="elasti-custom-header-check"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo "${CYAN}=== Custom Header Forwarding Test ===${NC}"
echo "  URL:            $URL"
echo "  Custom header:  $HEADER_NAME: $HEADER_VALUE"
echo "  Curl pod:       $CURL_POD_NAME (namespace: $CURL_NAMESPACE)"
echo "  Request timeout: ${TIMEOUT}s (covers resolver hold + deployment scale-up)"
echo "${CYAN}=====================================${NC}"

response=$(kubectl exec -n "$CURL_NAMESPACE" "$CURL_POD_NAME" -- curl \
    --max-time "$TIMEOUT" \
    -s \
    -H "$HEADER_NAME: $HEADER_VALUE" \
    "$URL" 2>&1)

result=$?
echo "  curl exit code: $result"
echo "  Response body:  $response"
echo ""

if [ "$result" != "0" ]; then
    echo "${RED}FAILED: curl exited with code $result (timeout or connection error).${NC}"
    echo ""
    echo "${CYAN}Resolver logs:${NC}"
    kubectl logs -n elasti services/elasti-resolver-service --all-pods=true --tail=60 | sed 's/^/  /' || true
    echo ""
    echo "${CYAN}Target logs (${TARGET_RESOURCE}/${TARGET_NAME}):${NC}"
    kubectl logs -n "$TARGET_NAMESPACE" "${TARGET_RESOURCE}/${TARGET_NAME}" --tail=30 | sed 's/^/  /' || true
    exit 1
fi

if ! echo "$response" | grep -q "$HEADER_VALUE"; then
    echo "${RED}FAILED: Custom header '$HEADER_NAME: $HEADER_VALUE' not found in response body.${NC}"
    echo "The resolver did not forward the header to the target deployment."
    echo ""
    echo "${CYAN}Resolver logs:${NC}"
    kubectl logs -n elasti services/elasti-resolver-service --all-pods=true --tail=60 | sed 's/^/  /' || true
    exit 1
fi

echo "${GREEN}PASSED: '$HEADER_NAME: $HEADER_VALUE' is present in the /headers response.${NC}"
echo "Custom headers are correctly forwarded through the resolver to the scaled-up deployment."
echo "${CYAN}=====================================${NC}"
