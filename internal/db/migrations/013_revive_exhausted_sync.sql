-- A task edit must produce runnable sync work even after the board's previous
-- job used all of its automatic retries. Revive the exhausted job (keeping its
-- last error for visibility until it succeeds) instead of being blocked by it.
DROP TRIGGER IF EXISTS ticket_sync_insert;
DROP TRIGGER IF EXISTS ticket_sync_update;
CREATE TRIGGER ticket_sync_insert AFTER INSERT ON tickets
WHEN NEW.is_draft = 0 AND EXISTS(SELECT 1 FROM projects WHERE id=NEW.project_id AND github_repo != '')
BEGIN
 UPDATE sync_jobs SET status='pending',attempts=0,next_retry_at=NULL
 WHERE project_id=NEW.project_id AND status='failed' AND attempts >= 5;
 INSERT INTO sync_jobs(id,project_id,action,status,created_at,updated_at)
 SELECT lower(hex(randomblob(16))),NEW.project_id,'full_sync','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
 WHERE NOT EXISTS(SELECT 1 FROM sync_jobs WHERE project_id=NEW.project_id AND status IN ('pending','failed'));
END;
CREATE TRIGGER ticket_sync_update AFTER UPDATE OF title,description,status,priority,user_story,acceptance_criteria,technical_details,testing_details,is_draft,deleted_at ON tickets
WHEN NEW.is_draft = 0 AND EXISTS(SELECT 1 FROM projects WHERE id=NEW.project_id AND github_repo != '')
BEGIN
 UPDATE sync_jobs SET status='pending',attempts=0,next_retry_at=NULL
 WHERE project_id=NEW.project_id AND status='failed' AND attempts >= 5;
 INSERT INTO sync_jobs(id,project_id,action,status,created_at,updated_at)
 SELECT lower(hex(randomblob(16))),NEW.project_id,'full_sync','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP
 WHERE NOT EXISTS(SELECT 1 FROM sync_jobs WHERE project_id=NEW.project_id AND status IN ('pending','failed'));
END;
