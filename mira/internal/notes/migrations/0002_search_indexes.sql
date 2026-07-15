-- Generated tsvector column for full-text search (title weighted higher).
ALTER TABLE notes ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('french', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('french', coalesce(content, '')), 'B')
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_notes_search_vector
    ON notes USING GIN (search_vector);

-- Approximate nearest-neighbour index for cosine similarity on embeddings.
-- ivfflat needs some rows to build good clusters; it still works empty,
-- just re-run `REINDEX INDEX idx_note_embeddings_vector;` once you have data.
CREATE INDEX IF NOT EXISTS idx_note_embeddings_vector
    ON note_embeddings USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
