# PebbleDB Reader Tools

This directory contains tools to read and inspect PebbleDB databases.

## Problem: Database is Locked

If the database is locked by another process (like `main.exe`), you have two options:

### Option 1: Stop the locking process
```bash
# Find the process
lsof /path/to/database/directory

# Kill it
kill <PID>
```

### Option 2: Copy the database first
```bash
# Use the provided script
./copy_and_read.sh

# This will copy the database to /tmp and provide the new path
```

## Tools Available

### 1. `simple_reader/` - Best for viewing data
Shows formatted JSON output with summary statistics.

**Usage:**
```bash
# Using default path
cd simple_reader
go run .

# Using custom path
cd simple_reader
go run . /path/to/database

# Using a temporary copy
./copy_and_read.sh
cd simple_reader
go run . /tmp/slot-db-readonly-<timestamp>
```

### 2. `read_db/` - Compact view
Shows first 50 entries in a compact format.

**Usage:**
```bash
# Using default path
cd read_db
go run .

# Using custom path
cd read_db
go run . /path/to/database
```

## For the Wails App

If you want to use the GUI (Wails app):

1. **Copy the database first** (if it's locked):
   ```bash
   cp -r /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db /tmp/slot-db-copy
   ```

2. **Run the Wails app:**
   ```bash
   cd ..
   wails dev
   ```

3. **In the UI:**
   - Click "New Connection"
   - Enter a name (e.g., "slot-db")
   - Select the directory: `/tmp/slot-db-copy`
   - Browse your data!

## Troubleshooting

**Error: "resource temporarily unavailable"**
- The database is locked by another process
- Use Option 1 or Option 2 above

**Error: "no such file or directory"**
- Check the database path is correct
- Make sure the database directory exists

**Error: compile errors**
- Make sure you're in the tools directory: `cd tools`
- The go.mod file is in the parent directory, which is correct
