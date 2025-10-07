# GPaw Explorer

A modern desktop application for exploring and inspecting PebbleDB blockchain data. Built with **Go** and **React**, powered by **Wails** framework for native desktop experience.

![GPaw Explorer](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Wails](https://img.shields.io/badge/Wails-v2.10.2-FF6B6B?style=flat)
![React](https://img.shields.io/badge/React-18.2-61DAFB?style=flat&logo=react)
![TypeScript](https://img.shields.io/badge/TypeScript-4.6-3178C6?style=flat&logo=typescript)

## ✨ Features

- 🔍 **Search & Browse** - Search by block number or transaction hash
- 🌳 **Tree View** - Organize keys by prefix for easy navigation
- 📊 **Database Stats** - View total keys, size, and performance metrics
- 💾 **Export Data** - Export values to files
- 🎨 **Multiple Formats** - View data as JSON, String, Hex, or Base64
- 🔄 **Live Pagination** - Load large datasets efficiently
- 🎯 **Multiple Connections** - Manage multiple PebbleDB instances
- 🖥️ **Cross-Platform** - Works on macOS, Windows, and Linux

## 🛠️ Technology Stack

### Backend
- **Go 1.24** - Programming language
- **Wails v2.10.2** - Framework for building desktop apps with Go and web technologies
- **PebbleDB v1.1.5** - Embedded key-value store
- **golang.org/x/sync** - Concurrency utilities

### Frontend
- **React 18.2** - UI library
- **TypeScript 4.6** - Type-safe JavaScript
- **Vite 3.0** - Fast build tool and dev server
- **Tailwind CSS 3.4** - Utility-first CSS framework
- **Lucide React** - Icon library
- **Radix UI** - Headless UI components
- **shadcn/ui** - Re-usable component library

### Key Dependencies

**Frontend Libraries:**
```json
{
  "@radix-ui/react-select": "^2.2.6",
  "class-variance-authority": "^0.7.1",
  "clsx": "^2.1.1",
  "lucide-react": "^0.544.0",
  "react": "^18.2.0",
  "react-dom": "^18.2.0",
  "tailwind-merge": "^3.3.1"
}
```

**Backend Dependencies:**
```
github.com/cockroachdb/pebble v1.1.5
github.com/wailsapp/wails/v2 v2.10.2
golang.org/x/sync v0.11.0
```

## 📋 Prerequisites

Before installing GPaw Explorer, ensure you have the following installed:

### Common Requirements (All Platforms)
- **Go 1.24+** - [Download Go](https://golang.org/dl/)
- **Node.js 16+** - [Download Node.js](https://nodejs.org/)
- **npm or Yarn** - Package manager (comes with Node.js)

### Platform-Specific Requirements

#### macOS
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Windows
```powershell
# Install WebView2 (usually pre-installed on Windows 11)
# Download from: https://developer.microsoft.com/en-us/microsoft-edge/webview2/

# Install GCC (via MSYS2 or TDM-GCC)
# Download MSYS2: https://www.msys2.org/
# After installing MSYS2, run:
pacman -S mingw-w64-x86_64-gcc

# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Linux (Debian/Ubuntu)
```bash
# Install required dependencies
sudo apt update
sudo apt install -y build-essential libgtk-3-dev libwebkit2gtk-4.0-dev

# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Linux (Fedora/Red Hat)
```bash
# Install required dependencies
sudo dnf install -y gcc-c++ gtk3-devel webkit2gtk4.0-devel

# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### Linux (Arch)
```bash
# Install required dependencies
sudo pacman -S webkit2gtk gtk3

# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 🚀 Installation

### 1. Clone the Repository
```bash
git clone https://github.com/yourusername/gpaw-explorer.git
cd gpaw-explorer
```

### 2. Install Go Dependencies
```bash
go mod download
```

### 3. Install Frontend Dependencies
```bash
cd frontend
npm install
# or
yarn install
```

### 4. Build Frontend Assets
```bash
npm run build
# or
yarn build
cd ..
```

## 🏃 Running the Application

### Development Mode (with live reload)
```bash
# From the project root
wails dev
```

This will:
- Start the Vite dev server on `http://127.0.0.1:5173`
- Launch the desktop application
- Enable hot reload for frontend changes
- Auto-rebuild for backend changes

### Production Build

#### Build for Your Platform
```bash
wails build
```

The compiled application will be in the `build/bin` directory:
- **macOS**: `gpaw-explorer.app`
- **Windows**: `gpaw-explorer.exe`
- **Linux**: `gpaw-explorer`

#### Build for Specific Platform
```bash
# Build for macOS (from any platform)
wails build -platform darwin/amd64
wails build -platform darwin/arm64

# Build for Windows
wails build -platform windows/amd64

# Build for Linux
wails build -platform linux/amd64
```

## 📁 Project Structure

```
gpaw-explorer/
├── app.go                    # Backend logic (PebbleDB operations)
├── app_test.go              # Backend tests
├── main.go                  # Application entry point
├── go.mod                   # Go dependencies
├── go.sum                   # Go dependency checksums
├── wails.json               # Wails configuration
│
├── build/                   # Build assets and outputs
│   ├── bin/                # Compiled binaries
│   ├── darwin/             # macOS icons and config
│   ├── windows/            # Windows icons and manifest
│   └── linux/              # Linux desktop file
│
├── frontend/               # React frontend
│   ├── src/
│   │   ├── App.tsx        # Main application component
│   │   ├── main.tsx       # React entry point
│   │   ├── style.css      # Global styles
│   │   ├── components/    # UI components
│   │   │   └── ui/        # shadcn/ui components
│   │   │       └── select.tsx
│   │   └── lib/
│   │       └── utils.ts   # Utility functions
│   ├── wailsjs/           # Generated Wails bindings
│   ├── dist/              # Built frontend assets
│   ├── package.json       # Frontend dependencies
│   ├── tsconfig.json      # TypeScript configuration
│   ├── vite.config.ts     # Vite configuration
│   └── tailwind.config.js # Tailwind CSS config
│
└── README.md              # This file
```

## 🎯 Usage

### 1. Add a Database Connection
1. Click the **"New Connection"** button in the sidebar
2. Enter a connection name (e.g., `slot-db`)
3. Paste or browse to your PebbleDB directory path
4. Click **"Connect"**

Example path:
```
/Users/user/Desktop/blockchain/pebbledb/slot-db
```

### 2. Browse Keys
- Switch between **Tree View** (organized by prefix) or **List View**
- Expand prefixes to see individual keys
- Click any key to view its value

### 3. Search Data
- Select search type: **Block Number** or **TX Hash**
- Enter your search term
- Click **Search** or press Enter

### 4. View Values
- View data in multiple formats:
  - **Auto** - Smart detection (JSON, String, or Hex)
  - **String** - UTF-8 text
  - **Hex** - Hexadecimal representation
  - **Base64** - Base64 encoded

### 5. Export Data
- Select a key
- Click **Export** button
- Choose save location

## 🐛 Troubleshooting

### Database Locked Error
PebbleDB allows only one process to access the database at a time. If you encounter a "database locked" error:
```bash
# Copy the database to a temporary location
cp -r /path/to/original/db /tmp/db-copy
# Then open /tmp/db-copy in GPaw Explorer
```

### Blank Window on Startup
Ensure frontend assets are built:
```bash
cd frontend
npm run build
cd ..
wails dev
```

### Port 5173 Already in Use
Kill the process using port 5173:
```bash
# macOS/Linux
lsof -ti:5173 | xargs kill -9

# Windows
netstat -ano | findstr :5173
taskkill /PID <PID> /F
```

### Wails Command Not Found
Add Go bin directory to your PATH:
```bash
# Add to ~/.bashrc, ~/.zshrc, or equivalent
export PATH=$PATH:$(go env GOPATH)/bin
```

### Linux: Missing Dependencies
If you see errors about missing libraries:
```bash
# Ubuntu/Debian
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev

# Fedora
sudo dnf install gtk3-devel webkit2gtk4.0-devel
```

## 🧪 Development

### Run Frontend Only (for UI development)
```bash
cd frontend
npm run dev
```
Access at `http://127.0.0.1:5173`

### Run Tests
```bash
# Backend tests
go test ./...

# Frontend tests (if configured)
cd frontend
npm test
```

### Build Frontend Production Assets
```bash
cd frontend
npm run build
```

### Format Code
```bash
# Go
go fmt ./...

# Frontend
cd frontend
npm run format  # If configured
```

## 📦 Building for Distribution

### Create Production Builds for All Platforms

```bash
# macOS (Intel)
wails build -platform darwin/amd64

# macOS (Apple Silicon)
wails build -platform darwin/arm64

# Windows
wails build -platform windows/amd64

# Linux
wails build -platform linux/amd64
```

Binaries will be in `build/bin/`

## 🤝 Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Wails](https://wails.io/) - Amazing Go + Web framework
- [PebbleDB](https://github.com/cockroachdb/pebble) - Fast key-value store
- [Radix UI](https://www.radix-ui.com/) - Accessible component primitives
- [shadcn/ui](https://ui.shadcn.com/) - Beautiful component library
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework

## 📞 Support

If you encounter any issues or have questions:
- Open an issue on GitHub
- Check the [Wails documentation](https://wails.io/docs/introduction)
- Review the troubleshooting section above

---

Built with ❤️ using Go, React, and Wails
