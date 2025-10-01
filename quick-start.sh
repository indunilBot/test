#!/bin/bash

# Quick Start Script for GPaw Explorer

echo "========================================="
echo "GPaw Explorer - Quick Start"
echo "========================================="
echo ""

SOURCE_DB="/Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db"
TEMP_DB="/tmp/slot-db-$(date +%Y%m%d-%H%M%S)"

echo "Step 1: Copying database (this avoids locking issues)..."
echo "  From: $SOURCE_DB"
echo "  To:   $TEMP_DB"
echo ""

if [ ! -d "$SOURCE_DB" ]; then
    echo "ERROR: Source database not found at $SOURCE_DB"
    exit 1
fi

cp -r "$SOURCE_DB" "$TEMP_DB"

if [ $? -ne 0 ]; then
    echo "ERROR: Failed to copy database"
    exit 1
fi

echo "✓ Database copied successfully!"
echo ""
echo "Step 2: Starting Wails dev server (using Yarn)..."
echo ""
echo "========================================="
echo "INSTRUCTIONS:"
echo "========================================="
echo "1. Wait for the app window to open"
echo "2. Click 'New Connection' button"
echo "3. Enter name: slot-db"
echo "4. When file picker opens, paste this path:"
echo ""
echo "   $TEMP_DB"
echo ""
echo "5. Click 'Select' or 'Open'"
echo "6. Browse your blockchain data!"
echo ""
echo "Note: This project uses Yarn for package management"
echo "========================================="
echo ""

# Start wails dev (will use yarn dev as configured in wails.json)
wails dev
