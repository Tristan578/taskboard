# First-user pilot: connect, receive, review

Status: ready to run; **no participant or live vendor-agent sessions have been
observed by this development task**. Do not substitute automated test outcomes.
Use the PR's installer artifacts for an internal preview and disclose outstanding
distribution signing. Respect participants' organization and installation policies.

## First cohort

Recruit three consenting participants: someone organizing household or shift work,
someone preparing a career or small-business task, and a developer with Codex CLI
or Claude Code. Include someone who has never configured MCP. Record existing apps;
do not assume a chat subscription includes an installed local agent. No invitations
have been sent.

Use low-sensitivity goals. Do not record credentials, private task content or screens
without consent. Store research notes in approved private storage, not this public repo.

## Session script

1. Time from opening Player2. Ask: “Use your AI to make a small plan you would find
   useful, and bring its result onto this board.” Let them find **Connect my AI**.
2. Ask what the connection allows the agent to read or change. Observe their app
   choice and access acknowledgement without explaining the interface first.
3. Chat-only users should take the manual path. Record whether they understand it
   does not update automatically. This is not an automatic-connection success.
   Ask whether they would install a local agent; do not force installation to pass.
4. For local agents, observe setup, tool approval and the first-task prompt. Record
   installation/sign-in time separately while retaining total elapsed time. Count
   every help request, terminal difficulty and unclear term.
5. Watch the report and result arrive. Ask what “report received” means, whether it
   proves the agent is still running, and how the result is useful. Let the user
   decide whether to mark Done.
6. Close and reopen the browser. Ask them to find saved work and identify the next
   action, then explain how they would disable the connection in their agent settings.
7. Ask: “Where did you get stuck?”, “What did you expect to happen automatically?”,
   and “Would this save effort next week? What would you use it for?”

With the technical participant, also point an agent at a disposable alternate
database. A missing pilot task should cause troubleshooting, not replacement work.
Do not use production data. Validate live GitHub writes separately in an explicitly
disposable repository; this planning task does not establish GitHub parity.

## Session record

| Field | Record |
| --- | --- |
| Participant | Anonymous code |
| Environment | OS/architecture, Player2 commit, AI app/version, prior agent installation |
| Path | Local agent / generic MCP / manual chat |
| Understanding | Access and saved-work location; exact wording with consent |
| Times | App open, setup complete, first report, saved result, review, successful return |
| Assistance | Count, exact step and explanation required |
| Errors | Visible message, retained work, recovery outcome |
| Utility | Participant's explanation of why the result helps |
| Outcome | Independent / assisted / abandoned / environment-blocked |
| Follow-up | Specific fix and fresh-participant retest |

## Decision rule

Proposed target: a useful first task completed independently within five minutes
after an already-installed agent is ready, with total setup time also reported.
This is a hypothesis, not a result or an exclusion of installation friction. Require
correct access/storage understanding, no lost work and no false connected state.
Report manual and automatic paths separately. Three participants provide formative
evidence, not a conversion estimate.

Block expansion on lost work, unintended sharing, wrong-board writes, misleading
connection status or repeatable setup dead ends. Fix and retest with a fresh
participant. Signing, interactive OS checks and live GitHub writes remain explicit
in `launch-assessment.json`.
