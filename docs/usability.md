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
- A consistent SQLite backup and installation-specific MCP settings are downloadable.
- Windows per-user installer, macOS app/package, Debian package and Linux local
  launcher are built alongside the npm binaries.

## Acceptance evidence and unresolved limits

Automated tests cover first-board creation, retries without duplicates, invalid
plans without partial writes, everyday versus software completion, backup restore
reads, HTTP boundaries, and task handoff UI. CI builds native installer artifacts.
Record the actual final CI results in the pull request.

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
