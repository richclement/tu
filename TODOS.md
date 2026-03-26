# TODOS

## Product

### Built-In Agent-Sensitive File Heuristics

**What:** Add built-in heuristics and labels for agent-sensitive file classes such as `AGENTS.md`, skill docs, and large architecture docs.

**Why:** This could create a sharper out-of-the-box "fix this first" experience once real usage validates which file classes consistently matter for agent context hygiene.

**Context:** The CEO review for `tu` intentionally deferred built-in heuristics from v1 to avoid hard-coding opinions too early. v1 stays profile-driven. After real-world usage data and feedback, `tu` may evolve to auto-highlight categories of files that commonly poison agent context and deserve elevated ranking or labels.

**Effort:** M
**Priority:** P3
**Depends on:** Real-world usage and feedback from v1

### Additional Provider Adapters

**What:** Add post-v1 support for additional counting adapters beyond the initial local exact tokenizer and heuristic fallback.

**Why:** This preserves the path to provider-specific counting when real usage proves it is needed without bloating the initial offline-first release.

**Context:** The eng review for `tu` intentionally reduced v1 scope to a Go CLI with one exact local tokenizer path, heuristic fallback, and no remote counting. If demand appears after launch, this work should add a second adapter behind the same scan pipeline and result contract instead of reopening the v1 scope.

**Effort:** M
**Priority:** P3
**Depends on:** Shipping v1 and validating real demand

## Distribution

### Package Manager Installation

**What:** Add package-manager distribution, likely Homebrew first, after the initial GitHub Releases workflow is stable.

**Why:** This lowers installation friction and makes the CLI easier to adopt once the binary contract and release pipeline settle.

**Context:** The eng review locked v1 distribution to GitHub Release binaries with automated cross-platform builds. After that path is stable, the next distribution step should be one mainstream install route that keeps install and uninstall simple for repeat users.

**Effort:** M
**Priority:** P2
**Depends on:** Stable v1 release workflow
