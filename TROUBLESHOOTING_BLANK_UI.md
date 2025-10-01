# Troubleshooting Blank UI

## Issue: Running `wails dev` but seeing blank/white page

### Quick Fixes

#### 1. Hard Refresh the Browser
```bash
# In the app window or browser, press:
Cmd + Shift + R  (Mac)
Ctrl + Shift + R (Windows/Linux)
```

#### 2. Check if React is Running
Visit: http://localhost:5173 (Vite dev server)

**See content?** → Frontend is working, issue is with Wails integration
**See blank?** → Frontend has an error

#### 3. Check Browser Console
1. Open app window or visit http://localhost:34115
2. Press `F12` or `Cmd+Option+I`
3. Look for red errors in Console tab

Common errors:
- `Failed to fetch` → Backend not connected
- `Cannot find module` → Missing dependency
- White screen + no errors → CSS issue

#### 4. Verify Tailwind CSS
```bash
cd frontend
yarn install
```

Check that `frontend/node_modules/tailwindcss` exists.

#### 5. Restart Everything
```bash
# Kill all processes
pkill -f "wails dev"
pkill -f "vite"

# Clean restart
cd /Users/user/Desktop/indunil/paw/Go/GPaw-explorer/gpaw-explorer
rm -rf frontend/node_modules
cd frontend && yarn install
cd ..
wails dev
```

#### 6. Test with Simple Component
The page should show white text "Test - Can you see this?" with a blue button.

If you see this, React is working!

### Diagnostic Steps

1. **Check Wails is running:**
   ```
   Look for: "To develop in the browser... navigate to: http://localhost:34115"
   ```

2. **Check Vite is running:**
   ```
   Look for: "VITE v3.2.11  ready in XXXms"
   ```

3. **Check for compilation errors:**
   ```
   Look for: "ERROR" in red text
   ```

4. **Check browser console:**
   - Open DevTools (F12)
   - Look at Console tab
   - Look at Network tab (any failed requests?)

### Common Causes

| Symptom | Cause | Solution |
|---------|-------|----------|
| Completely blank white page | CSS not loaded | Hard refresh (Cmd+Shift+R) |
| "Loading..." forever | Backend not connected | Check wails dev is running |
| Dark page, no content | Tailwind not generating CSS | `cd frontend && yarn install` |
| Errors in console | Missing dependency | `cd frontend && yarn install` |
| Old content showing | Cache issue | Hard refresh or clear cache |

### Still Not Working?

1. **Check the actual URLs:**
   - Desktop app: Uses http://localhost:34115
   - Vite direct: Uses http://localhost:5173

2. **Try browser directly:**
   ```bash
   # Open in your browser
   open http://localhost:34115
   ```

3. **Check if ports are in use:**
   ```bash
   lsof -i :5173
   lsof -i :34115
   ```

4. **Full clean restart:**
   ```bash
   # Kill everything
   pkill -f wails
   pkill -f vite
   pkill -f node

   # Clean install
   cd frontend
   rm -rf node_modules yarn.lock
   yarn install

   # Restart
   cd ..
   wails dev
   ```

### What Should You See?

When working correctly, you should see:

- **Left sidebar**: Dark blue (#19334D) background
- **"GPaw Explorer"** title with database icon
- **"New Connection"** button (blue)
- **"DATABASES"** label
- Message: "No databases connected. Add a new connection."

If you see this, the UI is working! Click "New Connection" to connect to your database.

### Debug Mode

A ready-made Tailwind test view lives at `frontend/src/TestApp.tsx`. To quickly confirm that CSS is loading, temporarily edit `frontend/src/main.tsx` to render it instead of the main app:

```tsx
import TestApp from './TestApp'

root.render(
    <React.StrictMode>
        <TestApp/>
    </React.StrictMode>
)
```

Save and confirm you see a **dark blue screen** with the text "Test - Can you see this?" and a bright blue "Tailwind Button".

**YES** → React + Tailwind are working; investigate the main app component or data connection.
**NO** → The frontend build or Tailwind pipeline is broken; reinstall dependencies or restart Vite/Wails.

---

## Current Status

✅ Tailwind CSS installed (v3.4.0)
✅ Dependencies installed with Yarn
✅ Style.css updated with proper root styling
✅ Test component created for debugging

The app **should** be showing the UI now. If not, follow the steps above!
