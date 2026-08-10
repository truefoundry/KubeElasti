# KubeElasti Governance

This document describes who runs KubeElasti, how decisions are made, and how you can take on a
role in the project.

It covers the whole KubeElasti GitHub organization: code, docs, releases, issues, and the
community spaces the project runs.

## Values

- **Open**: discussion and decisions happen in public whenever possible.
- **Fair**: people are judged on their work, not their employer or background.
- **Respectful**: all interaction follows the [Code of Conduct](./CODE_OF_CONDUCT.md).
- **Sustainable**: leadership is shared and written down, so the project outlives any one person
  or company.

## Vendor Neutrality

KubeElasti is a vendor-neutral CNCF project. Maintainers set project direction, roadmap, and
technical decisions in the interest of the whole community, not any single employer or vendor.
Affiliation does not grant preferential treatment in reviews, roadmap decisions, release timing,
or applications for reviewer or maintainer roles. See the CNCF guidance on
[vendor neutrality](https://contribute.cncf.io/maintainers/community/vendor-neutrality/).

## Roles

Anyone who opens an issue, sends a pull request, writes docs, answers questions, or helps in
Slack is a contributor. That needs no application and grants no special permissions. Beyond
that, KubeElasti has two roles. Both are organization-wide: there are no per-repository roles.

### Organization Reviewer

A reviewer is a trusted contributor who helps keep the queue moving. The current reviewers are
listed in [REVIEWERS](./REVIEWERS).

**Can:**

- Review pull requests and leave binding review feedback.
- Triage, label, and close issues.
- Be requested for review on any repository in the organization.

**Cannot:**

- Merge pull requests.
- Vote on maintainer or governance decisions.

**Expected to:**

- Give timely, constructive, technically sound reviews.
- Help contributors meet the project's quality bar.
- Escalate design, release, or governance questions to maintainers.

### Organization Maintainer

Maintainers are the decision-making body and are accountable for the health of the project.
The current roster is in [MAINTAINERS](./MAINTAINERS), which is the single source of truth.

**Can:**

- Approve and merge pull requests.
- Cut and publish releases.
- Vote on roles, governance changes, and project direction.
- Administer repository and organization settings.

**Expected to:**

- Review changes and keep the queue from stalling.
- Make and record decisions in public.
- Own release quality and the roadmap.
- Mentor contributors and reviewers.
- Enforce this document, the [Code of Conduct](./CODE_OF_CONDUCT.md), and the
  [Security Policy](./SECURITY.md).

## Applying for a Role

You apply for yourself. You do not need a nomination from an existing maintainer.

Open a [role application issue](https://github.com/KubeElasti/KubeElasti/issues/new?template=role_application.yml)
and fill in the short form.

### Requirements

**Organization Reviewer**

- At least 1 pull request merged into a KubeElasti repository.
- Active in the community (issues, reviews, discussions, or Slack) for at least 1 month.

**Organization Maintainer**

- At least 2 pull requests merged into a KubeElasti repository.
- Active in the community for at least 1 month before applying.
- Working knowledge of the project's architecture, release process, and contributor workflow.

Non-code work counts: documentation, testing, triage, community support, and release operations
are all valid ways to meet the bar, as long as the contribution history is public and verifiable.

Meeting the minimums makes you eligible, not automatically accepted. Maintainers also weigh the
quality and consistency of your work and your conduct in the community.

### What Happens Next

1. You open the application issue.
2. Maintainers discuss it publicly on the issue. Expect a response within 2 weeks.
3. Maintainers vote on the issue, following [VOTING.md](./VOTING.md).
4. If approved, a maintainer opens a pull request adding you to [MAINTAINERS](./MAINTAINERS) or
   [REVIEWERS](./REVIEWERS), and grants the matching GitHub permissions.
5. If declined, maintainers say why on the issue and what would make a future application
   stronger. You may reapply after 3 months.

## Decision Making

Most decisions need no process. Discuss it in a public issue, pull request, or discussion, and
go with the consensus.

**Lazy consensus** is the default. If a proposal is made in public and no maintainer raises a
substantive objection within a reasonable review period, it proceeds. An objection must come
with reasoning and, where possible, a way to resolve it.

**A formal vote** is required when:

- Maintainers disagree and cannot resolve it through discussion.
- The decision is high impact or hard to reverse.
- This document requires one (adding or removing a role holder, changing governance).

### Voting

The full process, ballot format, and templates are in [VOTING.md](./VOTING.md). In short:

- Only active maintainers vote, one vote each.
- Votes happen in a public GitHub issue, open for 2 weeks.
- At least half of active maintainers must vote for the result to count.
- Adding or removing a reviewer or maintainer needs two-thirds of votes cast.
- Everything else needs a simple majority.
- A tie means the proposal does not pass.

## Approving and Merging Pull Requests

Maintainers merge a pull request when:

- It fits the goals of the project.
- All required CI checks pass. A maintainer may merge past a failing check only after reviewing
  it and confirming the failure is unrelated (for example, a flaky end-to-end test).
- It follows [CONTRIBUTING.md](./CONTRIBUTING.md) and does not violate the
  [Code of Conduct](./CODE_OF_CONDUCT.md).

Non-trivial changes need at least 1 maintainer approval. Changes that alter the API, the
security posture, or the release process need 2. Dependency bumps and small fixes need 1.

If maintainers disagree on a pull request, other maintainers are asked to review and settle it.

## Inactivity and Removal

Reviewers and maintainers are expected to stay reasonably active.

A role holder with no meaningful participation for roughly 3 months who does not answer a
check-in may be marked inactive. Inactive maintainers do not count toward vote totals or quorum.
They can return to active status by resuming participation, with agreement from the active
maintainers.

Anyone may be removed for extended inactivity, repeatedly failing the responsibilities of their
role, or conduct that harms the project or community. Removal requires a two-thirds vote, held
per [VOTING.md](./VOTING.md). Code of Conduct violations are handled through
[CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md) first; a removal vote may follow.

## Stepping Down

You can step down at any time, for any reason, and you do not owe anyone an explanation. It is
better for the project to have an accurate roster than a stale one, and stepping down does not
stop you from contributing or reapplying later.

To step down:

1. Open a public issue in [KubeElasti/KubeElasti](https://github.com/KubeElasti/KubeElasti/issues/new)
   using the template below. There is no issue form for this; copy the text and fill it in.
2. Hand off anything in flight: open pull requests, in-progress releases, and anything you are
   the only person who knows how to do.
3. A maintainer opens a pull request removing you from [MAINTAINERS](./MAINTAINERS) or
   [REVIEWERS](./REVIEWERS) and revokes the matching GitHub, registry, and Slack permissions. No vote is
   needed; stepping down is your decision alone.

**Template:**

```markdown
Title: [STEP DOWN] <your GitHub handle>

## Role

<Organization Reviewer | Organization Maintainer>

## Effective date

<YYYY-MM-DD, or "immediately">

## Reason (optional)

<One line, if you want to share it. Feel free to skip.>

## Handover

- Open PRs / issues to reassign: <links, or "none">
- Releases or duties in progress: <details, or "none">
- Access to revoke: <GitHub org, ghcr.io, Slack admin, anything else>
- Knowledge to write down before I go: <docs to update, or "none">

## Staying involved

<e.g. "I plan to keep contributing occasionally", or "stepping away entirely">
```

Emeritus maintainers keep the credit for their work. Past contributions are not removed from
history, and returning later means applying again through the normal process.

## Communication

- **GitHub issues**: bugs, proposals, roadmap items, role applications, formal votes.
- **GitHub pull requests**: code review and change approval.
- **GitHub discussions**: questions, design exploration, community conversation.
- **CNCF Slack [#kubeelasti](https://cloud-native.slack.com/archives/C0AMUFC5Y3D)**: real-time
  discussion and support.

Project decisions are not made in private, with two exceptions: security reports and Code of
Conduct matters.

## Code of Conduct

KubeElasti follows the [CNCF Code of Conduct](./CODE_OF_CONDUCT.md). It applies to everyone,
including maintainers. Report violations through the path in
[CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md).

## Security

Report security issues only through GitHub Private Vulnerability Reporting, as described in
[SECURITY.md](./SECURITY.md). Never open a public issue for a security problem until coordinated
disclosure allows it. Maintainers may handle reports directly or delegate to a smaller trusted
group.

## Access Control

Repository and organization access follow the roles in this document:

- **[CODEOWNERS](./.github/CODEOWNERS)** assigns default review ownership to the Organization
  Maintainers listed in [MAINTAINERS](./MAINTAINERS). Code and docs ownership in GitHub matches
  those documented roles.
- **Organization Maintainers** receive write (or admin, when needed for releases and settings)
  on KubeElasti repositories, and can merge after the approval rules above.
- **Organization Reviewers** receive triage permissions sufficient to review, label, and manage
  issues and pull requests, without merge rights.
- **Two-factor authentication (2FA)** is required for every member of the KubeElasti GitHub
  organization. Maintainers keep org-wide 2FA enforcement enabled.
- **Branch protection** on default branches requires pull requests, passing required status
  checks, and maintainer review before merge. Force-pushes to protected default branches are
  not allowed.
- Access is granted when someone is added to [MAINTAINERS](./MAINTAINERS) or
  [REVIEWERS](./REVIEWERS), and revoked when they step down or are removed.

## Changing This Document

Open a pull request. It needs approval from a simple majority of active maintainers, voted per
[VOTING.md](./VOTING.md). Say clearly
in the description what is changing and why, and give the community time to comment before
merging.
