# Demo Script

Goal:
Show that MissionLedger is better than logs for governing a coding/ops agent.

Flow:
1. Create a mission with `read_file` and `terminal` in scope.
2. Run `read_file` successfully.
3. Attempt `secret.read` and show hard denial.
4. Attempt `terminal` and show escalation.
5. Approve mission.
6. Retry `terminal` and show success.
7. Attempt an over-budget call and show degraded state.
8. Export proofpack.

Success condition:
A skeptical buyer can see exactly:
- who requested the work
- what was in scope
- which risky action was blocked pending approval
- when approval happened
- when budget guardrails tripped
- what proof artifact remains afterward
