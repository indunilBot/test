package main

import (
	"context"
	"fmt"
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
	if _, exists := connections.Load(name); exists {
		return fmt.Errorf("connection '%s' already exists", name)
	}
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

// GetConnectionPaths returns a copy of the connection name -> path map.
func (a *App) GetConnectionPaths() map[string]string {
	result := make(map[string]string)
	connections.Range(func(k, v interface{}) bool {
		result[k.(string)] = v.(string)
		return true
	})
	return result
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

// UpdateConnection allows changing the name or path (or both) for an existing connection.
func (a *App) UpdateConnection(oldName, newName, path string) error {
	connAny, ok := connections.Load(oldName)
	if !ok {
		return fmt.Errorf("connection '%s' not found", oldName)
	}
	oldPath := connAny.(string)

	dbAny, _ := dbs.Load(oldName)
	oldDB, _ := dbAny.(*pebble.DB)

	newPath := path
	if newPath == "" {
		newPath = oldPath
	}

	if newName == "" {
		newName = oldName
	}

	if newName != oldName {
		if _, exists := connections.Load(newName); exists {
			return fmt.Errorf("connection '%s' already exists", newName)
		}
	}

	var newDB *pebble.DB
	var err error

	if newPath != oldPath || oldDB == nil {
		newDB, err = pebble.Open(newPath, &pebble.Options{ReadOnly: true})
		if err != nil {
			return err
		}
	} else {
		newDB = oldDB
	}

	// Remove old mappings
	connections.Delete(oldName)
	dbs.Delete(oldName)

	// Store the updated entries
	connections.Store(newName, newPath)
	dbs.Store(newName, newDB)

	// Close the old DB if we replaced it with a new handle
	if newDB != oldDB && oldDB != nil {
		oldDB.Close()
	}

	return nil
}

// shutdown closes all DBs (called on app close).
func (a *App) shutdown(ctx context.Context) {
	dbs.Range(func(k, v interface{}) bool {
		db := v.(*pebble.DB)
		db.Close()
		return true
	})
}
