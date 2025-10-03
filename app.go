package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/pebble"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	db, err := pebble.Open(path, &pebble.Options{
		ReadOnly:       true,
		ErrorIfExists:  false,
		DisableWAL:     true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "resource temporarily unavailable") {
			return fmt.Errorf("database is locked by another process. Please close any applications using this database and try again")
		}
		return fmt.Errorf("failed to open database: %v", err)
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

// formatKeyForDisplay converts binary keys to readable format
func formatKeyForDisplay(keyBytes []byte) string {
	// If it's valid UTF-8 and printable, use as-is
	if utf8.Valid(keyBytes) {
		str := string(keyBytes)
		if isPrintable(str) {
			return str
		}
	}
	// Otherwise, show as hex with prefix
	return "0x" + hex.EncodeToString(keyBytes)
}

// GetKeysByPrefix returns keys grouped by prefix (e.g., "user:" -> "user").
func (a *App) GetKeysByPrefix(dbName string) map[string][]string {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return map[string][]string{}
	}
	db := dbAny.(*pebble.DB)

	prefixes := make(map[string][]string)
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return map[string][]string{}
	}
	defer iter.Close()

	count := 0
	maxKeys := 10000 // Limit to 10k keys for UI performance

	for iter.First(); iter.Valid(); iter.Next() {
		if count >= maxKeys {
			break
		}

		keyBytes := iter.Key()
		key := formatKeyForDisplay(keyBytes)

		// Extract prefix - try both ':' and first few bytes
		parts := strings.SplitN(key, ":", 2)
		prefix := parts[0]

		// If no colon, group by first few characters or hex prefix
		if len(parts) == 1 {
			if strings.HasPrefix(key, "0x") && len(key) > 6 {
				prefix = key[:6] // Group by first 3 bytes in hex
			} else if len(key) > 4 {
				prefix = key[:4]
			} else {
				prefix = "misc"
			}
		}

		prefixes[prefix] = append(prefixes[prefix], key)
		count++
	}

	if err := iter.Error(); err != nil {
		return map[string][]string{}
	}

	return prefixes
}

// KeyValueData represents a key-value pair with metadata
type KeyValueData struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueHex    string `json:"valueHex"`
	ValueBase64 string `json:"valueBase64"`
	Type        string `json:"type"` // "string", "json", "binary"
	Size        int    `json:"size"`
}

// detectValueType tries to determine the best representation for the value
func detectValueType(data []byte) string {
	// Check if it's valid UTF-8 string
	if utf8.Valid(data) {
		str := string(data)
		// Try to parse as JSON
		var js json.RawMessage
		if json.Unmarshal(data, &js) == nil {
			return "json"
		}
		// Check if it's printable string
		if isPrintable(str) {
			return "string"
		}
	}
	return "binary"
}

// isPrintable checks if a string contains mostly printable characters
func isPrintable(s string) bool {
	printableCount := 0
	for _, r := range s {
		if r >= 32 && r <= 126 || r == '\n' || r == '\t' {
			printableCount++
		}
	}
	return len(s) > 0 && float64(printableCount)/float64(len(s)) > 0.95
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

// parseKeyFromDisplay converts display key back to bytes (handles hex keys)
func parseKeyFromDisplay(displayKey string) []byte {
	// If it starts with "0x", it's a hex-encoded key
	if strings.HasPrefix(displayKey, "0x") {
		decoded, err := hex.DecodeString(displayKey[2:])
		if err == nil {
			return decoded
		}
	}
	// Otherwise, use as-is
	return []byte(displayKey)
}

// GetValueWithMetadata returns the value with multiple format representations
func (a *App) GetValueWithMetadata(dbName string, key string) *KeyValueData {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return nil
	}
	db := dbAny.(*pebble.DB)

	// Parse the key (might be hex-encoded)
	keyBytes := parseKeyFromDisplay(key)

	value, closer, err := db.Get(keyBytes)
	if err != nil {
		return &KeyValueData{
			Key:   key,
			Value: fmt.Sprintf("Error reading value: %v", err),
			Type:  "error",
		}
	}
	defer closer.Close()

	valueType := detectValueType(value)

	result := &KeyValueData{
		Key:         key,
		Value:       string(value),
		ValueHex:    hex.EncodeToString(value),
		ValueBase64: base64.StdEncoding.EncodeToString(value),
		Type:        valueType,
		Size:        len(value),
	}

	// If it's JSON, pretty print it
	if valueType == "json" {
		var prettyJSON interface{}
		if err := json.Unmarshal(value, &prettyJSON); err == nil {
			if formatted, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
				result.Value = string(formatted)
			}
		}
	}

	return result
}

// GetDatabaseStats returns statistics about the database for debugging.
func (a *App) GetDatabaseStats(dbName string) map[string]interface{} {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return map[string]interface{}{"error": "database not found"}
	}
	db := dbAny.(*pebble.DB)

	stats := make(map[string]interface{})

	// Count total keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		stats["error"] = err.Error()
		return stats
	}
	defer iter.Close()

	keyCount := 0
	var firstKey, lastKey string
	for iter.First(); iter.Valid(); iter.Next() {
		if keyCount == 0 {
			firstKey = formatKeyForDisplay(iter.Key())
		}
		lastKey = formatKeyForDisplay(iter.Key())
		keyCount++

		// Limit iteration for stats to avoid hanging
		if keyCount >= 100000 {
			break
		}
	}

	stats["totalKeys"] = keyCount
	stats["firstKey"] = firstKey
	stats["lastKey"] = lastKey

	if err := iter.Error(); err != nil {
		stats["iterError"] = err.Error()
	}

	return stats
}

// OpenDirectoryDialog opens a native directory picker and returns the selected path.
func (a *App) OpenDirectoryDialog() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select PebbleDB Directory",
	})
	return path, err
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
		newDB, err = pebble.Open(newPath, &pebble.Options{
			ReadOnly:      true,
			ErrorIfExists: false,
			DisableWAL:    true,
		})
		if err != nil {
			if strings.Contains(err.Error(), "resource temporarily unavailable") {
				return fmt.Errorf("database is locked by another process. Please close any applications using this database and try again")
			}
			return fmt.Errorf("failed to open database: %v", err)
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
