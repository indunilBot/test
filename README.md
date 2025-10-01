# GPaw Explorer

A Wails desktop client for inspecting PebbleDB blockchain data using a React + TypeScript frontend.

## Getting Started

```bash
# Install JS dependencies (first time only)
cd frontend
yarn install

# Build the frontend assets
yarn build

# Run the desktop app (from repository root)
cd ..
wails dev
```

Use the **New Connection** button in the sidebar to point GPaw Explorer at a PebbleDB directory. Keys appear on the left; the selected value is rendered on the right with JSON pretty-printing when possible.

## Available Commands

```bash
# Development mode with live reload
wails dev

# Production build
wails build

# Frontend only
cd frontend
yarn dev   # Vite dev server
yarn build # Production assets
```

## Project Layout

```
gpaw-explorer/
├── app.go          # PebbleDB bindings exposed to the frontend
├── main.go         # Wails bootstrap and window configuration
├── wails.json      # Wails project configuration
├── build/          # Packaging assets (icons, manifests)
├── frontend/       # React + TypeScript source
│   ├── src/App.tsx
│   ├── src/main.tsx
│   ├── src/style.css
│   └── ...
└── go.mod          # Go module definition
```

## Troubleshooting

- **Database locked** – copy the PebbleDB directory to a temporary location before opening it in the app.
- **Blank window** – ensure `yarn build` has been run so that `frontend/dist` is populated, then restart `wails dev`.
- **Port conflicts** – stop any processes using port `5173` before launching the dev server.

---

The application targets macOS and Windows packaging through Wails using the assets in `build/`. Rebuild the frontend (`yarn build`) before producing release binaries with `wails build`.
