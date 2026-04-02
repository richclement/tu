# Code Patterns

Use these patterns when `tu` highlights large code or test files. The goal is not cosmetic fragmentation. The goal is to reduce mixed responsibility and make targeted reading possible for future agents and developers.

## First-Pass Inspection

Inspect code progressively:

1. Read imports.
2. Read top-level types, constants, and public entrypoints.
3. Locate section boundaries.
4. Search for suspicious hotspots before opening the whole file.

Common hotspot signals:

- giant switch statements
- large literals or embedded data tables
- multiple unrelated public entrypoints
- parser plus validator plus formatter in one file
- tests covering unrelated subsystems

## Good Split Seams

Prefer seams with stable responsibility boundaries:

- parser
- validator
- formatter
- transport client
- config loader
- domain model
- error mapping
- CLI output formatter
- test fixture builders
- shared assertions

The split should make the public interface clearer, not harder to follow.

## Bad Split Seams

Avoid recommending splits that are only cosmetic:

- chopping one coherent module into numbered files
- splitting by arbitrary line counts
- moving helpers out when they are only used once and already local
- creating tiny files that force the reader to bounce constantly

## Large Test Files

Treat large test files as their own class. Good split patterns:

- by behavior area
- by subsystem
- by input family
- by fixture family
- by output contract

If the file is large mostly because of inline fixtures, consider moving fixtures or builders out before splitting test logic itself.

## Recommended Report Language

Use language like:

- "This file appears large because it mixes parsing, validation, and formatting. Recommend extracting those seams into separate files while keeping the exported interface unchanged."
- "This test file appears to cover multiple behavior families and embeds bulky fixtures. Recommend separating shared fixture builders from behavior-specific assertions."

## When Not To Recommend A Split

Do not recommend splitting when:

- the file is above target but still cohesive and easy to navigate
- the size comes mostly from a necessary table or generated block that should instead be excluded from triage
- a split would create a maze of tiny files with no stable ownership boundary
