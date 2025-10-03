package main

import (
	"fmt"
	"testing"

	"github.com/cockroachdb/pebble"
)

func TestGetKeysByPrefixRespectsLimits(t *testing.T) {
	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	prefixes := []string{"user", "order", "logs"}
	for _, prefix := range prefixes {
		for i := 0; i < 60; i++ {
			key := fmt.Sprintf("%s:%04d", prefix, i)
			if err := db.Set([]byte(key), []byte("value"), pebble.NoSync); err != nil {
				t.Fatalf("set key %s: %v", key, err)
			}
		}
	}

	if err := db.Flush(); err != nil {
		t.Fatalf("flush pebble: %v", err)
	}

	const dbName = "test_limit_db"
	dbs.Store(dbName, db)
	t.Cleanup(func() { dbs.Delete(dbName) })

	app := NewApp()
	t.Setenv("GPAW_MAX_KEYS_TOTAL", "100")
	t.Setenv("GPAW_MAX_KEYS_PER_PREFIX", "50")

	result := app.GetKeysByPrefix(dbName)
	if len(result) == 0 {
		t.Fatalf("expected prefixes, got none")
	}

	total := 0
	for prefix, keys := range result {
		if len(keys) > 50 {
			t.Fatalf("prefix %s exceeded configured limit: %d keys", prefix, len(keys))
		}
		total += len(keys)
	}

	if total > 100 {
		t.Fatalf("total keys %d exceed configured limit", total)
	}

	if keys, ok := result["logs"]; ok {
		if len(keys) != 50 {
			t.Fatalf("expected logs prefix to be limited to 50 keys, got %d", len(keys))
		}
	} else {
		t.Fatalf("expected logs prefix to be present")
	}

	if _, ok := result["user"]; ok {
		t.Fatalf("expected user prefix to be dropped due to total limit")
	}
}
