---
title: "KubeElasti vs KEDA"
description: "Detailed technical comparison between KubeElasti and KEDA HTTP Add-on. Architecture, scaling mechanisms, resource management, and operational differences."
keywords:
  - KubeElasti vs KEDA
  - KEDA HTTP Add-on comparison
  - Kubernetes auto-scaling
  - scale to zero architecture
  - HTTP proxy scaling
  - event-driven autoscaling
tags:
  - comparison
  - keda
  - architecture
  - scaling
  - technical
---

# KubeElasti vs KEDA Technical Comparison

This document provides a comprehensive technical comparison between KubeElasti and KEDA (Kubernetes Event-Driven Autoscaling), specifically focusing on the KEDA HTTP Add-on for HTTP-based scaling scenarios.

## Architecture Overview

### KubeElasti Architecture

KubeElasti implements a dual-mode architecture with intelligent traffic management:

- **Controller**: Manages ElastiService CRDs and orchestrates scaling decisions
- **Resolver**: Acts as HTTP proxy/load balancer with dynamic routing
- **Dual Mode Operation**: Switches between proxy mode (scale-from-zero) and serve mode (direct routing)
- **Prometheus Integration**: Built-in metrics collection and query-based scaling triggers

### KEDA Architecture

KEDA provides event-driven autoscaling through external scalers:

- **KEDA Operator**: Manages ScaledObject CRDs and HPA integration
- **KEDA HTTP Add-on**: Separate component for HTTP-based scaling
- **External Scaler Pattern**: Delegates scaling decisions to external metrics
- **HPA Integration**: Leverages Kubernetes HPA for actual scaling operations

## Scaling Mechanisms

| Feature | KubeElasti | KEDA HTTP Add-on |
|---------|------------|------------------|
| **Scaling Trigger** | Prometheus queries with custom metrics | HTTP request queue depth |
| **Scale-to-Zero** | Native support with proxy mode | Supported via interceptor |
| **Scale-from-Zero** | Automatic via resolver proxy | Requires HTTP interceptor |
| **Scaling Algorithm** | Custom controller with configurable thresholds | HPA-based with external metrics |
| **Cold Start Handling** | Intelligent proxy buffering | Request queuing in interceptor |
| **Scaling Speed** | Configurable intervals (default: 30s) | HPA-controlled (default: 15s) |

## Traffic Management

### KubeElasti Traffic Flow

```
Client Request → Resolver (Proxy/Serve Mode) → Target Service
                     ↓
              Prometheus Metrics → Controller → Scaling Decision
```

**Proxy Mode**: Resolver acts as reverse proxy, buffers requests during scaling
**Serve Mode**: Direct routing to target service for optimal performance

### KEDA HTTP Add-on Traffic Flow

```
Client Request → HTTP Interceptor → Target Service
                     ↓
              Queue Metrics → KEDA Operator → HPA → Scaling Decision
```

**Always-On Proxy**: HTTP interceptor remains in request path
**Queue-Based**: Scaling decisions based on request queue depth

## Resource Requirements

### KubeElasti

```yaml
# Controller Resource Usage
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

# Resolver Resource Usage
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 256Mi
```

### KEDA HTTP Add-on

```yaml
# KEDA Operator Resource Usage
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 1000m
    memory: 1000Mi

# HTTP Interceptor Resource Usage
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 1000m
    memory: 1000Mi
```

## Configuration Complexity

### KubeElasti Configuration

```yaml
apiVersion: elasti.truefoundry.com/v1alpha1
kind: ElastiService
metadata:
  name: example-service
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: target-deployment
  triggers:
    - type: prometheus
      metadata:
        serverAddress: http://prometheus:9090
        query: 'rate(http_requests_total[1m])'
        threshold: '0.1'
  minReplicas: 0
  maxReplicas: 10
```

### KEDA HTTP Add-on Configuration

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: example-scaledobject
spec:
  scaleTargetRef:
    name: target-deployment
  minReplicaCount: 0
  maxReplicaCount: 10
  triggers:
  - type: external
    metadata:
      scalerAddress: keda-http-add-on-external-scaler:9090
      host: example.com
      targetPendingRequests: '10'
---
apiVersion: http.keda.sh/v1alpha1
kind: HTTPScaledObject
metadata:
  name: example-http-scaledobject
spec:
  host: example.com
  targetPendingRequests: 10
  scaleTargetRef:
    deployment: target-deployment
    service: target-service
    port: 8080
```

## Performance Characteristics

### Latency Impact

| Scenario | KubeElasti | KEDA HTTP Add-on |
|----------|------------|------------------|
| **Serve Mode** | ~1ms overhead | ~2-5ms overhead |
| **Proxy Mode** | ~5-10ms overhead | ~2-5ms overhead |
| **Cold Start** | 200-500ms (with buffering) | 300-800ms (queue processing) |
| **Scaling Decision** | 30s default interval | 15s HPA interval |

### Throughput Characteristics

- **KubeElasti**: Higher throughput in serve mode due to direct routing
- **KEDA**: Consistent throughput with always-on proxy pattern

## Monitoring and Observability

### KubeElasti Metrics

```prometheus
# Built-in Prometheus metrics
elasti_service_mode{service="example", mode="proxy|serve"}
elasti_service_replicas{service="example"}
elasti_service_requests_total{service="example"}
elasti_service_scaling_duration_seconds{service="example"}
```

### KEDA Metrics

```prometheus
# KEDA operator metrics
keda_scaler_active{scaler="external-scaler"}
keda_scaled_object_paused{scaledObject="example"}
keda_scaler_metrics_value{scaler="external-scaler"}

# HTTP Add-on metrics
keda_http_requests_pending{host="example.com"}
keda_http_requests_total{host="example.com"}
```

## Operational Considerations

### KubeElasti

**Advantages**:
- Intelligent mode switching reduces proxy overhead
- Built-in Prometheus integration
- Simpler architecture with fewer components
- Custom scaling algorithms beyond HTTP metrics

**Limitations**:
- Requires Prometheus for advanced scaling
- Custom CRD learning curve
- Less mature ecosystem

### KEDA HTTP Add-on

**Advantages**:
- Mature ecosystem with extensive scaler support
- Standard HPA integration
- Active community and enterprise support
- Multiple trigger types beyond HTTP

**Limitations**:
- Always-on proxy overhead
- More complex multi-component architecture
- Limited customization of scaling algorithms
- Additional resource overhead

## Use Case Recommendations

### Choose KubeElasti When

- Performance optimization is critical (serve mode benefits)
- Custom scaling logic based on business metrics
- Prometheus-centric monitoring stack
- Simplified operational model preferred

### Choose KEDA When

- Multi-trigger scaling requirements (HTTP + queue + database)
- Enterprise support and ecosystem maturity needed
- Standard HPA integration preferred
- Existing KEDA infrastructure in place

## Migration Considerations

### From KEDA to KubeElasti

1. Replace ScaledObject with ElastiService CRD
2. Convert HTTP interceptor configuration to resolver settings
3. Migrate external scaler metrics to Prometheus queries
4. Update monitoring dashboards for new metrics

### From KubeElasti to KEDA

1. Deploy KEDA operator and HTTP Add-on
2. Create HTTPScaledObject and ScaledObject resources
3. Configure HTTP interceptor routing
4. Migrate Prometheus-based triggers to external scaler pattern
