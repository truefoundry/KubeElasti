# KubeElasti Voting

How a formal vote is called, cast, counted, and recorded. This is the companion to
[GOVERNANCE.md](./GOVERNANCE.md), which decides _when_ a vote is needed. Most decisions never
reach a vote: lazy consensus is the default.

## When to Vote

A formal vote is required when:

- Maintainers disagree and cannot resolve it through discussion.
- The decision is high impact or hard to reverse.
- [GOVERNANCE.md](./GOVERNANCE.md) requires one: adding or removing a reviewer or maintainer,
  or changing governance.

Stepping down is not a vote. It takes effect when the person says so.

## Who Votes

- Only active maintainers listed in [MAINTAINERS](./MAINTAINERS).
- One vote each.
- Maintainers marked inactive do not vote and do not count toward quorum.
- Reviewers and contributors do not vote, but they can and should comment on the issue.

A maintainer with a personal stake in the outcome (their own application, their own removal)
does not vote on it and is not counted in the totals for that vote.

## Rules

|                                        |                                                                                                             |
| -------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Where                                  | A public GitHub issue in [KubeElasti/KubeElasti](https://github.com/KubeElasti/KubeElasti), labelled `vote` |
| Voting period                          | 2 weeks, or sooner once every eligible maintainer has voted                                                 |
| Quorum                                 | At least half of active maintainers must cast a vote                                                        |
| Add or remove a reviewer or maintainer | Two-thirds of votes cast must be `+1`                                                                       |
| Everything else                        | Simple majority of votes cast                                                                               |
| Tie                                    | The proposal does not pass                                                                                  |
| No quorum                              | The vote fails and may be re-run once, with the period extended to 4 weeks                                  |

Votes are held in private only when confidentiality is required: Code of Conduct matters and
security issues. The outcome is still published, with details withheld as needed.

## Ballots

Vote by commenting on the issue. Start the comment with one of:

- `+1` in favour
- `0` abstain (counts toward quorum, not toward the majority)
- `-1` against

A `-1` must say why. An objection without reasoning does not block a proposal.

You can change your vote until the period closes. Your last comment is the one that counts.

## Calling a Vote

1. Open an issue using the template below. Add the `vote` label.
2. Link the discussion, pull request, or application it came from.
3. State the deadline as an actual date.
4. Notify the other maintainers in [#kubeelasti](https://cloud-native.slack.com/archives/C0AMUFC5Y3D)
   on CNCF Slack.

**Template:**

```markdown
Title: [VOTE] <one-line summary of what is being decided>

## Proposal

<What exactly is being decided. Write it so a "+1" is unambiguous.>

## Background

<Why this needs a vote. Link the discussion, PR, or role application.>

## Threshold

<Two-thirds (role change) | Simple majority (everything else)>

## Eligible voters

<List the active maintainers. Note anyone recused and why.>

## Voting period

Opens: <YYYY-MM-DD>
Closes: <YYYY-MM-DD, 2 weeks later>

## How to vote

Comment on this issue starting with `+1`, `0`, or `-1`. A `-1` must include a reason.
```

## Closing a Vote

When the period ends, the maintainer who called the vote posts the result and closes the issue.

**Template:**

```markdown
## Result: <PASSED | FAILED>

- Eligible voters: <n>
- Votes cast: <n> (quorum <met | not met>)
- +1: <n> (@handles)
- 0: <n> (@handles)
- -1: <n> (@handles)
- Threshold: <two-thirds | simple majority>

## Next steps

<e.g. "PR #123 opened adding @handle to MAINTAINERS", or "Proposal declined, see reasons below.">
```

Follow-up actions, such as updating [MAINTAINERS](./MAINTAINERS) or [REVIEWERS](./REVIEWERS),
link back to the vote issue so the record stays traceable.

## Record

Every vote issue stays open to the public and is never deleted. To find past votes, filter
issues by the `vote` label.
