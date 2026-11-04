ALTER TABLE tickets ADD COLUMN revision INTEGER NOT NULL DEFAULT 0;
CREATE TABLE work_state (
 ticket_id TEXT PRIMARY KEY REFERENCES tickets(id) ON DELETE CASCADE,
 agent TEXT NOT NULL DEFAULT '', session_id TEXT NOT NULL DEFAULT '',
 heartbeat_at DATETIME, summary TEXT NOT NULL DEFAULT '',
 completion_url TEXT NOT NULL DEFAULT '', completed_at DATETIME,
 conflict TEXT NOT NULL DEFAULT ''
);
CREATE TABLE sync_locks (project_id TEXT PRIMARY KEY, owner TEXT NOT NULL, expires_at DATETIME NOT NULL);
CREATE TRIGGER ticket_sync_insert AFTER INSERT ON tickets
WHEN NEW.is_draft = 0 AND EXISTS(SELECT 1 FROM projects WHERE id=NEW.project_id AND github_repo != '')
BEGIN
 INSERT INTO sync_jobs(id,project_id,action,status,created_at,updated_at)
 SELECT lower(hex(randomblob(16))),NEW.project_id,'full_sync','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
 WHERE NOT EXISTS(SELECT 1 FROM sync_jobs WHERE project_id=NEW.project_id AND status IN ('pending','failed'));
END;
CREATE TRIGGER ticket_sync_update AFTER UPDATE OF title,description,status,priority,user_story,acceptance_criteria,technical_details,testing_details,is_draft,deleted_at ON tickets
WHEN NEW.is_draft = 0 AND EXISTS(SELECT 1 FROM projects WHERE id=NEW.project_id AND github_repo != '')
BEGIN
 INSERT INTO sync_jobs(id,project_id,action,status,created_at,updated_at)
 SELECT lower(hex(randomblob(16))),NEW.project_id,'full_sync','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
 WHERE NOT EXISTS(SELECT 1 FROM sync_jobs WHERE project_id=NEW.project_id AND status IN ('pending','failed'));
END;
