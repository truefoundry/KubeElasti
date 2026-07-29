---
name: release
description: Cut a KubeElasti release (stable, rc, or beta). Use when asked to "create a release", "cut vX.Y.Z", "release X.Y.Z-rc1", or bump the chart/app version and changelog for a new tag. Prepares the version files, changelog, release PR, and the git tag that triggers the signed-release workflow.
---

# Release a version of KubeElasti

Prepare and cut a release. The canonical process lives in `RELEASE.md` at the repo
root - read it first; this skill is the operational checklist that automates the
preparation steps.

## Version scheme

KubeElasti uses SemVer with a leading `v` on the git tag only.

| Kind   | Git tag / GitHub release | `version` & `appVersion` in files |
| ------ | ------------------------ | --------------------------------- |
| Stable | `v0.1.31`                | `0.1.31`                          |
| RC     | `v0.1.31-rc1`            | `0.1.31-rc1`                      |
| Beta   | `v0.1.31-beta`           | `0.1.31-beta`                     |

The version written into chart files never carries the `v`. The git tag always does.

## What triggers the actual release

Pushing a tag matching `v*` triggers `.github/workflows/release.yaml`, which builds and
cosign-signs the operator + resolver images, packages and signs the Helm chart, generates
SBOMs, pushes the chart to `oci://ghcr.io/kubeelasti/charts`, and attaches everything to the
GitHub release. You only prepare files and create the tag/release - CI does the rest.

## Preparation steps

Do these on a release branch (e.g. `sd/release-vX.Y.Z` or `release-vX.Y.Z`), never directly on `main`.

1. **Determine the changes since the last release.** Find the previous tag and list what
   landed on `main` since:
   ```bash
   git fetch --tags origin
   PREV=$(git tag | sort -V | tail -1)   # or the last stable tag explicitly
   git log --oneline "$PREV"..origin/main
   ```
   For each merged PR, get title + author for the changelog:
   ```bash
   gh pr view <N> --repo KubeElasti/KubeElasti --json title,author,url
   ```

2. **Bump `charts/elasti/Chart.yaml`** - set both `version` and `appVersion` to the
   no-`v` version (quote `appVersion`):
   ```yaml
   version: 0.1.31-rc1
   appVersion: "0.1.31-rc1"
   ```

3. **Bump `charts/elasti/values.yaml`** - update BOTH image tags (operator ~line 93,
   resolver ~line 219) to the same no-`v` version:
   ```yaml
   tag: 0.1.31-rc1
   ```
   Confirm exactly two occurrences: `grep -n "tag:" charts/elasti/values.yaml`.

4. **Update `CHANGELOG.md`** - add a new section directly under `## Unreleased`, above the
   previous release. Use the date `YYYY-MM-DD` (today). Group entries under the standard
   headings and keep only those with content: `New`, `Experimental`, `Improvements`,
   `Fixes`, `Breaking Changes`, `Other`, `New Contributors`. Entry format:
   ```markdown
   ## v0.1.31-rc1 (2026-07-27)

   ### Fixes

   * <what changed, plain language> by `@author` in [#303](https://github.com/KubeElasti/KubeElasti/pull/303)
   ```
   For an rc, list only the new changes since the last release. The full comparison against
   the last stable goes in the GitHub release notes, not the changelog.

## Do NOT hand-edit these

CI regenerates them; editing by hand causes conflicts:

- `charts/elasti/README.md` (readme-generator-for-helm)
- `docs/index.yaml` (helm index)

## Commit, PR, and tag

Every commit MUST be signed off (`-s`) and GPG/DCO clean. Do not add Claude as co-author or
put any agent signature in commits or the PR.

```bash
git add charts/elasti/Chart.yaml charts/elasti/values.yaml CHANGELOG.md
git commit -s -m "release: v0.1.31-rc1"
git push -u origin <release-branch>
gh pr create --repo KubeElasti/KubeElasti --base main \
  --title "release: v0.1.31-rc1" --body "<summary of changes>"
```

Merge the PR to `main` first. Then create the GitHub release (which creates and pushes the
tag, triggering the workflow):

```bash
gh release create v0.1.31-rc1 --repo KubeElasti/KubeElasti \
  --title "KubeElasti v0.1.31-rc1" --prerelease \
  --notes "<release notes>"
```

Use `--prerelease` for rc/beta tags. Omit it for stable. Title format: `KubeElasti vX.Y.Z`.

## Before finishing

- Verify chart lints and templates render: `helm lint charts/elasti` and
  `helm template charts/elasti >/dev/null`.
- Confirm the three prepared files are the only source changes in the release commit.
- Ask before pushing tags or creating the GitHub release - those are irreversible and
  outward-facing. Preparing the branch/PR is fine to do proactively.
