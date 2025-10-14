---
title: "KubeElasti Architecture FAQ - Design Decisions and Technical Questions"
description: "Common questions about KubeElasti's architecture decisions, component separation, and design choices for Kubernetes serverless scaling."
keywords:
  - KubeElasti FAQ
  - architecture questions
  - design decisions
  - resolver operator separation
  - Kubernetes architecture
  - technical FAQ
---

# Architecture FAQ

This FAQ addresses common questions about KubeElasti's architecture decisions and design choices.

### Q: Why does KubeElasti use separate images for the Resolver and Operator?

**A:** KubeElasti uses separate images for architectural and operational reasons:

**Resolver Requirements:**
- Designed to be **horizontally scalable** with multiple instances
- Handles incoming traffic and needs high availability
- Supports HPA (Horizontal Pod Autoscaler) for dynamic scaling
- Focused on one responsibility: receive, hold, and process requests

**Operator Requirements:**
- Runs as a **singleton service** with only one active instance
- Acts as an orchestrator handling state updates and triggers
- Built using kubebuilder with standard controller patterns
- Handles signals and triggers, not data processing

### Q: Why can't the Operator run with multiple replicas?

**A:** The Operator must run as a singleton because:
- This follows standard Kubernetes controller patterns
- Only one controller can be active at a time for the ElastiService resource

### Q: How does the Resolver achieve high availability?

**A:** The Resolver is designed for horizontal scaling:
- Multiple instances can run simultaneously
- HPA automatically scales based on load
- Each instance can independently handle incoming requests
- Load balancing distributes traffic across instances

### Q: Why does KubeElasti use multiple go.mod files with go.work?

**A:** This wasn't originally planned but evolved organically:
- Initially had separate Resolver and Operator modules
- The `pkg` module was introduced later for shared components
- Alternative approaches (single go.mod with `./cmd/` structure) were considered
- The current structure provides clear separation of concerns

