# Swarm Desktop

Desktop GUI for swarm-cli — DAG creation, pipeline management, and agent monitoring.

## Tech Stack

- **Electron** - Desktop application framework
- **React** - UI framework
- **TypeScript** - Type safety
- **Vite** - Fast bundler for the renderer process
- **Tailwind CSS** - Utility-first CSS

## Development

```bash
# Install dependencies
npm install

# Start Vite dev server (renderer)
npm run dev

# In another terminal, start Electron (connects to Vite dev server)
npm run electron:dev
```

## Building

```bash
# Build both renderer and main process
npm run start

# Package for distribution
npm run package
```

## Project Structure

```
electron/
├── src/
│   ├── main/           # Electron main process
│   │   └── index.ts    # App entry, window management, IPC handlers
│   ├── preload/        # Preload scripts (bridge between main and renderer)
│   │   └── index.ts    # Exposes swarm API to renderer
│   └── renderer/       # React frontend
│       ├── App.tsx     # Main React component
│       ├── main.tsx    # React entry point
│       ├── index.html  # HTML template
│       └── index.css   # Tailwind styles
├── dist/               # Build output
├── package.json
├── vite.config.ts
├── tailwind.config.cjs
└── tsconfig*.json
```

## IPC API

The preload script exposes these methods to the renderer via `window.swarm`:

- `list()` - List all agents
- `run(args)` - Start a new agent
- `kill(agentId)` - Terminate an agent
- `pause(agentId)` - Pause an agent
- `resume(agentId)` - Resume a paused agent
- `logs(agentId)` - Get agent logs
- `inspect(agentId)` - Get agent details
