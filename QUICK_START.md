# Quick Start Guide

## ✅ All Errors Fixed!

The "main redeclared" errors are **completely fixed**. Each tool now lives in its own directory.

## 🚀 Choose Your Method:

### Method 1: GUI (Easiest)
```bash
./quick-start.sh
```
This will:
1. Copy the database
2. Start the Wails app
3. Show you the path to use

Then in the app UI:
- Click "New Connection"
- Enter name: "slot-db"
- Paste the path shown in terminal
- Browse your data!

### Method 2: Command Line
```bash
# Copy database
cp -r /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db /tmp/my-db

# View data
cd tools/simple_reader
go run . /tmp/my-db
```

### Method 3: Stop Locking Process
```bash
# Find the process
ps aux | grep main.exe

# Kill it (replace PID with actual number)
kill <PID>

# Now run the app normally
wails dev
```

## 📁 Project Structure

```
gpaw-explorer/
├── app.go              # Backend logic
├── main.go             # Main app
├── tools/              # Command-line tools
│   ├── read_db/        # Compact viewer
│   │   └── read_db.go
│   ├── simple_reader/  # Detailed viewer
│   │   └── simple_reader.go
│   ├── copy_and_read.sh
│   └── README.md
├── quick-start.sh      # One-command launcher
├── SOLUTION.md         # Detailed explanation
└── QUICK_START.md      # This file
```

## ✨ What's Fixed:

1. ✅ No more "main redeclared" errors
2. ✅ Tools work without conflicts
3. ✅ Main app compiles perfectly
4. ✅ Can read locked databases (via copying)
5. ✅ Clear documentation

## 🎯 Your Database Contains:

Blockchain data with:
- Block numbers
- Headers (timestamps, hashes, nonces)
- Transactions
- Validators
- Rewards

All stored as JSON in PebbleDB!

## 💡 Tips:

- If you see VSCode errors, reload the window (`Cmd+Shift+P` → "Reload Window")
- Always copy the database if it's locked by another process
- The copied database in `/tmp` is temporary - it will be deleted on reboot
- To change how many entries are shown, edit `maxDisplay` in the tool files

## Need Help?

Check these files:
- `SOLUTION.md` - Detailed problem explanation
- `tools/README.md` - Tool documentation
- Your database is at: `/Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db`
