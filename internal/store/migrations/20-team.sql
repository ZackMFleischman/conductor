CREATE TABLE team_runs (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id),
 coordinator_session_id TEXT NOT NULL, coordination_id TEXT UNIQUE,
 status TEXT NOT NULL CHECK(status IN ('starting','ready','degraded','stopping','stopped','released')),
 revision INTEGER NOT NULL CHECK(revision>0), epoch INTEGER NOT NULL CHECK(epoch>0),
 profile TEXT NOT NULL, created_at TEXT NOT NULL,
 UNIQUE(project_id,id),
 FOREIGN KEY(project_id,coordinator_session_id) REFERENCES sessions(project_id,id)
);
CREATE UNIQUE INDEX team_project_coordinator ON team_runs(project_id) WHERE coordination_id IS NOT NULL;
CREATE TABLE team_launches (
 id TEXT PRIMARY KEY, project_id TEXT NOT NULL, run_id TEXT NOT NULL,
 role TEXT NOT NULL, agent_id TEXT NOT NULL, ticket_id TEXT,
 host_id TEXT, session_id TEXT,
 host_state TEXT NOT NULL CHECK(host_state IN ('starting','unknown','active','finished','stopped')),
 observed_epoch INTEGER NOT NULL DEFAULT 0, acknowledged_epoch INTEGER NOT NULL DEFAULT 0,
 checkpoint TEXT NOT NULL DEFAULT '', evidence TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
 UNIQUE(project_id,host_id), UNIQUE(project_id,session_id),
 FOREIGN KEY(project_id,run_id) REFERENCES team_runs(project_id,id),
 FOREIGN KEY(project_id,agent_id) REFERENCES agents(project_id,id),
 FOREIGN KEY(project_id,session_id) REFERENCES sessions(project_id,id),
 FOREIGN KEY(project_id,ticket_id) REFERENCES tickets(project_id,id)
);
CREATE UNIQUE INDEX team_active_identity ON team_launches(project_id,agent_id) WHERE host_state NOT IN ('finished','stopped');
