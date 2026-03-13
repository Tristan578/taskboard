# GitHub Actions Failure Review + Ready-to-Use LLM Fix Prompt

Last reviewed: 2026-03-13
Repository: `Tristan578/taskboard`
Workflow: `.github/workflows/ci.yml`

## 1) Failure review (runs so far)

### Run `23055480977` (`workflow_dispatch`, branch `feat/100-percent-coverage`) — **failure**
- `build-and-test` failed at setup:
  - `Unable to resolve action securego/gosec-action, repository not found`
- `frontend-checks` failed during `npm run lint && npm run build` with TypeScript errors:
  - `CreateTicketModal.tsx`: `isDraft` not in `Partial<Ticket>`
  - `TicketPanel.tsx`: `ticket.isDraft` missing on `Ticket` (multiple lines)
  - `Board.tsx`: `ticket.isDraft` missing on `Ticket`
  - `Projects.tsx`: `Cannot find name 'useCallback'`
  - `Teams.tsx`: `Cannot find name 'useCallback'`

### Run `23059759737` (`push`, branch `main`) — **failure**
- Same two failure classes as above:
  1. `build-and-test` setup failure resolving `securego/gosec-action`
  2. `frontend-checks` TypeScript errors for `isDraft` and missing `useCallback`

### Run `23060356104` (`pull_request`, branch `copilot/review-actions-failures`) — **action_required**
- No jobs executed (`total_count: 0`), run concluded as `action_required`.
- This is not a code error from the workflow steps; it indicates a run-level gate/approval state.

## 2) Root-cause summary

1. **Security scanning action reference is invalid**
   - `.github/workflows/ci.yml` currently uses:
     - `uses: securego/gosec-action@master`
   - This action reference cannot be resolved by GitHub Actions in current runs.

2. **Frontend type/model drift**
   - UI code references `ticket.isDraft`, but `Ticket` type does not include `isDraft`.
   - `Projects.tsx` and `Teams.tsx` use `useCallback` without importing it from React.

## 3) Copy/paste prompt for a coding LLM agent

Use the prompt below as-is:

---
You are fixing CI failures in `Tristan578/taskboard`. Make the smallest safe changes necessary to get the GitHub Actions workflow green.

## Context
- Workflow file: `.github/workflows/ci.yml`
- Failing runs:
  - `23055480977`
  - `23059759737`
- Failure classes:
  1. `build-and-test`: `Unable to resolve action securego/gosec-action, repository not found`
  2. `frontend-checks`: TypeScript errors:
     - `src/components/CreateTicketModal.tsx`: `isDraft` not in `Partial<Ticket>`
     - `src/components/TicketPanel.tsx`: `Property 'isDraft' does not exist on type 'Ticket'`
     - `src/pages/Board.tsx`: `Property 'isDraft' does not exist on type 'Ticket'`
     - `src/pages/Projects.tsx`: `Cannot find name 'useCallback'`
     - `src/pages/Teams.tsx`: `Cannot find name 'useCallback'`

## Required outcomes
1. CI workflow uses a valid Go security scan step (or equivalent secure alternative) that actually resolves.
2. Frontend TypeScript compiles cleanly with existing lint/build scripts.
3. Changes are minimal and scoped only to failing areas.

## Constraints
- Do not refactor unrelated code.
- Keep behavior intact unless required for correctness.
- Do not remove tests or reduce quality gates.
- If adding/updating dependencies/actions, pin to stable maintained versions.

## Implementation hints
- Inspect model/type definitions for `Ticket` under `web/src` and align with `isDraft` usage.
- If `isDraft` is intended feature state, add it to shared `Ticket` type(s) and any creation/update payload types as optional where appropriate.
- Add missing `useCallback` imports in:
  - `web/src/pages/Projects.tsx`
  - `web/src/pages/Teams.tsx`
- Fix `.github/workflows/ci.yml` gosec step to a resolvable action or install/run gosec via `go install` + CLI invocation.

## Validation checklist (must run locally)
1. `cd web && npm ci && npm run lint && npm run build`
2. `go test ./...`
3. (If workflow changed) sanity-check YAML and confirm security scan step is valid.

## Definition of done
- All listed TypeScript errors are gone.
- Workflow no longer fails with “Unable to resolve action securego/gosec-action”.
- Existing test/build commands pass.
- PR notes briefly explain each fix and why it was minimal.
---
