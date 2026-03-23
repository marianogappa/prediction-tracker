package sqlite

import "database/sql"

const schema = `
CREATE TABLE IF NOT EXISTS predictions (
    id          TEXT PRIMARY KEY,
    statement   TEXT NOT NULL,
    rule        TEXT NOT NULL,
    asset       TEXT NOT NULL,
    start_time  INTEGER NOT NULL,
    deadline    INTEGER NOT NULL,
    source_url  TEXT NOT NULL DEFAULT '',
    author_name TEXT NOT NULL DEFAULT '',
    author_url  TEXT NOT NULL DEFAULT '',
    state       TEXT NOT NULL DEFAULT 'draft',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
    id            TEXT PRIMARY KEY,
    prediction_id TEXT NOT NULL,
    type          TEXT NOT NULL,
    timestamp     INTEGER NOT NULL,
    payload       TEXT,
    FOREIGN KEY (prediction_id) REFERENCES predictions(id)
);
CREATE INDEX IF NOT EXISTS idx_events_prediction ON events(prediction_id, timestamp);

CREATE TABLE IF NOT EXISTS candle_values (
    prediction_id TEXT NOT NULL,
    timestamp     INTEGER NOT NULL,
    open          REAL NOT NULL,
    high          REAL NOT NULL,
    low           REAL NOT NULL,
    close         REAL NOT NULL,
    source        TEXT NOT NULL,
    PRIMARY KEY (prediction_id, timestamp),
    FOREIGN KEY (prediction_id) REFERENCES predictions(id)
);
`

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
