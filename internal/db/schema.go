package db

// Schema contains all CREATE TABLE/TRIGGER statements for aimemo.
const Schema = `
CREATE TABLE IF NOT EXISTS entities (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT    NOT NULL,
    entity_type   TEXT    NOT NULL DEFAULT 'concept',
    tags          TEXT    NOT NULL DEFAULT '[]',
    created_at    INTEGER NOT NULL DEFAULT (unixepoch('now', 'subsec') * 1000),
    updated_at    INTEGER NOT NULL DEFAULT (unixepoch('now', 'subsec') * 1000),
    deleted_at    INTEGER,
    access_count  INTEGER NOT NULL DEFAULT 0,
    last_accessed INTEGER,
    UNIQUE(name)
);

CREATE TABLE IF NOT EXISTS observations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id   INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    content     TEXT    NOT NULL,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch('now', 'subsec') * 1000),
    UNIQUE(entity_id, content)
);

CREATE TABLE IF NOT EXISTS relations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id     INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    to_id       INTEGER NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation    TEXT    NOT NULL,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch('now', 'subsec') * 1000),
    UNIQUE(from_id, to_id, relation)
);

CREATE TABLE IF NOT EXISTS journal (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    content    TEXT    NOT NULL,
    tags       TEXT    NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL DEFAULT (unixepoch('now', 'subsec') * 1000)
);

CREATE INDEX IF NOT EXISTS idx_journal_created ON journal(created_at DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS entities_fts USING fts5(
    name,
    entity_type,
    content='entities',
    content_rowid='id',
    tokenize='porter unicode61'
);

CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
    content,
    content='observations',
    content_rowid='id',
    tokenize='porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS entities_fts_insert AFTER INSERT ON entities BEGIN
    INSERT INTO entities_fts(rowid, name, entity_type) VALUES (new.id, new.name, new.entity_type);
END;

CREATE TRIGGER IF NOT EXISTS entities_fts_delete AFTER DELETE ON entities BEGIN
    INSERT INTO entities_fts(entities_fts, rowid, name, entity_type) VALUES('delete', old.id, old.name, old.entity_type);
END;

CREATE TRIGGER IF NOT EXISTS entities_fts_update AFTER UPDATE OF name, entity_type ON entities
WHEN old.deleted_at IS NULL AND new.deleted_at IS NULL BEGIN
    INSERT INTO entities_fts(entities_fts, rowid, name, entity_type) VALUES('delete', old.id, old.name, old.entity_type);
    INSERT INTO entities_fts(rowid, name, entity_type) VALUES (new.id, new.name, new.entity_type);
END;

CREATE TRIGGER IF NOT EXISTS entities_fts_soft_delete AFTER UPDATE OF deleted_at ON entities
WHEN old.deleted_at IS NULL AND new.deleted_at IS NOT NULL BEGIN
    INSERT INTO entities_fts(entities_fts, rowid, name, entity_type) VALUES('delete', old.id, old.name, old.entity_type);
END;

CREATE TRIGGER IF NOT EXISTS entities_fts_soft_restore AFTER UPDATE OF deleted_at ON entities
WHEN old.deleted_at IS NOT NULL AND new.deleted_at IS NULL BEGIN
    INSERT INTO entities_fts(rowid, name, entity_type) VALUES (new.id, new.name, new.entity_type);
END;

CREATE TRIGGER IF NOT EXISTS observations_fts_insert AFTER INSERT ON observations BEGIN
    INSERT INTO observations_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS observations_fts_delete AFTER DELETE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, content) VALUES('delete', old.id, old.content);
END;

CREATE VIRTUAL TABLE IF NOT EXISTS journal_fts USING fts5(
    content,
    content='journal',
    content_rowid='id',
    tokenize='porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS journal_fts_insert AFTER INSERT ON journal BEGIN
    INSERT INTO journal_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS journal_fts_delete AFTER DELETE ON journal BEGIN
    INSERT INTO journal_fts(journal_fts, rowid, content) VALUES('delete', old.id, old.content);
END;
`
