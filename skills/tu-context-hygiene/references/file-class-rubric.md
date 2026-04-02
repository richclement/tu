# File Class Rubric

Use repo ranking from `tu` plus the following default budgets. These are opinionated defaults for human-agent collaboration, not hard global laws.

## Budget Table

| File class | Target | Warn above | Critical above | Primary concern |
| --- | ---: | ---: | ---: | --- |
| Root agent guidance docs | 200-600 | 800 | 1500 | Root docs should route, not exhaustively teach |
| Root README and onboarding docs | 400-1200 | 1500 | 2500 | Onboarding docs should orient without becoming the whole corpus |
| Architecture/design/implementation index docs | 300-900 | 1200 | 2000 | Index docs should decide what not to read next |
| Leaf markdown docs | 800-2000 | 3000 | 5000 | Leaf docs can be larger, but still need trust and navigation |
| Code files | 800-1800 | 2500 | 4000 | Mixed responsibilities increase context waste quickly |
| Test files | 1200-2500 | 3500 | 5000 | Huge tests often hide multiple behavior areas or fixture families |
| Generated/vendor/fixture files | n/a | n/a | n/a | Usually exclude or deprioritize from remediation |

## Scoring Dimensions

Score each large file qualitatively across these dimensions:

### Token size

- Healthy: within or near the target for its class
- Warning: above the file-class warning budget
- Critical: above the critical budget or clearly dominating the repo

### Navigability

Ask:

- Can the agent tell where to read next?
- Does the file provide a heading map, index, or stable section boundaries?
- Does the structure reduce the need to read the entire file?

Warning signs:

- giant unbroken markdown
- a single code file with many unrelated regions
- no obvious section boundaries

### Ownership

Ask:

- Is there a clear owner or owning team?
- Is the intended audience obvious?

Warning signs:

- no owner metadata
- unclear responsibility for updates

### Freshness

Ask:

- Is there a status or last-updated signal?
- Can the agent tell whether this file is current?

Warning signs:

- stale dates
- no freshness markers on a frequently changing reference doc
- duplicated procedural details that may drift

### Canonicality

Ask:

- Is this the source of truth or a copy of other material?
- Would an agent know which document or module wins if they conflict?

Warning signs:

- multiple docs summarize the same workflow
- policy text duplicated across root docs and leaf docs

### Cohesion

Ask:

- Does the file do one job or several?
- Would splitting reduce context without obscuring the interface?

Warning signs:

- architecture doc mixed with rollout checklist and changelog
- code file mixing parsing, validation, transport, and rendering
- tests covering unrelated subsystems in one file

### Blast radius

Ask:

- Does routine work force edits in this file?
- Does every task reopen the same oversized shared document?

Warning signs:

- one doc becomes the choke point for every update
- one code file is touched by unrelated changes every week

## Severity Heuristics

Use these default severity calls:

- Low: slightly above target, but trustworthy and easy to navigate
- Medium: above warning threshold or missing one major trust signal
- High: above critical threshold, clearly mixed-responsibility, or repeatedly touched by unrelated work

Escalate severity when size combines with poor ownership, poor freshness, or poor navigability. De-escalate when a larger file is intentionally reference-oriented and easy to trust.

## Typical Recommendations By Class

- Root guidance docs -> convert to a map with durable rules and links
- Index docs -> keep index concise and move detail into leaf docs
- Leaf markdown docs -> split by question-to-answer and add metadata
- Code files -> extract stable seams and preserve a clear public interface
- Test files -> split by behavior area, fixture family, or subsystem
