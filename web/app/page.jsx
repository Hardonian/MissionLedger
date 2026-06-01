const API_BASE_URL = process.env.MISSIONLEDGER_API_BASE_URL || "http://127.0.0.1:8080";

async function fetchJSON(path) {
  const response = await fetch(`${API_BASE_URL}${path}`, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`${path} returned ${response.status}`);
  }
  return response.json();
}

function summarize(missions) {
  return missions.reduce(
    (acc, mission) => {
      acc.total += 1;
      acc.budget += Number(mission.budget_usd || 0);
      acc.budgetUsed += Number(mission.budget_used_usd || 0);
      if (mission.state === "waiting_approval") acc.waitingApproval += 1;
      if (mission.state === "degraded") acc.degraded += 1;
      return acc;
    },
    { total: 0, waitingApproval: 0, degraded: 0, budget: 0, budgetUsed: 0 },
  );
}

function budgetPercent(mission) {
  if (!mission || !mission.budget_usd) return 0;
  return Math.min(100, Math.round((mission.budget_used_usd / mission.budget_usd) * 100));
}

async function loadConsoleData() {
  try {
    const health = await fetchJSON("/healthz");
    const missionsPayload = await fetchJSON("/v1/missions?limit=20");
    const missions = missionsPayload.missions || [];
    const selected = missions[0] ? await fetchJSON(`/v1/missions/${missions[0].id}`) : null;
    return {
      apiError: null,
      health,
      missions,
      selected,
      summary: summarize(missions),
    };
  } catch (error) {
    return {
      apiError: error.message,
      health: null,
      missions: [],
      selected: null,
      summary: summarize([]),
    };
  }
}

function formatMoney(value) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value || 0);
}

function formatTime(value) {
  if (!value) return "—";
  return new Date(value).toLocaleString();
}

export default async function Page() {
  const { apiError, health, missions, selected, summary } = await loadConsoleData();

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">MissionLedger / operator console</p>
          <h1>Governed missions with explicit degraded truth</h1>
          <p className="subtitle">
            Read-only first pass. Backed by the live MissionLedger API. No fake green states.
          </p>
        </div>
        <div className={`status-panel ${apiError ? "status-panel--degraded" : ""}`}>
          <span className="status-label">API</span>
          <strong>{apiError ? "degraded" : health?.status || "unknown"}</strong>
          <span className="status-meta">storage={health?.storage || "unreachable"}</span>
          <span className="status-meta">base={API_BASE_URL}</span>
        </div>
      </section>

      {apiError ? (
        <section className="banner banner--degraded">
          <strong>API unreachable.</strong>
          <span>{apiError}</span>
        </section>
      ) : null}

      <section className="metrics-grid">
        <article className="metric-card">
          <span>Total missions</span>
          <strong>{summary.total}</strong>
        </article>
        <article className="metric-card">
          <span>Waiting approval</span>
          <strong>{summary.waitingApproval}</strong>
        </article>
        <article className="metric-card">
          <span>Degraded missions</span>
          <strong>{summary.degraded}</strong>
        </article>
        <article className="metric-card">
          <span>Budget used</span>
          <strong>{formatMoney(summary.budgetUsed)}</strong>
          <small>of {formatMoney(summary.budget)}</small>
        </article>
      </section>

      <section className="content-grid">
        <article className="panel">
          <div className="panel-header">
            <div>
              <h2>Recent missions</h2>
              <p>Latest 20 missions from the API.</p>
            </div>
          </div>
          {missions.length === 0 ? (
            <div className="empty-state">
              <strong>No missions yet.</strong>
              <p>Create one via the API or run ./scripts/smoke.sh to generate real mission traffic.</p>
            </div>
          ) : (
            <div className="mission-list">
              {missions.map((mission) => (
                <div key={mission.id} className={`mission-row ${selected?.id === mission.id ? "mission-row--selected" : ""}`}>
                  <div>
                    <strong>{mission.id}</strong>
                    <p>{mission.objective}</p>
                  </div>
                  <div className="mission-meta">
                    <span className={`badge badge--${mission.state}`}>{mission.state}</span>
                    <span>{formatMoney(mission.budget_used_usd)} / {formatMoney(mission.budget_usd)}</span>
                    <span>{mission.events?.length || 0} proof events</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </article>

        <article className="panel">
          <div className="panel-header">
            <div>
              <h2>Mission detail</h2>
              <p>{selected ? selected.id : "No mission selected"}</p>
            </div>
          </div>

          {!selected ? (
            <div className="empty-state">
              <strong>No detail yet.</strong>
              <p>The console shows the newest mission once the API has data.</p>
            </div>
          ) : (
            <>
              <div className="detail-grid">
                <div>
                  <span className="detail-label">Tenant</span>
                  <strong>{selected.tenant_id}</strong>
                </div>
                <div>
                  <span className="detail-label">State</span>
                  <strong>{selected.state}</strong>
                </div>
                <div>
                  <span className="detail-label">Created by</span>
                  <strong>{selected.created_by}</strong>
                </div>
                <div>
                  <span className="detail-label">Approved by</span>
                  <strong>{selected.approved_by || "—"}</strong>
                </div>
              </div>

              <div className="budget-card">
                <div className="budget-header">
                  <span>Budget burn</span>
                  <strong>{formatMoney(selected.budget_used_usd)} / {formatMoney(selected.budget_usd)}</strong>
                </div>
                <div className="budget-bar">
                  <div className="budget-bar__fill" style={{ width: `${budgetPercent(selected)}%` }} />
                </div>
              </div>

              <div className="panel-subsection">
                <h3>Requested tools</h3>
                <div className="chip-row">
                  {(selected.requested_tools || []).map((tool) => (
                    <span key={tool} className="chip">{tool}</span>
                  ))}
                </div>
              </div>

              <div className="panel-subsection">
                <h3>Proof timeline</h3>
                <div className="event-list">
                  {(selected.events || []).map((event) => (
                    <div key={`${event.sequence}-${event.created_at}`} className="event-row">
                      <div className="event-seq">#{event.sequence}</div>
                      <div>
                        <strong>{event.event_type}</strong>
                        <p>{event.reason}</p>
                        <small>
                          {event.policy_decision} · {event.verification_state} · {event.tool_name || "no-tool"}
                        </small>
                      </div>
                      <time>{formatTime(event.created_at)}</time>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </article>
      </section>
    </main>
  );
}
