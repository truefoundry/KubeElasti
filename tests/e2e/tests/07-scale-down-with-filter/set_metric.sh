#!/bin/sh

COMMAND="echo '${1} ${2}' | curl --data-binary @- http://prometheus-pushgateway.monitoring.svc.cluster.local:9091/metrics/job/some_job"

# Namespace is setup in 00-setup.yaml
kubectl exec -n 7 curl-target-gw -- sh -c "${COMMAND}"
