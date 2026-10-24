# Usability follow-through

Scope: reduce the distance from installation to a useful first task for people with
and without coding experience. The user's October 23, 2026 direction expands the
product beyond developer-only setup while retaining verified software completion.

## Ground truth

| Source | Authority | Consequence |
| --- | --- | --- |
| Current user direction | Product intent | No developer vocabulary, account, repo or API key required for first value |
| Prior completion requirement | Product intent | Software board Done still means verified merge or deployment |
| CLI, browser and migration 010 | Implemented baseline | Previously required CLI installation and exposed technical setup first |
| Migration 011 and guided-start handlers | New implementation | Everyday completion is an explicit per-board policy; old boards remain verified |
| Package workflow and tests | Implementation evidence | Native packages are built, but signing and public release are separate, outstanding steps |

The previous CLI-local AGENTS.md conflicted with the user's completion rule and
used outdated commands. It is replaced with the current workflow. Old screenshots
are removed from the README entry path because they are not evidence of the new UI.

## Shipped behavior in this branch

- Launching without arguments opens the local browser and starts/reuses the service.
- First-run routing is based on saved projects, not an easily lost browser flag.
- Guided setup collects a goal and reviewed tasks, saving them in one transaction.
  A stable request ID prevents duplicate boards after a lost response/retry.
- Examples are static, editable starter plans, not fabricated AI output.
- Existing AI chat is usable through reviewed copy/paste with no new credentials.
- Browser navigation is responsive. Technical tools and integration settings are
  optional. Failure messages keep the current input available.
- Existing boards retain verified completion. New everyday boards allow explicit
  completion without GitHub and cannot be linked to GitHub or silently converted.
- Consistent SQLite backups and installation-specific MCP settings are downloadable.
  Daily automatic backups, reviewed restoration into separate boards and deleted-task
  Undo are available under Backups & recovery. Completion-driven daily/weekly/monthly
  tasks preserve monthly anchors and avoid duplicate occurrences.
- Windows per-user installer, macOS app/package, Debian package and Linux local
  launcher are built alongside the npm binaries.
- Next up brings overdue tasks, reviews, blockers and stale agent activity into
  one searchable list. Ordering follows visible rules, not an AI-generated score.
- Browser saves, moves, completion and deletion include a revision check. A stale
  editor keeps its draft, shows the saved fields for comparison and requires an
  explicit choice before replacing them. MCP advertises the same optional check;
  legacy clients must adopt it to gain protection against stale client snapshots.
- Due dates are shown as calendar dates and can be cleared, as can team assignments.
  Field and relationship edits commit together so failed labels do not leave a
  partially saved task. Failed deletion leaves the editor open with an error.
- The local server adds a Content Security Policy, framing and referrer limits,
  browser Fetch Metadata checks and request/WebSocket size limits. Markdown images
  become explicit links. This remains a trusted local app, not an authenticated
  server for remote access.

## Acceptance evidence and unresolved limits

Automated tests cover first-board creation, retries without duplicates, invalid
plans without partial writes, everyday versus software completion, backup restore
reads, HTTP boundaries, and task handoff UI. CI builds native installer artifacts.
The pull request and launch assessment link execution evidence for the tested code commit.

Additional tests exercise stale edits through storage, HTTP and MCP, atomic
rollback, date/team clearing, conflict comparison, unsaved-close warnings,
deletion failure, task ordering, Markdown links and browser request boundaries.

## Completion and release evidence

Automatic backups, guided recovery, deletion Undo and recurring tasks are implemented.
Recovery checks cover supported old backups, schema validation, transactional rollback,
retry idempotency, staging limits, expiration, retention, and failed backup visibility.
Recurrence checks cover month ends, leap years, late completion, repeated completion,
and rollback when the next occurrence cannot be saved.

The package workflow now installs and exercises matching native binaries on disposable
runners. Other architectures remain package-build checks. A separate CI job imports
issues through GitHub's actual API using read-only permissions and a transport guard
that rejects mutations. At code commit `a320efb`, all six installer builds and the
CI suite passed. Installed workflows passed on Linux x64, macOS ARM64 and Windows
x64; the live read imported 10 issues in one request. Windows uninstall preserved
the test boards. Real two-way writes
in a disposable repository are a separate validation, not covered by that live read.

`release-test-matrix.json` and `launch-assessment.json` keep release gates explicit.
`usability-acceptance.md` provides the device/participant protocol; it is not a claim
that people have completed it. Cloud Browser returned ERR_CONNECTION_REFUSED when
opening the running local app, so visual inspection remains unavailable here.

No autonomous app builder, provider subscription bridge, hosted multi-device sync,
or universal agent connector is claimed. A first-app template is a guided project,
not an app-generation service. Visual browser QA and clean-machine interactive
installation remain release checks. Mac Developer ID signing/notarization and
Windows code signing require release-owner credentials; no credentials were
invented, obtained, or bypassed. Keep this as a preview until those checks pass.

Technical reference for release ownership:
[Apple distribution signing](https://developer.apple.com/developer-id/),
[Apple notarization](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution),
[Inno Setup architecture rules](https://jrsoftware.org/ishelp/topic_archidentifiers.htm).
