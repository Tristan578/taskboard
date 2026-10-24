# First-use acceptance

Status: **not yet observed with participants**. Automated component and installed
package checks do not establish that first-time users understand the product.

For guided connection, use [the first-user pilot script](first-user-pilot.md).
The implemented contract is in [agent connection](agent-connection.md).

Use a fresh OS account and synthetic tasks. Ask the participant to think aloud;
do not explain the interface unless they cannot continue. Record where assistance
was needed and the participant's exact wording, without recording private tasks.

| Task | Pass condition | Record |
| --- | --- | --- |
| Install and open | Finds the correct installer and opens Player2 through the OS shortcut | OS, architecture, elapsed time, warnings, assistance |
| Plan something useful | Creates a board and changes one example task to match a real goal | Unexpected terms, abandoned steps |
| Work with existing AI | Finds the prompt, understands what will be shared, returns useful notes | Whether they expect automatic chat synchronization |
| Protect an edit | Sees a concurrent-edit warning, compares saved work and retains the desired notes | Any accidental loss or confusion |
| Repeat weekly work | Sets a date and repeat rule, completes it, finds one next task | Whether next date and fresh checklist make sense |
| Recover a deletion | Deletes a synthetic task and uses Undo or Deleted tasks | Finds notes and checklist intact |
| Recover a backup | Previews a backup, confirms separate restored boards, finds their content | Understands current boards and GitHub connection behavior |
| Return later | Closes/reopens Player2 and chooses a useful task from Next up | Ability to continue without coaching |

Start with people who have never configured a developer tool, including a shift
worker or household organizer, plus a technical user. This is a formative test,
not a representative statistical sample. A reasonable initial acceptance target
is independent first-task completion within five minutes after installation,
with no data loss and correct understanding of local storage and AI sharing.
The target is a product judgment to validate, not a measured result.

Block broader release on reproducible lost work, unintentionally shared content,
unrecoverable installation failures, or misleading completion/sync behavior.
Retest changed flows with fresh participants. Keep signing, live two-way GitHub
validation and interactive OS behavior visible in `launch-assessment.json`.
