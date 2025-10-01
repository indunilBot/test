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
	fmt.Println("Trying to open database in read-only mode...")
	fmt.Println("If this fails, the database may be locked by another process")
	fmt.Println()

	// Try to open with ErrorIfNotExists to get better error
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly:         true,
		ErrorIfNotExists: false,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v\n\nPlease close any applications using this database and try again.", err)
	}
	defer db.Close()

	fmt.Println("=== PebbleDB Contents ===\n")

	// Create an iterator
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	maxDisplay := 30

	// Iterate through keys
	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		value := iter.Value()
		count++

		if count <= maxDisplay {
			fmt.Printf("\n=== Entry #%d ===\n", count)
			fmt.Printf("Key: %s\n", key)

			// Try to parse as JSON
			var jsonData interface{}
			if err := json.Unmarshal(value, &jsonData); err == nil {
				prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
				fmt.Printf("Value (JSON):\n%s\n", string(prettyJSON))
			} else {
				// Display as hex or string
				valueStr := string(value)
				if len(valueStr) > 500 {
					valueStr = valueStr[:500] + "... (truncated)"
				}
				fmt.Printf("Value: %s\n", valueStr)
			}
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	fmt.Printf("\n\n=== Summary ===\n")
	fmt.Printf("Total keys found: %d\n", count)
	if count > maxDisplay {
		fmt.Printf("(Displayed first %d entries)\n", maxDisplay)
	}

	// Also show some stats
	fmt.Printf("\nTo see all data, increase 'maxDisplay' in the code.\n")
}
