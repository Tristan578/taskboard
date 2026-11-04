ALTER TABLE tickets ADD COLUMN repeat_rule TEXT NOT NULL DEFAULT '' CHECK(repeat_rule IN ('','daily','weekly','monthly'));
ALTER TABLE tickets ADD COLUMN repeat_anchor INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tickets ADD COLUMN repeat_source_id TEXT REFERENCES tickets(id) ON DELETE SET NULL;
CREATE TABLE restore_receipts (request_id TEXT PRIMARY KEY, project_ids TEXT NOT NULL, created_at DATETIME NOT NULL);
CREATE UNIQUE INDEX idx_repeat_source ON tickets(repeat_source_id);
