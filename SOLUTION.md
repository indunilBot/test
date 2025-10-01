# Solution: Reading PebbleDB Data

## The Problem
Your GPaw Explorer wasn't showing data because the database was **locked** by another process (`main.exe` PID 29595).

PebbleDB uses file locking to prevent concurrent access, so you can't open the database while another process is using it.

## The Solution

### ✅ Fixed Issues:
1. ✅ Removed broken SST reader tool
2. ✅ Fixed all tool files to work correctly
3. ✅ Added command-line argument support
4. ✅ Created helper scripts

### Three Ways to Read Your Data:

#### Option 1: Command-Line Tools (Works Now!)
```bash
# Copy the database first
cp -r /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db /tmp/slot-db-copy

# Read it
cd tools/simple_reader
go run . /tmp/slot-db-copy
```

#### Option 2: Use the Wails GUI
```bash
# Copy the database
cp -r /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db /tmp/slot-db-copy

# Run the app
wails dev

# In the UI:
# 1. Click "New Connection"
# 2. Name it: "slot-db"
# 3. Select path: /tmp/slot-db-copy
# 4. Browse your data!
```

#### Option 3: Stop the Locking Process
```bash
# Find the process
lsof /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db

# Kill it
kill 29595

# Now you can open the original database
```

## What's in Your Database?

Your database contains blockchain data:

- **Keys**: `block:00000000000000000001:...` (block data)
- **Values**: JSON with:
  - `block_number`: Block index
  - `header`: Block header with timestamp, nonce, merkleRoot, etc.
  - `transactions`: List of transactions (empty in most blocks)
  - `validators`: Validator information
  - `reward`: Block reward

Example entry:
```json
{
  "block_number": 1,
  "header": {
    "difficulty": 0,
    "merkleRoot": "QjwiDIH0Tdryud8TGLTMAU3cmQJrLwRu26JpMdSfACc=",
    "nonce": 1759295116,
    "prevHash": "7KkJKQ3OPmX+TCQJvdhTHjrUsk5KmaIvEpObBrnmPzM=",
    "timestamp": 1759295116318796000,
    "version": 0
  },
  "reward": 0,
  "transactions": null,
  "validators": []
}
```

## Files in tools/ Directory

- `read_db/` - Compact viewer (first 50 entries)
- `simple_reader/` - Detailed viewer with JSON formatting (first 30 entries)
- `copy_and_read.sh` - Helper script to copy and read database
- `README.md` - Documentation

All tools now support custom paths:
```bash
cd tools/simple_reader
go run . /path/to/your/database
```

## Troubleshooting

**"resource temporarily unavailable"**
→ Database is locked. Use Option 1 or Option 3 above.

**"no such file or directory"**
→ Check the database path.

**Wails app shows no data**
→ Make sure you clicked "New Connection" and selected the database directory.

**"main redeclared in this block" error in VSCode**
→ This is fixed! Each tool is now in its own subdirectory. Reload VSCode window if you still see it.
