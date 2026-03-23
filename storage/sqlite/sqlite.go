package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
	"github.com/marianogappa/predictions-tracker/storage"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) InsertPrediction(ctx context.Context, p domain.Prediction) error {
	ruleJSON, err := json.Marshal(p.Rule)
	if err != nil {
		return fmt.Errorf("marshaling rule: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO predictions (id, statement, rule, asset, start_time, deadline, source_url, author_name, author_url, state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Statement, string(ruleJSON), p.Asset,
		p.StartTime.Unix(), p.Deadline.Unix(),
		p.SourceURL, p.AuthorName, p.AuthorURL,
		string(p.State),
		p.CreatedAt.Unix(), p.UpdatedAt.Unix(),
	)
	return err
}

func (s *Store) UpdatePrediction(ctx context.Context, p domain.Prediction) error {
	ruleJSON, err := json.Marshal(p.Rule)
	if err != nil {
		return fmt.Errorf("marshaling rule: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE predictions SET statement=?, rule=?, asset=?, start_time=?, deadline=?, source_url=?, author_name=?, author_url=?, state=?, updated_at=?
		 WHERE id=?`,
		p.Statement, string(ruleJSON), p.Asset,
		p.StartTime.Unix(), p.Deadline.Unix(),
		p.SourceURL, p.AuthorName, p.AuthorURL,
		string(p.State), p.UpdatedAt.Unix(),
		p.ID,
	)
	return err
}

func (s *Store) GetPrediction(ctx context.Context, id string) (domain.Prediction, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, statement, rule, asset, start_time, deadline, source_url, author_name, author_url, state, created_at, updated_at
		 FROM predictions WHERE id=?`, id)
	return scanPrediction(row)
}

func (s *Store) ListPredictions(ctx context.Context, filter storage.PredictionFilter) ([]domain.Prediction, error) {
	query := `SELECT id, statement, rule, asset, start_time, deadline, source_url, author_name, author_url, state, created_at, updated_at FROM predictions`
	var conditions []string
	var args []any

	if len(filter.States) > 0 {
		placeholders := make([]string, len(filter.States))
		for i, st := range filter.States {
			placeholders[i] = "?"
			args = append(args, string(st))
		}
		conditions = append(conditions, "state IN ("+strings.Join(placeholders, ",")+")")
	}
	if filter.Asset != "" {
		conditions = append(conditions, "asset = ?")
		args = append(args, filter.Asset)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var preds []domain.Prediction
	for rows.Next() {
		p, err := scanPredictionRows(rows)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	return preds, rows.Err()
}

func (s *Store) InsertEvent(ctx context.Context, e domain.Event) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, prediction_id, type, timestamp, payload) VALUES (?, ?, ?, ?, ?)`,
		e.ID, e.PredictionID, string(e.Type), e.Timestamp.Unix(), string(e.Payload),
	)
	return err
}

func (s *Store) ListEvents(ctx context.Context, predictionID string) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, prediction_id, type, timestamp, payload FROM events WHERE prediction_id=? ORDER BY timestamp ASC`, predictionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) ListAllEvents(ctx context.Context) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, prediction_id, type, timestamp, payload FROM events ORDER BY timestamp ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) InsertValues(ctx context.Context, predictionID string, vals []domain.CandleValue) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO candle_values (prediction_id, timestamp, open, high, low, close, source) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vals {
		if _, err := stmt.ExecContext(ctx, predictionID, v.Timestamp, v.Open, v.High, v.Low, v.Close, v.Source); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetValues(ctx context.Context, predictionID string, from, to time.Time) ([]domain.CandleValue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT timestamp, open, high, low, close, source FROM candle_values
		 WHERE prediction_id=? AND timestamp >= ? AND timestamp <= ?
		 ORDER BY timestamp ASC`,
		predictionID, from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vals []domain.CandleValue
	for rows.Next() {
		var v domain.CandleValue
		if err := rows.Scan(&v.Timestamp, &v.Open, &v.High, &v.Low, &v.Close, &v.Source); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

func (s *Store) GetLastValueTimestamp(ctx context.Context, predictionID string) (int64, error) {
	var ts sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT MAX(timestamp) FROM candle_values WHERE prediction_id=?`, predictionID).Scan(&ts)
	if err != nil {
		return 0, err
	}
	if !ts.Valid {
		return 0, nil
	}
	return ts.Int64, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPrediction(row *sql.Row) (domain.Prediction, error) {
	var (
		p                                    domain.Prediction
		ruleJSON                             string
		startTime, deadline, createdAt, updatedAt int64
		state                                string
	)
	err := row.Scan(&p.ID, &p.Statement, &ruleJSON, &p.Asset, &startTime, &deadline,
		&p.SourceURL, &p.AuthorName, &p.AuthorURL, &state, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, fmt.Errorf("prediction not found")
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal([]byte(ruleJSON), &p.Rule); err != nil {
		return p, fmt.Errorf("unmarshaling rule: %w", err)
	}
	p.State = domain.PredictionState(state)
	p.StartTime = time.Unix(startTime, 0).UTC()
	p.Deadline = time.Unix(deadline, 0).UTC()
	p.CreatedAt = time.Unix(createdAt, 0).UTC()
	p.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return p, nil
}

func scanPredictionRows(rows *sql.Rows) (domain.Prediction, error) {
	var (
		p                                    domain.Prediction
		ruleJSON                             string
		startTime, deadline, createdAt, updatedAt int64
		state                                string
	)
	err := rows.Scan(&p.ID, &p.Statement, &ruleJSON, &p.Asset, &startTime, &deadline,
		&p.SourceURL, &p.AuthorName, &p.AuthorURL, &state, &createdAt, &updatedAt)
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal([]byte(ruleJSON), &p.Rule); err != nil {
		return p, fmt.Errorf("unmarshaling rule: %w", err)
	}
	p.State = domain.PredictionState(state)
	p.StartTime = time.Unix(startTime, 0).UTC()
	p.Deadline = time.Unix(deadline, 0).UTC()
	p.CreatedAt = time.Unix(createdAt, 0).UTC()
	p.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return p, nil
}

func scanEvent(rows *sql.Rows) (domain.Event, error) {
	var (
		e       domain.Event
		ts      int64
		evtType string
		payload sql.NullString
	)
	if err := rows.Scan(&e.ID, &e.PredictionID, &evtType, &ts, &payload); err != nil {
		return e, err
	}
	e.Type = domain.EventType(evtType)
	e.Timestamp = time.Unix(ts, 0).UTC()
	if payload.Valid {
		e.Payload = json.RawMessage(payload.String)
	}
	return e, nil
}
