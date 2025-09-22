# Changelog

<!--
    Please refer to https://github.com/truefoundry/KubeElasti/blob/main/CONTRIBUTING.md#Changelog and follow the guidelines before adding a new entry.
-->

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
* Add Docs for KubeElasti at https://kubeelasti.dev by @ramantehlan in [#142](https://github.com/truefoundry/KubeElasti/pull/142)
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

All the unreleased changes are listed under `Unreleased` section.

## Unreleased

<!--
    Add new changes here and sort them alphabetically.
Example -
- **General**: Add support for statefulset as a scale target reference ([#10](https://github.com/truefoundry/elasti/pull/10))
-->
