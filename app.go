package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/syncmap"
)

var connections syncmap.Map 
var dbs syncmap.Map         

const (
	defaultMaxTotalKeys     = 5000
	defaultMaxKeysPerPrefix = 500
	defaultPageLimit        = 1000
	maxPageLimit            = 5000
)

// readOnlyFS wraps vfs.FS to ignore LOCK file for concurrent read-only access
type readOnlyFS struct {
	vfs.FS
}

// Lock returns a no-op lock for read-only access, allowing concurrent readers
func (fs *readOnlyFS) Lock(name string) (io.Closer, error) {
	// For read-only access, we bypass the lock mechanism entirely
	// This allows multiple processes to open the DB concurrently
	return &noOpLock{}, nil
}

type noOpLock struct{}

func (l *noOpLock) Close() error { return nil }

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
	// Use custom FS that bypasses LOCK file for concurrent read-only access
	roFS := &readOnlyFS{FS: vfs.Default}

	db, err := pebble.Open(path, &pebble.Options{
		ReadOnly:         true,
		ErrorIfExists:    false,
		ErrorIfNotExists: false,
		DisableWAL:       true,
		FS:               roFS, // Use custom FS that ignores locks
	})

	if err != nil {
		if strings.Contains(err.Error(), "resource temporarily unavailable") ||
		   strings.Contains(err.Error(), "lock") ||
		   strings.Contains(err.Error(), "LOCK") {
			return fmt.Errorf("database is locked by another process. Unable to open in read-only mode: %v", err)
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
	// Check for common prefixes and format accordingly
	key := string(keyBytes)

	// Handle "block:<number>:<hash>" format
	if strings.HasPrefix(key, "block:") {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) == 3 {
			prefix := parts[0]
			blockNum := parts[1]
			hashBytes := []byte(parts[2])

			// Convert hash to hex if it contains non-printable characters
			if !isPrintable(parts[2]) {
				hashHex := hex.EncodeToString(hashBytes)
				return fmt.Sprintf("%s:%s:%s", prefix, blockNum, hashHex)
			}
		}
	}

	// Handle "txh_idx:<hash>" format
	if strings.HasPrefix(key, "txh_idx:") {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			hashBytes := []byte(parts[1])
			if !isPrintable(parts[1]) {
				hashHex := hex.EncodeToString(hashBytes)
				return fmt.Sprintf("txh_idx:%s", hashHex)
			}
		}
	}

	// If it's valid UTF-8 and printable, use as-is
	if utf8.Valid(keyBytes) {
		if isPrintable(key) {
			return key
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

type KeyPageResult struct {
	Prefixes   map[string][]string `json:"prefixes"`
	Count      int                 `json:"count"`
	HasMore    bool                `json:"hasMore"`
	NextCursor string              `json:"nextCursor,omitempty"`
	LastKey    string              `json:"lastKey,omitempty"`
	Error      string              `json:"error,omitempty"`
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

func (a *App) GetKeysByPrefixPage(dbName string, cursor string, limit int) *KeyPageResult {
	result := &KeyPageResult{
		Prefixes: make(map[string][]string),
	}

	dbAny, ok := dbs.Load(dbName)
	if !ok {
		result.Error = "database not found"
		return result
	}
	db := dbAny.(*pebble.DB)

	if limit <= 0 {
		limit = defaultPageLimit
	} else if limit > maxPageLimit {
		limit = maxPageLimit
	}

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		result.Error = fmt.Sprintf("failed to create iterator: %v", err)
		return result
	}
	defer iter.Close()

	if cursor == "" {
		if !iter.First() {
			if err := iter.Error(); err != nil {
				result.Error = fmt.Sprintf("iterator error: %v", err)
			}
			return result
		}
	} else {
		cursorBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			result.Error = "invalid cursor"
			return result
		}
		if !iter.SeekGE(cursorBytes) {
			if err := iter.Error(); err != nil {
				result.Error = fmt.Sprintf("iterator error: %v", err)
			}
			return result
		}
		if bytes.Equal(iter.Key(), cursorBytes) {
			if !iter.Next() {
				if err := iter.Error(); err != nil {
					result.Error = fmt.Sprintf("iterator error: %v", err)
				}
				return result
			}
		}
	}

	const initialCapacity = 100
	count := 0
	var lastKeyRaw []byte
	lastKeyDisplay := ""

	for iter.Valid() && count < limit {
		keyBytes := append([]byte(nil), iter.Key()...)
		keyDisplay := formatKeyForDisplay(keyBytes)

		colonIdx := strings.IndexByte(keyDisplay, ':')
		var prefix string

		if colonIdx > 0 {
			prefix = keyDisplay[:colonIdx]
		} else if strings.HasPrefix(keyDisplay, "0x") && len(keyDisplay) > 6 {
			prefix = keyDisplay[:6]
		} else if len(keyDisplay) > 4 {
			prefix = keyDisplay[:4]
		} else {
			prefix = "misc"
		}

		keysForPrefix, exists := result.Prefixes[prefix]
		if !exists {
			keysForPrefix = make([]string, 0, initialCapacity)
		}

		keysForPrefix = append(keysForPrefix, keyDisplay)
		result.Prefixes[prefix] = keysForPrefix

		lastKeyRaw = keyBytes
		lastKeyDisplay = keyDisplay
		count++

		if !iter.Next() {
			break
		}
	}

	if err := iter.Error(); err != nil {
		result.Error = fmt.Sprintf("iterator error: %v", err)
		return result
	}

	result.Count = count
	result.LastKey = lastKeyDisplay
	if iter.Valid() {
		result.HasMore = true
	}
	if result.HasMore && lastKeyRaw != nil {
		result.NextCursor = base64.StdEncoding.EncodeToString(lastKeyRaw)
	}

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
	sampleKeys := make([]string, 0, 10)

	for iter.First(); iter.Valid(); iter.Next() {
		if keyCount == 0 {
			firstKey = formatKeyForDisplay(iter.Key())
		}
		lastKey = formatKeyForDisplay(iter.Key())

		// Collect sample keys to help understand key format
		if keyCount < 10 {
			sampleKeys = append(sampleKeys, formatKeyForDisplay(iter.Key()))
		}

		keyCount++

		// Limit iteration for stats to avoid hanging
		if keyCount >= 100000 {
			break
		}
	}

	stats["totalKeys"] = keyCount
	stats["firstKey"] = firstKey
	stats["lastKey"] = lastKey
	stats["sampleKeys"] = sampleKeys

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
		// Use custom FS that bypasses LOCK file for concurrent read-only access
		roFS := &readOnlyFS{FS: vfs.Default}

		newDB, err = pebble.Open(newPath, &pebble.Options{
			ReadOnly:         true,
			ErrorIfExists:    false,
			ErrorIfNotExists: false,
			DisableWAL:       true,
			FS:               roFS, // Use custom FS that ignores locks
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

// SearchResult represents a search result with metadata
type SearchResult struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	BlockNumber int64  `json:"blockNumber,omitempty"`
	TxHash      string `json:"txHash,omitempty"`
	Type        string `json:"type"` // "block" or "transaction"
}

// SearchByBlockNumber searches for a block by its block number using indexed keys
// Key format: "block:<20-digit-padded-number>:<block_hash>"
// Example: "block:00000000000000001244:abc123..."
func (a *App) SearchByBlockNumber(dbName string, blockNumber int64) (*KeyValueData, error) {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return nil, fmt.Errorf("database not found")
	}
	db := dbAny.(*pebble.DB)

	// Use the exact indexed key format from paw-corenet-layer
	// Key format: "block:<20-digit-padded>:<hash>"
	prefix := fmt.Sprintf("block:%020d:", blockNumber)
	fmt.Printf("🔍 Searching for block %d with prefix: %s\n", blockNumber, prefix)

	// Use efficient prefix iteration with bounds
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte(prefix),
		UpperBound: []byte(prefix + "\xff"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Seek directly to the prefix
	if iter.SeekGE([]byte(prefix)) && iter.Valid() {
		keyBytes := iter.Key()
		keyDisplay := formatKeyForDisplay(keyBytes)

		// Check if this key matches our prefix
		if bytes.HasPrefix(keyBytes, []byte(prefix)) {
			value := iter.Value()
			fmt.Printf("✓ Found block %d instantly via indexed key: %s\n", blockNumber, keyDisplay)
			return a.parseKeyValueData(keyDisplay, value), nil
		}
	}

	// If not found with indexed format, try other common formats
	alternativeKeys := []string{
		fmt.Sprintf("block:%d", blockNumber),
		fmt.Sprintf("%d", blockNumber),
		fmt.Sprintf("block_%d", blockNumber),
		fmt.Sprintf("%020d", blockNumber),
	}

	for _, altKey := range alternativeKeys {
		value, closer, err := db.Get([]byte(altKey))
		if err == nil {
			defer closer.Close()
			fmt.Printf("✓ Found block %d via alternative key format: %s\n", blockNumber, altKey)
			return a.parseKeyValueData(altKey, value), nil
		}
	}

	// Last resort: full scan with progress reporting
	fmt.Printf("⚠️  Block %d not found in indexes. Starting full scan (this will be slow)...\n", blockNumber)

	fullIter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer fullIter.Close()

	count := 0
	const reportInterval = 50000
	blockNumStr := strconv.FormatInt(blockNumber, 10)
	blockNumBytes := []byte(blockNumStr)

	for fullIter.First(); fullIter.Valid(); fullIter.Next() {
		count++

		if count%reportInterval == 0 {
			fmt.Printf("⏳ Scanned %d keys...\n", count)
		}

		value, err := fullIter.ValueAndErr()
		if err != nil {
			continue
		}

		// Fast path - check if value contains the block number
		if !bytes.Contains(value, blockNumBytes) {
			continue
		}

		// Parse JSON and verify block_number
		var data map[string]interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			continue
		}

		if block, ok := data["block"].(map[string]interface{}); ok {
			if blockNum, ok := block["block_number"].(float64); ok {
				if int64(blockNum) == blockNumber {
					foundKey := formatKeyForDisplay(fullIter.Key())
					fmt.Printf("✓ Found block %d after scanning %d keys\n", blockNumber, count)
					return a.parseKeyValueData(foundKey, value), nil
				}
			}
		}
	}

	return nil, fmt.Errorf("block %d not found (scanned %d keys)", blockNumber, count)
}

// checkBlockNumber helper to check if an iterator position contains the target block number
func (a *App) checkBlockNumber(iter *pebble.Iterator, blockNumber int64) *KeyValueData {
	value, err := iter.ValueAndErr()
	if err != nil {
		return nil
	}

	// Try fast path - check if value contains the block number string
	blockNumStr := strconv.FormatInt(blockNumber, 10)
	if !bytes.Contains(value, []byte(blockNumStr)) {
		return nil
	}

	// Parse JSON and check block_number
	var data map[string]interface{}
	if err := json.Unmarshal(value, &data); err != nil {
		return nil
	}

	// Check if this is a block entry
	if block, ok := data["block"].(map[string]interface{}); ok {
		if blockNum, ok := block["block_number"].(float64); ok {
			if int64(blockNum) == blockNumber {
				foundKey := formatKeyForDisplay(iter.Key())
				return a.parseKeyValueData(foundKey, value)
			}
		}
	}

	return nil
}

// SearchByTxHash searches for a transaction by its hash using indexed keys
// Index key format: "txh_idx:<tx_hash>" -> points to block key
// Then reads the block to get the full data
func (a *App) SearchByTxHash(dbName string, txHash string) (*KeyValueData, error) {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return nil, fmt.Errorf("database not found")
	}
	db := dbAny.(*pebble.DB)

	// Use the exact indexed key format from paw-corenet-layer
	// Index format: "txh_idx:<tx_hash>" -> block key
	txHashIndexKey := fmt.Sprintf("txh_idx:%s", txHash)
	fmt.Printf("🔍 Searching for transaction with index key: %s\n", txHashIndexKey)

	// Try to get the block key from the transaction index
	blockKey, closer, err := db.Get([]byte(txHashIndexKey))
	if err == nil {
		defer closer.Close()
		blockKeyDisplay := formatKeyForDisplay(blockKey)
		fmt.Printf("✓ Found transaction index, reading block: %s\n", blockKeyDisplay)

		// Get the actual block data
		blockData, closer2, err := db.Get(blockKey)
		if err == nil {
			defer closer2.Close()
			fmt.Printf("✓ Found transaction instantly via indexed lookup\n")
			return a.parseKeyValueData(blockKeyDisplay, blockData), nil
		}
	}

	// Try alternative index formats
	alternativeIndexKeys := []string{
		fmt.Sprintf("tx:%s", txHash),
		fmt.Sprintf("txhash:%s", txHash),
		fmt.Sprintf("transaction:%s", txHash),
		txHash,
	}

	for _, altKey := range alternativeIndexKeys {
		// Check if it's an index key
		blockKey, closer, err := db.Get([]byte(altKey))
		if err == nil {
			// Try to use it as a block key reference
			blockData, closer2, err := db.Get(blockKey)
			if err == nil {
				blockKeyDisplay := formatKeyForDisplay(blockKey)
				closer.Close()
				closer2.Close()
				fmt.Printf("✓ Found transaction via alternative index: %s -> %s\n", altKey, blockKeyDisplay)
				return a.parseKeyValueData(blockKeyDisplay, blockData), nil
			}
			closer.Close()

			// Maybe it's direct block data
			fmt.Printf("✓ Found transaction via direct key: %s\n", altKey)
			return a.parseKeyValueData(altKey, blockKey), nil
		}
	}

	// Last resort: full scan
	fmt.Printf("⚠️  Transaction %s not found in indexes. Starting full scan (this will be slow)...\n", txHash)

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	const reportInterval = 50000
	txHashBytes := []byte(txHash)

	for iter.First(); iter.Valid(); iter.Next() {
		count++

		if count%reportInterval == 0 {
			fmt.Printf("⏳ Scanned %d keys...\n", count)
		}

		value, err := iter.ValueAndErr()
		if err != nil {
			continue
		}

		// Fast path - check if value contains the tx hash
		if !bytes.Contains(value, txHashBytes) {
			continue
		}

		// Parse JSON and verify
		var data map[string]interface{}
		if err := json.Unmarshal(value, &data); err != nil {
			continue
		}

		if block, ok := data["block"].(map[string]interface{}); ok {
			if txs, ok := block["transactions"].([]interface{}); ok {
				for _, tx := range txs {
					if txMap, ok := tx.(map[string]interface{}); ok {
						if hash, ok := txMap["tx_hash"].(string); ok {
							if hash == txHash {
								foundKey := formatKeyForDisplay(iter.Key())
								fmt.Printf("✓ Found transaction after scanning %d keys\n", count)
								return a.parseKeyValueData(foundKey, value), nil
							}
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("transaction %s not found (scanned %d keys)", txHash, count)
}

// SearchKeys searches for keys matching a pattern with prefix optimization
func (a *App) SearchKeys(dbName string, searchTerm string, searchType string, limit int) []string {
	dbAny, ok := dbs.Load(dbName)
	if !ok {
		return []string{}
	}
	db := dbAny.(*pebble.DB)

	if limit <= 0 {
		limit = 100
	}

	var searchPrefix string
	switch searchType {
	case "block_number":
		searchPrefix = "block:"
	case "tx_hash":
		searchPrefix = "tx:"
	default:
		searchPrefix = ""
	}

	results := make([]string, 0, limit)
	searchKey := searchPrefix + searchTerm

	// Use PebbleDB's efficient prefix iteration
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte(searchKey),
		UpperBound: []byte(searchKey + "\xff"),
	})
	if err != nil {
		return results
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		key := formatKeyForDisplay(iter.Key())
		results = append(results, key)
		count++
	}

	return results
}

// parseKeyValueData helper function to convert raw data to KeyValueData
func (a *App) parseKeyValueData(key string, value []byte) *KeyValueData {
	originalSize := len(value)
	valueType := detectValueType(value)
	isTruncated := false
	truncatedMsg := ""

	// Check if value is too large
	if originalSize > MaxValueSize {
		truncatedMsg = fmt.Sprintf("⚠️ Value size (%s) exceeds display limit (%s). Showing preview only.",
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

	// Handle different value types
	if valueType == "json" {
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
		result.Value = fmt.Sprintf("[Binary Data - %s]", formatBytes(originalSize))
	}

	// Generate hex and base64
	if originalSize <= MaxHexDisplaySize {
		result.ValueHex = hex.EncodeToString(value)
	} else {
		result.ValueHex = fmt.Sprintf("[Too large for hex display - %s]", formatBytes(originalSize))
	}

	if originalSize <= MaxBase64Size {
		result.ValueBase64 = base64.StdEncoding.EncodeToString(value)
	} else {
		result.ValueBase64 = fmt.Sprintf("[Too large for base64 display - %s]", formatBytes(originalSize))
	}

	if truncatedMsg != "" && result.TruncatedMsg == "" {
		result.TruncatedMsg = truncatedMsg
	}

	return result
}

// shutdown closes all DBs (called on app close).
func (a *App) shutdown(ctx context.Context) {
	dbs.Range(func(k, v interface{}) bool {
		db := v.(*pebble.DB)
		db.Close()
		return true
	})
}
