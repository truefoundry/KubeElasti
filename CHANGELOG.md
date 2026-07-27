# Changelog

<!--
    Please refer to https://github.com/KubeElasti/KubeElasti/blob/main/CONTRIBUTING.md#Changelog and follow the guidelines before adding a new entry.
-->

All the unreleased changes are listed under `Unreleased` section. Add your changes here, they will be moved to the next release.

## Unreleased

## v0.1.30 (2026-07-23)

### Improvements

* feat: add opt-in namespace-scoped mode. Set `global.allowedNamespaces` (Helm) or the `WATCH_NAMESPACES` operator env var to confine KubeElasti to an explicit list of namespaces using namespace-scoped Role/RoleBinding instead of cluster-wide RBAC. Cluster-scoped stays the default and is unchanged. Adds the `global.installCRD` toggle so an admin can apply the CRD once and let namespace-confined tenants install without cluster-wide permissions by `@ramantehlan` in [#317](https://github.com/KubeElasti/KubeElasti/pull/317)

## v0.1.30-rc1 (2026-07-22)

### Breaking Changes

* Images and the Helm chart move to GHCR (`ghcr.io/kubeelasti`) instead of JFrog. Update image pull and Helm repository references accordingly; this may be a breaking change for users still pointing at the old JFrog URLs. by `@ramantehlan` in [#308](https://github.com/KubeElasti/KubeElasti/pull/308)

### Fixes

* fix: add grace period for CNI to converge before disabling resolver proxy during scale-from-zero events by `@chaserhkj` in [#293](https://github.com/KubeElasti/KubeElasti/pull/293). Fixes [#280](https://github.com/KubeElasti/KubeElasti/issues/280)
* fix: bound generated private service and EndpointSlice names to Kubernetes length limits by `@ramantehlan` in [#295](https://github.com/KubeElasti/KubeElasti/pull/295). Fixes [#294](https://github.com/KubeElasti/KubeElasti/issues/294)

### Improvements

* helm: support nodeSelector, tolerations, and affinity on elastiController and elastiResolver by `@mcastellini` in [#292](https://github.com/KubeElasti/KubeElasti/pull/292)
* feat: update Go modules to v1.26.5 and clear Grype-reported vulnerabilities by `@ramantehlan` in [#300](https://github.com/KubeElasti/KubeElasti/pull/300)
* feat: generate and publish SBOMs for images and chart on release by `@ramantehlan` in [#304](https://github.com/KubeElasti/KubeElasti/pull/304)
* feat: add signed releases with Cosign by `@ramantehlan` in [#311](https://github.com/KubeElasti/KubeElasti/pull/311)
* feat: OpenSSF Phase 1 hardening, including Gitleaks and read-only workflow defaults by `@ramantehlan` in [#301](https://github.com/KubeElasti/KubeElasti/pull/301) and [#306](https://github.com/KubeElasti/KubeElasti/pull/306)
* feat: update OpenSSF Scorecard configuration by `@ramantehlan` in [#305](https://github.com/KubeElasti/KubeElasti/pull/305)

### Other

* feat: update KubeElasti logo and maintainers by `@ramantehlan` in [#310](https://github.com/KubeElasti/KubeElasti/pull/310)
* docs: replace GitHub stars callout with project contributors by `@ramantehlan` in [#307](https://github.com/KubeElasti/KubeElasti/pull/307)
* chore(deps): bump google.golang.org/grpc in e2e gRPC test service by `@dependabot` in [#312](https://github.com/KubeElasti/KubeElasti/pull/312)

### New Contributors

* `@mcastellini` made their first contribution in [#292](https://github.com/KubeElasti/KubeElasti/pull/292)

## v0.1.25 (2026-06-05)

### Fixes

* fix: retry the resolver backoff dialer on "connection refused" (not only on timeouts), avoiding a 502 on the first request after scale-from-zero when the target pod or kube-proxy route is not ready yet by `@sandrotaje` in [#286](https://github.com/KubeElasti/KubeElasti/pull/286)

### Other

* feat: add E2E gRPC integration tests with H2C support by `@vorflux` in [#281](https://github.com/KubeElasti/KubeElasti/pull/281)
* adopter: add buildo by `@ramantehlan` in [#288](https://github.com/KubeElasti/KubeElasti/pull/288)
* docs: add Envoy Gateway routing support and integration docs by `@ramantehlan` in [#289](https://github.com/KubeElasti/KubeElasti/pull/289)

## v0.1.24 (2026-05-29)

### Fixes

* fix: unpause KEDA ScaledObject when ElastiService is deleted by @shubhamrai1993 in [#283](https://github.com/KubeElasti/KubeElasti/pull/283)

### Other
* docs: move to zensical for docs and landing page. by @ramantehlan in [#279](https://github.com/KubeElasti/KubeElasti/pull/279)


## v0.1.23 (2026-04-15)

### Breaking Changes

* add probeResponse to ElastiService, without scale up of the target service by `@ramantehlan`in 

### Other

* fix: Prevent clients from reusing connections for proxy mode, fixing connection-reusing clients behavior (e.g. git) at scale 0, by `@chaserhkj` in [`269`](https://github.com/KubeElasti/KubeElasti/pull/269)
* docs: add project governance document and link it from contributor-facing docs
* lint fixes: update lint version to v2 in github workflow & fix existing issues by `@ramantehlan` in [#265](https://github.com/KubeElasti/KubeElasti/pull/265)
* Update github issue and PR forms, updated raw images and PR/Issue greeting message by `@ramantehlan` in [#264](https://github.com/KubeElasti/KubeElasti/pull/264)


## v0.1.22 (2026-04-01)

### Breaking Changes

* feat: support scheduled activity for ElastiService objects by `@artazar` in [`#243`](https://github.com/truefoundry/KubeElasti/pull/243)
* moving all truefoundry/kubeelasti ref to kubeelasti/kubeelasti by @shubhamrai1993 in [`#246`](https://github.com/KubeElasti/KubeElasti/pull/246)
* add readiness and liveness probe to resolver deployment by @maanas-23 in [`#251`](https://github.com/KubeElasti/KubeElasti/pull/251)
* fix: clear owner references for private service, allowing referencing services created by other controllers by @chaserhkj in [`256`](https://github.com/KubeElasti/KubeElasti/pull/256)

### Improvements

* Helm Chart Changes by @ramantehlan in [`237`](https://github.com/KubeElasti/KubeElasti/pull/237)
* fix: variable name correction for pollingInterval value by @artazar in [`241`](https://github.com/KubeElasti/KubeElasti/pull/241)

### Other

* We are officially part of CNCF! by @ramantehlan in [`244`](https://github.com/KubeElasti/KubeElasti/pull/244)
* Add Raman Tehlan to MAINTAINERS list by @shubhamrai1993 in [`245`](https://github.com/KubeElasti/KubeElasti/pull/245)
* docs: replace Discord links with CNCF Slack by @shubhamrai1993 in [`254`](https://github.com/KubeElasti/KubeElasti/pull/254)

## New Contributors

* @artazar made their first contribution in [`241`](https://github.com/KubeElasti/KubeElasti/pull/241)
* @chaserhkj made their first contribution in [`256`](https://github.com/KubeElasti/KubeElasti/pull/256)


## v0.1.21 (2026-01-16)

### Improvements

* fix: stage all changes including deletions in helm-main sync workflow by @sachincool in [#225](https://github.com/truefoundry/KubeElasti/pull/225)
* feat: support HTTP/2 cleartext (H2C) in resolver by @quentinplessis in [#219](https://github.com/truefoundry/KubeElasti/pull/219)
* feat: support passing headers in prometheus requests (Grafana Mimir Support) by @quentinplessis in [#214](https://github.com/truefoundry/KubeElasti/pull/214)
* Workflow Update: Add helm chart to release | Update action conditions.  by @ramantehlan in [#229](https://github.com/truefoundry/KubeElasti/pull/229)

## New Contributors
* @quentinplessis made their first contribution in [#219](https://github.com/truefoundry/KubeElasti/pull/219)

## v0.1.20 (2026-01-08)

### Improvements

* Helm: Standardize labels and annotations across all resources in the chart by @bluebox in [#216](https://github.com/truefoundry/KubeElasti/pull/216)

## 0.1.19

### Improvements

* Adding support for passing custom registry and imagePullSecrets by @dunefro in [#210](https://github.com/truefoundry/KubeElasti/pull/210)
* Adding readme generator by @dunefro in [#211](https://github.com/truefoundry/KubeElasti/pull/211)

### Fixes

* fix incorrect error return case in unhealthy scaler check by @maanas-23 in [#209](https://github.com/truefoundry/KubeElasti/pull/209)

## 0.1.18

### Improvements

* SEO optimizations & docs update by @ramantehlan in [#200](https://github.com/truefoundry/KubeElasti/pull/200)
* Add adopters message on top by @ramantehlan in [#202](https://github.com/truefoundry/KubeElasti/pull/202)

### Fixes

* Update docs by @ramantehlan in [#201](https://github.com/truefoundry/KubeElasti/pull/201)
* Docs fix by @ramantehlan in [#203](https://github.com/truefoundry/KubeElasti/pull/203)
* Safely check for prefix in CRD name instead of looking at slice length by @shubhamrai1993 in [#204](https://github.com/truefoundry/KubeElasti/pull/204)

### Other

* Release 0.1.18-rc.1 by @shubhamrai1993 in [#205](https://github.com/truefoundry/KubeElasti/pull/205)

## 0.1.17

### Improvements

* Add support for StatefulSet as a scale target reference by @ramantehlan in [#188](https://github.com/truefoundry/KubeElasti/pull/188)
* helm: forward values to elasti through env variables by @rethil in [#178](https://github.com/truefoundry/KubeElasti/pull/178)

### Fixes

* Fix: 02 and 09 test for on traffic scale by @ramantehlan in [#194](https://github.com/truefoundry/KubeElasti/pull/194)
* e2e: fix 02 test failure by @rethil in [#195](https://github.com/truefoundry/KubeElasti/pull/195)

### Other

* added adopters.md to document adopters by @shubhamrai1993 in [#197](https://github.com/truefoundry/KubeElasti/pull/197)
* add link to star github repo by @shubhamrai1993 in [#198](https://github.com/truefoundry/KubeElasti/pull/198)
* chore: upgrade go modules & 3pp used for building by @rethil in [#186](https://github.com/truefoundry/KubeElasti/pull/186)

## 0.1.16 (2025-09-22)

### Fixes

* fix: first response content after scaling up is truncated by @rethil in [#163](https://github.com/truefoundry/KubeElasti/pull/163)
* copying over private service ports from public service directly by @shubhamrai1993 in [#190](https://github.com/truefoundry/KubeElasti/pull/190)
* fix 02 e2e test, which was because of incorrect readiness check of endpointslice.  by @ramantehlan in [#189](https://github.com/truefoundry/KubeElasti/pull/189)

### Improvements

* prometheus scaler: make healthcheck customizable by @rethil in [#153](https://github.com/truefoundry/KubeElasti/pull/153)
* resolver: use distroless/static as base image by @rethil in [#181](https://github.com/truefoundry/KubeElasti/pull/181)
* resolver: migrate to EndpointSlice API by @rethil in [#167](https://github.com/truefoundry/KubeElasti/pull/167)
* security: cleanup roles in helm chart by @rethil in [#179](https://github.com/truefoundry/KubeElasti/pull/179)

### Other

* Docs: Add demo video - Contributors section - Discord link by @ramantehlan in [#172](https://github.com/truefoundry/KubeElasti/pull/172)
* Add announcement and FAQ to the docs by @ramantehlan in [#177](https://github.com/truefoundry/KubeElasti/pull/177)
* fix e2e workflow and tests by @ramantehlan in [#184](https://github.com/truefoundry/KubeElasti/pull/184)

## 0.1.15

* Add validation for CRD fields for elasti service by @ramantehlan in [#122](https://github.com/truefoundry/elasti/pull/122)
* Forward source host to target by @ramantehlan in [#123](https://github.com/truefoundry/elasti/pull/123)

## 0.1.15-beta (2025-07-28)

### Fixes

* Forward source host to target by @ramantehlan in [#159](https://github.com/truefoundry/elasti/pull/159)

### Improvements

* Supporting second level cooldown period for prometheus uptime check by @shubhamrai1993 in [#125](https://github.com/truefoundry/KubeElasti/pull/125)
* Add validation for CRD fields for elasti service by @ramantehlan in [#138](https://github.com/truefoundry/elasti/pull/138)

### Other

* Add E2E tests via Kuttl by @ramantehlan in [#123](https://github.com/truefoundry/KubeElasti/pull/123)
* Add Docs for KubeElasti at <https://kubeelasti.dev> by @ramantehlan in [#142](https://github.com/truefoundry/KubeElasti/pull/142)
* Bump golang.org/x/oauth2 from 0.21.0 to 0.27.0 in /pkg in [#156](https://github.com/truefoundry/KubeElasti/pull/156)
* Bump golang.org/x/oauth2 from 0.21.0 to 0.27.0 in /operator in [#155](https://github.com/truefoundry/KubeElasti/pull/155)
* Bump golang.org/x/oauth2 from 0.21.0 to 0.27.0 in /resolver in [#151](https://github.com/truefoundry/KubeElasti/pull/151)
* Security Fix: Bump golang.org/x/net from 0.33.0 to 0.38.0 in /pkg in [#143](https://github.com/truefoundry/KubeElasti/pull/143)

### New Contributors

* @rethil made their first contribution in [#154](https://github.com/truefoundry/KubeElasti/pull/154)

## 0.1.14

* update workflow to update grype config by @DeeAjayi in [#113](https://github.com/truefoundry/KubeElasti/pull/113)
* Add support for namespace scoped elasti controller and fixes for cooldown period tracking by @shubhamrai1993 in [#115](https://github.com/truefoundry/KubeElasti/pull/115)
* Increasing elasti timeout to 10 minutes by @shubhamrai1993 in [#116](https://github.com/truefoundry/KubeElasti/pull/116)
* corrected the target name being passed by @shubhamrai1993 in [#118](https://github.com/truefoundry/KubeElasti/pull/118)
* using event recorder for emitting events by @shubhamrai1993 in [#117](https://github.com/truefoundry/KubeElasti/pull/117)
* dont scale up replicas if the current replicas are greater by @shubhamrai1993 in [#119](https://github.com/truefoundry/KubeElasti/pull/119)
* fix: port 5000 is used by the systems, using it might be hurdle for u… by @ramantehlan in [#120](https://github.com/truefoundry/KubeElasti/pull/120)
* dont scale down new service and handle missing prom data by @shubhamrai1993 in [#121](https://github.com/truefoundry/KubeElasti/pull/121)
