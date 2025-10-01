#!/bin/bash

# Script to copy a locked PebbleDB and read it

SOURCE_DB="/Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db"
TEMP_DB="/tmp/slot-db-readonly-$(date +%s)"

echo "Copying database from:"
echo "  $SOURCE_DB"
echo "To:"
echo "  $TEMP_DB"
echo ""

# Copy the database
cp -r "$SOURCE_DB" "$TEMP_DB"

if [ $? -ne 0 ]; then
    echo "Error: Failed to copy database"
    exit 1
fi

echo "Database copied successfully!"
echo ""
echo "Now you can:"
echo "1. Use this path in your Wails app: $TEMP_DB"
echo "2. Or run: cd tools && go run simple_reader.go"
echo ""
echo "Temporary database path:"
echo "$TEMP_DB"
