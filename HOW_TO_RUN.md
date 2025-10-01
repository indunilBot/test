# 🚀 How to Run GPaw Explorer

## The Simplest Way

```bash
cd /Users/user/Desktop/indunil/paw/Go/GPaw-explorer/gpaw-explorer
./quick-start.sh
```

**Done!** The app window opens automatically.

---

## Manual Way

```bash
cd /Users/user/Desktop/indunil/paw/Go/GPaw-explorer/gpaw-explorer
wails dev
```

**Done!** The app window opens automatically.

---

## Using the App

### Step 1: Copy Database (if locked)
```bash
cp -r /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db /tmp/my-db
```

### Step 2: In the App Window
1. Click **"New Connection"**
2. Name: `slot-db`
3. Select: `/tmp/my-db`
4. **Done!** Browse your data

---

## Your App is Already Running! 🎉

Check your terminal - you should see:
```
To develop in the browser... navigate to: http://localhost:34115
```

**Desktop window open?** ✅ Start using it!

**No window?** Visit: http://localhost:34115

---

## Common Commands

```bash
# Run the app
wails dev

# Build for production
wails build

# Stop the app
# Just close the window or Ctrl+C in terminal
```

That's it! Simple as that. 🎯
