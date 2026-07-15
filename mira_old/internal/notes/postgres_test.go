package notes

import "testing"

func TestMigrationPathUsesForwardSlashes(t *testing.T) {
	if got := migrationPath("001_init.sql"); got != "migrations/001_init.sql" {
		t.Fatalf("expected migrations/001_init.sql, got %s", got)
	}
}
