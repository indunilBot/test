package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/pebble"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/syncmap"
)

var connections syncmap.Map // name (string) -> path (string)
var dbs syncmap.Map         // name (string) -> *pebble.DB

const (
	defaultMaxTotalKeys     = 5000
	defaultMaxKeysPerPrefix = 500
)

func getEnvInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		fmt.Printf("WARN: invalid %s value %q, using default %d\n", key, raw, defaultValue)
		return defaultValue
	}
	return value
}

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
// Optimized version - loads keys in chunks for better performance
func (a *App) GetKeysByPrefix(dbName string) map[string][]string {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		fmt.Println("ERROR: Database not found:", dbName)
		return map[string][]string{}
	}
	db := dbAny.(*pebble.DB)

	maxTotalKeys := getEnvInt("GPAW_MAX_KEYS_TOTAL", defaultMaxTotalKeys)
	maxKeysPerPrefix := getEnvInt("GPAW_MAX_KEYS_PER_PREFIX", defaultMaxKeysPerPrefix)
	if maxTotalKeys <= 0 {
		maxTotalKeys = defaultMaxTotalKeys
	}
	if maxKeysPerPrefix <= 0 {
		maxKeysPerPrefix = defaultMaxKeysPerPrefix
	}

	prefixes := make(map[string][]string)
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		fmt.Println("ERROR: Failed to create iterator:", err)
		return map[string][]string{}
	}
	defer iter.Close()

	const initialCapacity = 100
	totalProcessed := 0
	totalStored := 0
	truncatedTotal := false
	truncatedPrefixes := make(map[string]int)

	for iter.First(); iter.Valid(); iter.Next() {
		if totalProcessed >= maxTotalKeys {
			truncatedTotal = true
			break
		}

		totalProcessed++

		keyBytes := iter.Key()
		key := formatKeyForDisplay(keyBytes)

		colonIdx := strings.IndexByte(key, ':')
		var prefix string

		if colonIdx > 0 {
			prefix = key[:colonIdx]
		} else if strings.HasPrefix(key, "0x") && len(key) > 6 {
			prefix = key[:6]
		} else if len(key) > 4 {
			prefix = key[:4]
		} else {
			prefix = "misc"
		}

		keysForPrefix, exists := prefixes[prefix]
		if !exists {
			keysForPrefix = make([]string, 0, initialCapacity)
		}

		if len(keysForPrefix) >= maxKeysPerPrefix {
			truncatedPrefixes[prefix]++
			prefixes[prefix] = keysForPrefix
			continue
		}

		keysForPrefix = append(keysForPrefix, key)
		prefixes[prefix] = keysForPrefix
		totalStored++
	}

	if err := iter.Error(); err != nil {
		fmt.Println("ERROR: Iterator error:", err)
		return map[string][]string{}
	}

	fmt.Printf("SUCCESS: Loaded %d keys into %d prefixes (processed %d total)\n", totalStored, len(prefixes), totalProcessed)

	if truncatedTotal {
		fmt.Printf("WARNING: Hit GPAW_MAX_KEYS_TOTAL limit (%d); results truncated.\n", maxTotalKeys)
	}
	if len(truncatedPrefixes) > 0 {
		fmt.Printf("WARNING: %d prefixes exceeded GPAW_MAX_KEYS_PER_PREFIX limit (%d).\n", len(truncatedPrefixes), maxKeysPerPrefix)
		count := 0
		for prefix, skipped := range truncatedPrefixes {
			if count >= 3 {
				break
			}
			fmt.Printf("  - Prefix '%s': skipped %d additional keys\n", prefix, skipped)
			count++
		}
	}

	count := 0
	for prefix, keys := range prefixes {
		if count < 3 {
			fmt.Printf("  - Prefix '%s': %d keys\n", prefix, len(keys))
			count++
		}
	}

	return prefixes
}

// GetKeysChunked returns keys in chunks with progress updates
func (a *App) GetKeysChunked(dbName string, chunkSize int) map[string]interface{} {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return map[string]interface{}{"error": "database not found"}
	}
	db := dbAny.(*pebble.DB)

	result := make(map[string]interface{})
	prefixes := make(map[string][]string)

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	defer iter.Close()

	count := 0
	const initialCapacity = 100

	for iter.First(); iter.Valid(); iter.Next() {
		keyBytes := iter.Key()
		key := formatKeyForDisplay(keyBytes)

		colonIdx := strings.IndexByte(key, ':')
		var prefix string

		if colonIdx > 0 {
			prefix = key[:colonIdx]
		} else if strings.HasPrefix(key, "0x") && len(key) > 6 {
			prefix = key[:6]
		} else if len(key) > 4 {
			prefix = key[:4]
		} else {
			prefix = "misc"
		}

		if _, exists := prefixes[prefix]; !exists {
			prefixes[prefix] = make([]string, 0, initialCapacity)
		}

		prefixes[prefix] = append(prefixes[prefix], key)
		count++

		// Return chunk when we hit the limit
		if count >= chunkSize {
			result["keys"] = prefixes
			result["count"] = count
			result["hasMore"] = true
			return result
		}
	}

	result["keys"] = prefixes
	result["count"] = count
	result["hasMore"] = false

	return result
}

// GetKeysWithPagination returns paginated keys for better performance with large datasets
func (a *App) GetKeysWithPagination(dbName string, offset int, limit int) map[string]interface{} {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return map[string]interface{}{"error": "database not found"}
	}
	db := dbAny.(*pebble.DB)

	result := make(map[string]interface{})
	prefixes := make(map[string][]string)

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	defer iter.Close()

	// Skip to offset
	count := 0
	iter.First()
	for count < offset && iter.Valid() {
		iter.Next()
		count++
	}

	// Collect keys up to limit
	collected := 0
	for iter.Valid() && collected < limit {
		keyBytes := iter.Key()
		key := formatKeyForDisplay(keyBytes)

		parts := strings.SplitN(key, ":", 2)
		prefix := parts[0]

		if len(parts) == 1 {
			if strings.HasPrefix(key, "0x") && len(key) > 6 {
				prefix = key[:6]
			} else if len(key) > 4 {
				prefix = key[:4]
			} else {
				prefix = "misc"
			}
		}

		prefixes[prefix] = append(prefixes[prefix], key)
		collected++
		iter.Next()
	}

	result["keys"] = prefixes
	result["hasMore"] = iter.Valid()
	result["offset"] = offset
	result["limit"] = limit
	result["count"] = collected

	return result
}

// KeyValueData represents a key-value pair with metadata
type KeyValueData struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	ValueHex     string `json:"valueHex"`
	ValueBase64  string `json:"valueBase64"`
	Type         string `json:"type"` // "string", "json", "binary"
	Size         int    `json:"size"`
	IsTruncated  bool   `json:"isTruncated"`
	TruncatedMsg string `json:"truncatedMsg,omitempty"`
}

const (
	MaxValueSize      = 1 * 1024 * 1024 // 1MB max for display
	MaxPreviewSize    = 100 * 1024      // 100KB for preview
	MaxHexDisplaySize = 50 * 1024       // 50KB for hex display
	MaxBase64Size     = 50 * 1024       // 50KB for base64
)

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

// truncateWithMessage truncates data and adds a message
func truncateWithMessage(data string, maxSize int, originalSize int) (string, string) {
	if len(data) <= maxSize {
		return data, ""
	}
	truncated := data[:maxSize]
	msg := fmt.Sprintf("\n\n... (Truncated. Showing %s of %s. Use Export to download full data)",
		formatBytes(maxSize), formatBytes(originalSize))
	return truncated, msg
}

// formatBytes formats byte size in human-readable format
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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

	originalSize := len(value)
	valueType := detectValueType(value)
	isTruncated := false
	truncatedMsg := ""

	// Check if value is too large
	if originalSize > MaxValueSize {
		truncatedMsg = fmt.Sprintf("⚠️ Value size (%s) exceeds display limit (%s). Showing preview only. Use Export to download full data.",
			formatBytes(originalSize), formatBytes(MaxValueSize))
		value = value[:MaxPreviewSize]
		isTruncated = true
	}

	result := &KeyValueData{
		Key:         key,
		Type:        valueType,
		Size:        originalSize,
		IsTruncated: isTruncated,
	}

	// Handle different value types with size limits
	if valueType == "json" {
		// For JSON, try to pretty print
		var prettyJSON interface{}
		if err := json.Unmarshal(value, &prettyJSON); err == nil {
			if formatted, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
				displayValue, msg := truncateWithMessage(string(formatted), MaxPreviewSize, originalSize)
				result.Value = displayValue
				if msg != "" {
					result.TruncatedMsg = msg
				}
			}
		} else {
			result.Value = string(value)
		}
	} else if valueType == "string" {
		displayValue, msg := truncateWithMessage(string(value), MaxPreviewSize, originalSize)
		result.Value = displayValue
		if msg != "" {
			result.TruncatedMsg = msg
		}
	} else {
		// Binary data - just show size info
		result.Value = fmt.Sprintf("[Binary Data - %s]", formatBytes(originalSize))
	}

	// Generate hex and base64 only for smaller values
	if originalSize <= MaxHexDisplaySize {
		result.ValueHex = hex.EncodeToString(value)
	} else {
		result.ValueHex = fmt.Sprintf("[Too large for hex display - %s. Use Export to download.]", formatBytes(originalSize))
	}

	if originalSize <= MaxBase64Size {
		result.ValueBase64 = base64.StdEncoding.EncodeToString(value)
	} else {
		result.ValueBase64 = fmt.Sprintf("[Too large for base64 display - %s. Use Export to download.]", formatBytes(originalSize))
	}

	if truncatedMsg != "" && result.TruncatedMsg == "" {
		result.TruncatedMsg = truncatedMsg
	}

	return result
}

// GetKeyCount returns just the count of keys for quick feedback
func (a *App) GetKeyCount(dbName string) int {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return 0
	}
	db := dbAny.(*pebble.DB)

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return 0
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count
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

// ExportValue exports a value to a file (for large values)
func (a *App) ExportValue(dbName string, key string) error {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return fmt.Errorf("database not found")
	}
	db := dbAny.(*pebble.DB)

	// Parse the key (might be hex-encoded)
	keyBytes := parseKeyFromDisplay(key)

	value, closer, err := db.Get(keyBytes)
	if err != nil {
		return fmt.Errorf("error reading value: %v", err)
	}
	defer closer.Close()

	// Show save dialog
	filename := fmt.Sprintf("export_%s.bin", strings.ReplaceAll(key, ":", "_"))
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: filename,
		Title:           "Export Value",
	})
	if err != nil || savePath == "" {
		return err
	}

	// Write to file
	err = os.WriteFile(savePath, value, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
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
