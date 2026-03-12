ALTER TABLE projects ADD COLUMN github_repo TEXT DEFAULT '';
ALTER TABLE projects ADD COLUMN github_last_synced DATETIME;

ALTER TABLE tickets ADD COLUMN github_issue_number INTEGER;
ALTER TABLE tickets ADD COLUMN github_last_synced_at DATETIME;
ALTER TABLE tickets ADD COLUMN github_last_synced_sha TEXT DEFAULT '';
ALTER TABLE tickets ADD COLUMN user_story TEXT DEFAULT '';
ALTER TABLE tickets ADD COLUMN acceptance_criteria TEXT DEFAULT '';
ALTER TABLE tickets ADD COLUMN technical_details TEXT DEFAULT '';
ALTER TABLE tickets ADD COLUMN testing_details TEXT DEFAULT '';
