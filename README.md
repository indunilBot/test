# GPaw Explorer - PebbleDB Blockchain Explorer

A desktop application to explore and visualize PebbleDB blockchain data built with Wails, React, and TypeScript.

## 🚀 How to Run This Project

### Quick Start (Easiest Method)

```bash
cd /Users/user/Desktop/indunil/paw/Go/GPaw-explorer/gpaw-explorer
./quick-start.sh
```

This will automatically:
- Copy the database to avoid locking issues
- Start the Wails dev server with Yarn
- Open the application window

### Manual Start

```bash
# Navigate to project
cd /Users/user/Desktop/indunil/paw/Go/GPaw-explorer/gpaw-explorer

# Start development server
wails dev
```

**That's it!** The app window will open automatically.

## 📖 Using the App

### Connect to Your Database

1. **Click "New Connection"** (top left button)
2. **Enter name**: `slot-db` (or any name)
3. **Select directory**:
   ```bash
   # If database is locked, copy it first:
   cp -r /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db /tmp/my-db

   # Then select: /tmp/my-db
   ```
4. **Browse your blockchain data!**

### UI Features

- **Left**: Database list
- **Middle**: Key browser (Tree/List view)
- **Right**: JSON value viewer
- **Search**: Filter keys
- **Refresh**: Reload data

## 🛠️ Commands

```bash
# Development (hot reload)
wails dev

# Build for production
wails build

# Frontend only
cd frontend
yarn dev        # Dev server
yarn build      # Production build

# Command-line tools
cd tools/simple_reader
go run . /tmp/my-db    # View DB in terminal
```

## 📁 Project Structure

```
gpaw-explorer/
├── app.go                    # Backend (Go)
├── main.go                   # Entry point
├── wails.json                # Config (uses Yarn)
├── frontend/                 # Frontend (React + TypeScript)
│   ├── src/App.tsx          # Main UI
│   ├── style.css            # Tailwind CSS
│   ├── package.json
│   └── yarn.lock            # Yarn lockfile
├── tools/                    # CLI tools
│   ├── read_db/
│   └── simple_reader/
├── quick-start.sh            # Quick launcher
└── README.md                 # This file
```

## 🎨 Tech Stack

- **Backend**: Go + Wails v2
- **Frontend**: React 18 + TypeScript + Tailwind CSS
- **Build**: Vite
- **Database**: PebbleDB
- **Package Manager**: Yarn

## 🐛 Troubleshooting

### Database Locked
```bash
# Copy database first
cp -r /path/to/slot-db /tmp/my-db
```

### Blank Page
```bash
cd frontend
yarn install
```

### Port In Use
```bash
pkill -f "wails dev"
```

## 📚 More Documentation

- [QUICK_START.md](QUICK_START.md) - Quick reference
- [YARN_SETUP.md](YARN_SETUP.md) - Yarn setup details
- [SOLUTION.md](SOLUTION.md) - Detailed solutions
- [tools/README.md](tools/README.md) - CLI tools guide

## 💡 Quick Tips

**Running right now?** → Your app is at http://localhost:34115

**Want to see data fast?** → `./quick-start.sh`

**Prefer terminal?** → `cd tools/simple_reader && go run . /tmp/my-db`

**Need to build?** → `wails build`

---

**The app is currently running!** 🎉

Check the terminal output - you should see:
```
To develop in the browser and call your bound Go methods from Javascript, navigate to: http://localhost:34115
```

The desktop window should already be open. If not, visit http://localhost:34115 in your browser.
