# Mission audit export

`GET /v1/missions/{id}/audit` returns an attachment-style JSON export with format `missionledger-audit-v1`, mission state, approver metadata, and the append-only proof events. Payload values remain represented by hashes in `ProofEvent`; the endpoint is intended for operator review and audit handoff.
