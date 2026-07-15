-- pgvector for embeddings similarity search
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS notes (
    id                TEXT PRIMARY KEY,
    title             TEXT NOT NULL,
    content           TEXT NOT NULL,
    summary           TEXT NOT NULL DEFAULT '',
    score             DOUBLE PRECISION NOT NULL DEFAULT 0,
    enrichment_status TEXT NOT NULL DEFAULT 'pending',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);

-- embedding dimension (384) matches the local hashing embedder; if you plug
-- in a different embedding model, update this dimension and re-migrate.
CREATE TABLE IF NOT EXISTS note_embeddings (
    note_id    TEXT PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    embedding  vector(384),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
