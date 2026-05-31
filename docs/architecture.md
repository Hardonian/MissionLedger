# Architecture

Current implemented architecture:
- Go HTTP API
- in-memory mission store
- deterministic tool policy package
- proof event trail embedded per mission
- smoke script that proves the end-to-end demo path locally

Planned architecture:
- Go API service for mission creation, approvals, policy, exports
- Go worker or gateway for real tool mediation
- PostgreSQL event store / projections
- Next.js operator console
- export service for JSON + Markdown proofpacks

Core flows:
1. Create mission
2. Attempt tool use
3. Policy decides allow / escalate / deny
4. Human approves risky scope when required
5. Spend is tracked against budget
6. Proofpack exports the full chain
