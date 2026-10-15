CREATE TABLE IF NOT EXISTS organizations (
 id INTEGER PRIMARY KEY, github_id INTEGER, login TEXT NOT NULL UNIQUE COLLATE NOCASE, url TEXT NOT NULL DEFAULT '',
 last_sync_at TEXT, last_sync_status TEXT NOT NULL DEFAULT '', last_sync_error TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS repositories (
 id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL REFERENCES organizations(id), github_id INTEGER NOT NULL,
 owner TEXT NOT NULL, name TEXT NOT NULL, full_name TEXT NOT NULL UNIQUE COLLATE NOCASE, url TEXT NOT NULL DEFAULT '',
 description TEXT NOT NULL DEFAULT '', visibility TEXT NOT NULL DEFAULT '', archived INTEGER NOT NULL DEFAULT 0,
 language TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL, UNIQUE(org_id, github_id)
);
CREATE TABLE IF NOT EXISTS alerts (
 id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL REFERENCES organizations(id), repository_id INTEGER NOT NULL REFERENCES repositories(id),
 github_id INTEGER NOT NULL, tool TEXT NOT NULL, tool_version TEXT NOT NULL DEFAULT '', tool_guid TEXT NOT NULL DEFAULT '',
 rule_id TEXT NOT NULL, rule_name TEXT NOT NULL DEFAULT '', rule_description TEXT NOT NULL DEFAULT '', rule_tags TEXT NOT NULL DEFAULT '[]', severity TEXT NOT NULL DEFAULT '',
 message TEXT NOT NULL DEFAULT '', ref TEXT NOT NULL DEFAULT '', commit_sha TEXT NOT NULL DEFAULT '', analysis_key TEXT NOT NULL DEFAULT '',
 environment TEXT NOT NULL DEFAULT '', category TEXT NOT NULL DEFAULT '',
 path TEXT NOT NULL DEFAULT '', start_line INTEGER NOT NULL DEFAULT 0, end_line INTEGER NOT NULL DEFAULT 0,
 start_column INTEGER NOT NULL DEFAULT 0, end_column INTEGER NOT NULL DEFAULT 0, url TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, is_open INTEGER NOT NULL DEFAULT 1,
 UNIQUE(repository_id, github_id)
);
CREATE TABLE IF NOT EXISTS sync_runs (
 id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL REFERENCES organizations(id), repository_id INTEGER REFERENCES repositories(id),
 started_at TEXT NOT NULL, finished_at TEXT, status TEXT NOT NULL, error TEXT NOT NULL DEFAULT '',
 repository_count INTEGER NOT NULL DEFAULT 0, alert_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_alerts_open ON alerts(is_open);
CREATE INDEX IF NOT EXISTS idx_alerts_group ON alerts(tool, rule_id);
CREATE INDEX IF NOT EXISTS idx_repositories_org ON repositories(org_id);
