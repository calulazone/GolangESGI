package notes

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrDuplicate  = errors.New("note already exists")
	ErrNotFound   = errors.New("note not found")
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	if connString == "" {
		connString = os.Getenv("MIRA_DATABASE_URL")
	}
	if connString == "" {
		return nil, fmt.Errorf("MIRA_DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, connString)
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

func NewStoreFromEnv(ctx context.Context) (NoteStore, error) {
	if os.Getenv("MIRA_STORE") == "postgres" || os.Getenv("MIRA_DATABASE_URL") != "" {
		return NewPostgresStore(ctx, "")
	}
	return NewJSONLStore("")
}

func (s *PostgresStore) Save(n *Note) error {
	if n == nil {
		return ErrValidation
	}
	if strings.TrimSpace(n.Title) == "" {
		return ErrValidation
	}
	if n.ID == "" {
		n.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}

	tags := normalizeTags(n.Tags)
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO notes (id, title, content, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			created_at = EXCLUDED.created_at
	`, n.ID, n.Title, n.Content, n.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, n.ID)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		_, err = tx.Exec(ctx, `
			INSERT INTO note_tags (note_id, tag)
			VALUES ($1, $2)
		`, n.ID, tag)
		if err != nil {
			return err
		}
	}

	embeddingValue := ""
	if n.Embedding != "" {
		embeddingValue = n.Embedding
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO note_embeddings (note_id, embedding, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (note_id) DO UPDATE SET embedding = EXCLUDED.embedding
	`, n.ID, embeddingValue, n.CreatedAt)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) All() ([]*Note, error) {
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, content, created_at
		FROM notes
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		tags, err := s.loadTags(ctx, n.ID)
		if err != nil {
			return nil, err
		}
		n.Tags = tags
		out = append(out, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PostgresStore) List(limit int) ([]*Note, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit >= len(all) {
		return all, nil
	}
	return all[len(all)-limit:], nil
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func runMigrations(ctx context.Context, pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		content, err := migrationFS.ReadFile(migrationPath(entry.Name()))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("migrate %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func migrationPath(name string) string {
	return filepath.ToSlash(filepath.Join("migrations", name))
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
