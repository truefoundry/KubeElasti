---
title: "KubeElasti Load Testing - Performance Testing with k6"
description: "Perform load testing on KubeElasti with k6. Learn how to test performance, scaling behavior, and system limits under load."
keywords:
  - KubeElasti load testing
  - k6 performance testing
  - Kubernetes load testing
  - performance testing
  - scale testing
  - stress testing
---

# Load testing

## 1. Update k6 tests

   Update `./test/load.js` to set your target URL and adjust any other configuration values.

## 2. Run load.js

   Run the following command to run the test.

   ```bash
   chmod +x ./test/generate_load.sh
   cd ./test
   ./generate_load.sh
   ```
