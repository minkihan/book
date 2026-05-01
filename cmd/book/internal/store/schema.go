// Package store provides SQLite storage for the book plan index.
//
// Korean: book 플랜 인덱스를 위한 SQLite 저장소를 제공한다.
package store

// SchemaVersion is the current schema version.
//
// Korean: 현재 스키마 버전.
const SchemaVersion = 6

// DDL contains the SQL statements to create the schema.
//
// Korean: 스키마 생성 SQL 문을 포함한다.
var DDL = []string{
	`CREATE TABLE IF NOT EXISTS meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS plans (
		number       INTEGER PRIMARY KEY,
		slug         TEXT NOT NULL,
		title        TEXT NOT NULL,
		status       TEXT NOT NULL DEFAULT 'backlog',
		project      TEXT NOT NULL DEFAULT '',
		priority     TEXT NOT NULL DEFAULT 'normal',
		created      TEXT,
		completed    TEXT,
		dir_path     TEXT NOT NULL,
		is_archived  INTEGER NOT NULL DEFAULT 0,
		check_total  INTEGER NOT NULL DEFAULT 0,
		check_done   INTEGER NOT NULL DEFAULT 0,
		readme_mtime  INTEGER NOT NULL DEFAULT 0,
		notes_mtime   INTEGER NOT NULL DEFAULT 0,
		check_mtime   INTEGER NOT NULL DEFAULT 0,
		hotfix_count  INTEGER NOT NULL DEFAULT 0,
		hotfix_mtime  INTEGER NOT NULL DEFAULT 0,
		dir_mtime     INTEGER NOT NULL DEFAULT 0,
		documented    INTEGER NOT NULL DEFAULT 0
	)`,

	`CREATE TABLE IF NOT EXISTS plan_tags (
		plan_number INTEGER NOT NULL,
		tag         TEXT NOT NULL,
		PRIMARY KEY (plan_number, tag)
	)`,

	`CREATE INDEX IF NOT EXISTS idx_plan_tags_tag ON plan_tags(tag)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS plans_fts USING fts5(
		plan_number UNINDEXED,
		title, readme_content, notes_content,
		tokenize='trigram'
	)`,
}
