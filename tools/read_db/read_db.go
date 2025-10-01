package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := "/Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db"

	// Allow custom path from command line
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	fmt.Printf("Database path: %s\n", dbPath)
	fmt.Println("Opening database in read-only mode...\n")

	// Open the database in read-only mode
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== PebbleDB Contents ===\n")

	// Create an iterator to read all key-value pairs
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	maxDisplay := 50 // Limit display to first 50 entries

	// Iterate through all keys
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		value := iter.Value()

		count++
		if count <= maxDisplay {
			fmt.Printf("Key: %s\n", key)

			// Try to pretty-print if it's JSON
			var jsonData interface{}
			if err := json.Unmarshal(value, &jsonData); err == nil {
				prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
				fmt.Printf("Value (JSON):\n%s\n", string(prettyJSON))
			} else {
				// If not JSON, display as raw string (truncated if too long)
				valueStr := string(value)
				if len(valueStr) > 200 {
					valueStr = valueStr[:200] + "... (truncated)"
				}
				fmt.Printf("Value: %s\n", valueStr)
			}
			fmt.Println("---")
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	fmt.Printf("\nTotal keys found: %d\n", count)
	if count > maxDisplay {
		fmt.Printf("(Showing first %d entries)\n", maxDisplay)
	}
}
