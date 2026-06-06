# MissionLedger Web Console

Current truth:
- this is now a real Next.js app scaffold, not a placeholder doc
- it is read-only in v1
- it renders live MissionLedger API data from `MISSIONLEDGER_API_BASE_URL`
- if the API is unreachable, it shows a degraded banner instead of pretending the system is healthy

Run locally:

```bash
cd web
npm install
MISSIONLEDGER_API_BASE_URL=http://127.0.0.1:8080 npm run dev
```

What the first view shows:
- API health and storage mode
- recent missions
- approval/degraded counts
- budget burn summary
- newest mission detail
- proof event timeline
