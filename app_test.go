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

func TestGetKeysByPrefixPage(t *testing.T) {
	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for i := 0; i < 30; i++ {
		key := fmt.Sprintf("user:%04d", i)
		if err := db.Set([]byte(key), []byte("value"), pebble.NoSync); err != nil {
			t.Fatalf("set key %s: %v", key, err)
		}
	}
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("order:%04d", i)
		if err := db.Set([]byte(key), []byte("value"), pebble.NoSync); err != nil {
			t.Fatalf("set key %s: %v", key, err)
		}
	}

	if err := db.Flush(); err != nil {
		t.Fatalf("flush pebble: %v", err)
	}

	const dbName = "test_page_db"
	dbs.Store(dbName, db)
	t.Cleanup(func() { dbs.Delete(dbName) })

	app := NewApp()

	page1 := app.GetKeysByPrefixPage(dbName, "", 20)
	if page1 == nil {
		t.Fatalf("expected page result")
	}
	if page1.Error != "" {
		t.Fatalf("unexpected error: %s", page1.Error)
	}
	if page1.Count != 20 {
		t.Fatalf("expected 20 keys on first page, got %d", page1.Count)
	}
	if !page1.HasMore {
		t.Fatalf("expected more pages")
	}
	if page1.NextCursor == "" {
		t.Fatalf("expected next cursor")
	}

	seen := make(map[string]struct{})
	for _, keys := range page1.Prefixes {
		for _, key := range keys {
			seen[key] = struct{}{}
		}
	}

	page2 := app.GetKeysByPrefixPage(dbName, page1.NextCursor, 20)
	if page2 == nil {
		t.Fatalf("expected second page")
	}
	if page2.Error != "" {
		t.Fatalf("unexpected error on second page: %s", page2.Error)
	}
	if page2.Count != 20 {
		t.Fatalf("expected remaining keys on second page, got %d", page2.Count)
	}

	for _, keys := range page2.Prefixes {
		for _, key := range keys {
			if _, exists := seen[key]; exists {
				t.Fatalf("duplicate key across pages: %s", key)
			}
			seen[key] = struct{}{}
		}
	}

	expectedTotal := 40
	if len(seen) != expectedTotal {
		t.Fatalf("expected %d unique keys, got %d", expectedTotal, len(seen))
	}

	if page2.HasMore {
		t.Fatalf("did not expect more pages")
	}

	invalid := app.GetKeysByPrefixPage(dbName, "not-base64", 5)
	if invalid == nil || invalid.Error == "" {
		t.Fatalf("expected error for invalid cursor")
	}
}
