package notes

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore opens a pool, runs migrations and registers the pgvector
// type on every new connection.
func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	if connString == "" {
		connString = os.Getenv("MIRA_DATABASE_URL")
	}
	if connString == "" {
		return nil, fmt.Errorf("MIRA_DATABASE_URL is not set")
	}

	// RegisterTypes (below) looks up the "vector" pg_type on every new
	// connection, so the extension must exist *before* that pool is
	// opened - otherwise every connection attempt fails with "vector
	// type not found". Use a plain, type-registration-free connection
	// just to create the extension first.
	if err := ensureVectorExtension(ctx, connString); err != nil {
		return nil, err
	}

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

func ensureVectorExtension(ctx context.Context, connString string) error {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	return err
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) Create(ctx context.Context, n *Note) error {
	if n == nil || strings.TrimSpace(n.Title) == "" || strings.TrimSpace(n.Content) == "" {
		return ErrValidation
	}
	if n.ID == "" {
		n.ID = fmt.Sprintf("note-%d", time.Now().UnixNano())
	}
	now := time.Now().UTC()
	n.CreatedAt, n.UpdatedAt = now, now
	if n.EnrichmentStatus == "" {
		n.EnrichmentStatus = EnrichmentPending
	}
	tags := normalizeTags(n.Tags)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO notes (id, title, content, summary, score, enrichment_status, created_at, updated_at)
		VALUES ($1, $2, $3, '', 0, $4, $5, $6)
	`, n.ID, n.Title, n.Content, n.EnrichmentStatus, n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return err
	}
	if err := replaceTags(ctx, tx, n.ID, tags); err != nil {
		return err
	}
	n.Tags = tags

	return tx.Commit(ctx)
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*Note, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, title, content, summary, score, enrichment_status, created_at, updated_at
		FROM notes WHERE id = $1
	`, id)

	n, err := scanNote(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	tags, err := s.loadTags(ctx, id)
	if err != nil {
		return nil, err
	}
	n.Tags = tags
	return n, nil
}

func (s *PostgresStore) List(ctx context.Context, limit int) ([]*Note, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, content, summary, score, enrichment_status, created_at, updated_at
		FROM notes ORDER BY created_at DESC, id DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		tags, err := s.loadTags(ctx, n.ID)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *PostgresStore) Update(ctx context.Context, id string, input UpdateNoteInput) (*Note, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notes WHERE id = $1)`, id).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	now := time.Now().UTC()
	contentChanged := input.Title != nil || input.Content != nil

	if input.Title != nil {
		if _, err := tx.Exec(ctx, `UPDATE notes SET title = $1, updated_at = $2 WHERE id = $3`, *input.Title, now, id); err != nil {
			return nil, err
		}
	}
	if input.Content != nil {
		if _, err := tx.Exec(ctx, `UPDATE notes SET content = $1, updated_at = $2 WHERE id = $3`, *input.Content, now, id); err != nil {
			return nil, err
		}
	}
	if contentChanged {
		if _, err := tx.Exec(ctx, `UPDATE notes SET enrichment_status = $1, updated_at = $2 WHERE id = $3`, EnrichmentPending, now, id); err != nil {
			return nil, err
		}
	}
	if input.Tags != nil {
		if err := replaceTags(ctx, tx, id, normalizeTags(input.Tags)); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchHybrid ranks notes by a weighted mix of full-text rank and vector
// cosine similarity. When queryEmbedding is nil it degrades gracefully to
// full-text-only ranking.
func (s *PostgresStore) SearchHybrid(ctx context.Context, query string, queryEmbedding []float32, limit int) ([]*Note, error) {
	if limit <= 0 {
		limit = 20
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, ErrValidation
	}

	var (
		rows pgx.Rows
		err  error
	)
	if queryEmbedding != nil {
		vec := pgvector.NewVector(queryEmbedding)
		rows, err = s.pool.Query(ctx, `
			SELECT n.id, n.title, n.content, n.summary, n.score, n.enrichment_status, n.created_at, n.updated_at
			FROM notes n
			LEFT JOIN note_embeddings e ON e.note_id = n.id
			WHERE n.search_vector @@ plainto_tsquery('french', $1)
			   OR e.embedding IS NOT NULL
			ORDER BY
				(0.5 * ts_rank(n.search_vector, plainto_tsquery('french', $1))
				 + 0.5 * COALESCE(1 - (e.embedding <=> $2), 0)) DESC
			LIMIT $3
		`, q, vec, limit)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, title, content, summary, score, enrichment_status, created_at, updated_at
			FROM notes
			WHERE search_vector @@ plainto_tsquery('french', $1)
			ORDER BY ts_rank(search_vector, plainto_tsquery('french', $1)) DESC
			LIMIT $2
		`, q, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Note
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		tags, err := s.loadTags(ctx, n.ID)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *PostgresStore) SetEnrichment(ctx context.Context, id string, status EnrichmentStatus, summary string, score float64, tags []string, embedding []float32) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		UPDATE notes SET summary = $1, score = $2, enrichment_status = $3, updated_at = $4 WHERE id = $5
	`, summary, score, status, now, id)
	if err != nil {
		return err
	}

	if len(tags) > 0 {
		if err := mergeTags(ctx, tx, id, normalizeTags(tags)); err != nil {
			return err
		}
	}

	if embedding != nil {
		vec := pgvector.NewVector(embedding)
		_, err = tx.Exec(ctx, `
			INSERT INTO note_embeddings (note_id, embedding, created_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (note_id) DO UPDATE SET embedding = EXCLUDED.embedding
		`, id, vec, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// --- helpers -----------------------------------------------------------

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNote(row rowScanner) (*Note, error) {
	var n Note
	if err := row.Scan(&n.ID, &n.Title, &n.Content, &n.Summary, &n.Score, &n.EnrichmentStatus, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

func (s *PostgresStore) loadTags(ctx context.Context, noteID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT tag FROM note_tags WHERE note_id = $1 ORDER BY tag`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func replaceTags(ctx context.Context, tx pgx.Tx, noteID string, tags []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, noteID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(ctx, `INSERT INTO note_tags (note_id, tag) VALUES ($1, $2)`, noteID, tag); err != nil {
			return err
		}
	}
	return nil
}

// mergeTags adds enrichment-derived tags without wiping user-supplied ones.
func mergeTags(ctx context.Context, tx pgx.Tx, noteID string, tags []string) error {
	for _, tag := range tags {
		_, err := tx.Exec(ctx, `
			INSERT INTO note_tags (note_id, tag) VALUES ($1, $2)
			ON CONFLICT (note_id, tag) DO NOTHING
		`, noteID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.ToLower(strings.TrimSpace(tag))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func runMigrations(ctx context.Context, pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		content, err := migrationFS.ReadFile(filepath.ToSlash(filepath.Join("migrations", entry.Name())))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("migrate %s: %w", entry.Name(), err)
		}
	}
	return nil
}
