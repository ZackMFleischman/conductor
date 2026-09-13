CREATE TABLE ticket_metadata(ticket_id TEXT PRIMARY KEY,project_id TEXT NOT NULL,spec_revision INTEGER NOT NULL DEFAULT 1,parent_id TEXT,kind TEXT NOT NULL DEFAULT 'implementation',blocked_reason TEXT NOT NULL DEFAULT '',submitted_commit TEXT NOT NULL DEFAULT '',submitted_spec_revision INTEGER NOT NULL DEFAULT 0,legacy_human INTEGER NOT NULL DEFAULT 0,FOREIGN KEY(project_id,ticket_id) REFERENCES tickets(project_id,id),FOREIGN KEY(project_id,parent_id) REFERENCES tickets(project_id,id));
INSERT INTO ticket_metadata(ticket_id,project_id,spec_revision,parent_id,kind,blocked_reason,submitted_commit,submitted_spec_revision,legacy_human)
 SELECT t.id,t.project_id,COALESCE(s.spec_revision,1),s.parent_id,COALESCE(s.kind,'implementation'),COALESCE(s.blocked_reason,''),COALESCE(s.submitted_commit,''),COALESCE(s.submitted_spec_revision,0),CASE WHEN s.ticket_id IS NULL THEN 1 ELSE 0 END FROM tickets t LEFT JOIN workflow_ticket_specs s ON s.ticket_id=t.id;
CREATE TRIGGER ticket_metadata_create AFTER INSERT ON tickets BEGIN INSERT INTO ticket_metadata(ticket_id,project_id) VALUES(NEW.id,NEW.project_id); END;
ALTER TABLE workflow_dependencies RENAME TO ticket_dependencies;
ALTER TABLE workflow_versions RENAME TO ticket_versions;
ALTER TABLE workflow_validations RENAME TO ticket_decisions;
ALTER TABLE ticket_decisions ADD COLUMN outcome TEXT NOT NULL DEFAULT 'accepted' CHECK(outcome IN('accepted','rejected'));
ALTER TABLE ticket_decisions ADD COLUMN reason TEXT NOT NULL DEFAULT '';
CREATE TABLE workflow_ticket_specs_v3(ticket_id TEXT PRIMARY KEY,project_id TEXT NOT NULL,validation_mode TEXT NOT NULL,required_checks TEXT NOT NULL DEFAULT '[]',policy_revision INTEGER NOT NULL,execution_mode TEXT NOT NULL,plan_review TEXT NOT NULL,prepared_revision INTEGER NOT NULL DEFAULT 0,authorized_revision INTEGER NOT NULL DEFAULT 0,paused INTEGER NOT NULL DEFAULT 0,FOREIGN KEY(project_id,ticket_id) REFERENCES tickets(project_id,id));
INSERT INTO workflow_ticket_specs_v3 SELECT ticket_id,project_id,validation_mode,required_checks,policy_revision,execution_mode,plan_review,prepared_revision,authorized_revision,paused FROM workflow_ticket_specs;
DROP TABLE workflow_ticket_specs;
ALTER TABLE workflow_ticket_specs_v3 RENAME TO workflow_ticket_specs;
INSERT INTO ticket_versions(id,project_id,ticket_id,spec_revision,actor_id,payload,created_at) SELECT 'migration-'||t.id,t.project_id,t.id,m.spec_revision,'migration',json_object('title',t.title,'body',t.body),t.created_at FROM tickets t JOIN ticket_metadata m ON m.ticket_id=t.id WHERE NOT EXISTS(SELECT 1 FROM ticket_versions v WHERE v.ticket_id=t.id);
CREATE TABLE ticket_submissions(id TEXT PRIMARY KEY,project_id TEXT NOT NULL,ticket_id TEXT NOT NULL,spec_revision INTEGER NOT NULL,session_id TEXT,commit_id TEXT NOT NULL,summary TEXT NOT NULL,evidence TEXT NOT NULL,qa TEXT NOT NULL,created_at TEXT NOT NULL,FOREIGN KEY(project_id,ticket_id) REFERENCES tickets(project_id,id),FOREIGN KEY(project_id,session_id) REFERENCES sessions(project_id,id));
-- Submission provenance comes from the submit event, never a later claim or edit.
-- Zero revision, NULL session and empty timestamp explicitly mean unknown.
WITH source AS (
 SELECT e.id,e.project_id,e.ticket_id,e.actor_id,e.body,e.created_at,
        CASE WHEN json_valid(e.payload) THEN e.payload ELSE '{}' END AS payload
 FROM events e WHERE e.kind='ticket.submit'
), submitted AS (
 SELECT id,project_id,ticket_id,actor_id,created_at,
        COALESCE(json_extract(payload,'$.Summary'),body) AS summary,
        COALESCE(json_extract(payload,'$.Evidence'),'') AS evidence,
        COALESCE(json_extract(payload,'$.QA'),'') AS qa,
        COALESCE(json_extract(payload,'$.Commit'),'') AS commit_id
 FROM source
)
INSERT INTO ticket_submissions
 SELECT 'event-'||e.id,e.project_id,e.ticket_id,
        CASE WHEN e.summary=t.summary AND e.evidence=t.evidence AND e.qa=t.qa AND e.commit_id=m.submitted_commit THEN m.submitted_spec_revision ELSE 0 END,
        s.id,e.commit_id,e.summary,e.evidence,e.qa,e.created_at
 FROM submitted e JOIN tickets t ON t.project_id=e.project_id AND t.id=e.ticket_id
 JOIN ticket_metadata m ON m.ticket_id=t.id
 LEFT JOIN sessions s ON s.project_id=e.project_id AND s.id=e.actor_id
 WHERE e.summary<>'';
-- Retain a snapshot when no matching submit event survived, without fabricating
-- its actor or submission time from unrelated current ticket/claim observations.
INSERT INTO ticket_submissions
 SELECT 'migration-'||t.id,t.project_id,t.id,m.submitted_spec_revision,NULL,m.submitted_commit,t.summary,t.evidence,t.qa,''
 FROM tickets t JOIN ticket_metadata m ON m.ticket_id=t.id
 WHERE t.summary<>'' AND NOT EXISTS(
   SELECT 1 FROM ticket_submissions s WHERE s.project_id=t.project_id AND s.ticket_id=t.id AND s.summary=t.summary AND s.evidence=t.evidence AND s.qa=t.qa
 );
