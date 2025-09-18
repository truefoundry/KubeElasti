#!/bin/sh

# Enhanced traffic testing script with comprehensive debugging
set -u

# Validate input
if [ -z "$1" ]; then
    echo "ERROR: No URL provided. Usage: $0 <url>"
    exit 3
fi

URL="$1"
POD_NAME="curl-target-gw"
NAMESPACE="default"
MAX_RETRIES=5
TIMEOUT=30

echo "=== Traffic Test Configuration ==="
echo "Target URL: $URL"
echo "Pod: $POD_NAME"
echo "Namespace: $NAMESPACE"
echo "Retries: $MAX_RETRIES"
echo "Timeout: ${TIMEOUT}s"
echo "Timestamp: $(date)"
echo "=================================="

# Check if pod exists and is ready
echo "Checking pod status..."
if ! kubectl get pod "$POD_NAME" -n "$NAMESPACE" >/dev/null 2>&1; then
    echo "ERROR: Pod $POD_NAME not found in namespace $NAMESPACE"
    kubectl get pods -n "$NAMESPACE" || true
    exit 4
fi

POD_STATUS=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
echo "Pod status: $POD_STATUS"

if [ "$POD_STATUS" != "Running" ]; then
    echo "WARNING: Pod is not in Running state"
    kubectl describe pod "$POD_NAME" -n "$NAMESPACE" || true
fi

echo ""
echo "Starting traffic test..."

for i in $(seq 1 $MAX_RETRIES); do
    echo "--- Request $i/$MAX_RETRIES ---"
    echo "Time: $(date)"
    
    # Execute curl with detailed output
    echo "Executing: kubectl exec -n $NAMESPACE $POD_NAME -- curl --max-time $TIMEOUT -s -o /dev/null -w \"%{http_code}\" \"$URL\""
    
    start_time=$(date +%s)
    code=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- curl \
        --max-time "$TIMEOUT" \
        -s \
        -o /dev/null \
        -w "%{http_code}" \
        "$URL" 2>&1)
    result=$?
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    echo "Curl exit code: $result"
    echo "HTTP status code: $code"
    echo "Request duration: ${duration}s"
    
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
        
        # Additional debugging
        echo "Pod logs (last 10 lines):"
        kubectl logs "$POD_NAME" -n "$NAMESPACE" --tail=10 || echo "  - Could not retrieve logs"
        
        echo "Pod network info:"
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- ip addr show || echo "  - Could not get network info"
        
        exit 1
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
        
        # Try to get more details with verbose curl
        echo "Attempting verbose request for debugging..."
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- curl \
            --max-time 10 \
            -v \
            "$URL" 2>&1 | head -20 || echo "  - Verbose request failed"
        
        exit 2
    fi
    
    echo "SUCCESS: Request $i completed successfully (HTTP $code in ${duration}s)"
    
    if [ $i -lt $MAX_RETRIES ]; then
        echo "Waiting 1 second before next request..."
        sleep 1
    fi
    echo ""
done

echo "=== Test Summary ==="
echo "All $MAX_RETRIES requests completed successfully"
echo "Target: $URL"
echo "Completed at: $(date)"
echo "===================="
