-- Existing software boards retain verified merge/deployment completion.
ALTER TABLE projects ADD COLUMN completion_policy TEXT NOT NULL DEFAULT 'verified'
    CHECK(completion_policy IN ('verified', 'manual'));
CREATE TABLE workspace_requests (
    request_id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE
);
