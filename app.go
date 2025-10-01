package main

import (
	"context"
	"strings"

	"github.com/cockroachdb/pebble"
	"golang.org/x/sync/syncmap"
)

var connections syncmap.Map // name (string) -> path (string)
var dbs syncmap.Map         // name (string) -> *pebble.DB

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context contains the app lifecycle hooks and runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AddConnection adds a new PebbleDB connection by opening the directory.
func (a *App) AddConnection(name string, path string) error {
	db, err := pebble.Open(path, &pebble.Options{ReadOnly: true})
	if err != nil {
		return err
	}
	dbs.Store(name, db)
	connections.Store(name, path)
	return nil
}

// GetDatabases returns the list of connected database names.
func (a *App) GetDatabases() []string {
	var list []string
	connections.Range(func(k, v interface{}) bool {
		list = append(list, k.(string))
		return true
	})
	if list == nil {
		return []string{}
	}
	return list
}

// GetKeysByPrefix returns keys grouped by prefix (e.g., "user:" -> "user").
func (a *App) GetKeysByPrefix(dbName string) map[string][]string {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return nil
	}
	db := dbAny.(*pebble.DB)

	prefixes := make(map[string][]string)
	iter, _ := db.NewIter(&pebble.IterOptions{})
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		parts := strings.SplitN(key, ":", 2)
		prefix := parts[0]
		if len(parts) == 1 {
			prefix = "default" // For keys without ':'
		}
		prefixes[prefix] = append(prefixes[prefix], key)
	}
	iter.Close()
	return prefixes
}

// GetValue returns the value for a key as a string (assumed JSON or text).
func (a *App) GetValue(dbName string, key string) string {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return ""
	}
	db := dbAny.(*pebble.DB)
	value, closer, err := db.Get([]byte(key))
	if err != nil {
		return ""
	}
	defer closer.Close()
	return string(value)
}

// shutdown closes all DBs (called on app close).
func (a *App) shutdown(ctx context.Context) {
	dbs.Range(func(k, v interface{}) bool {
		db := v.(*pebble.DB)
		db.Close()
		return true
	})
}
