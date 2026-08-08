# Crown & Borough Frontend

The frontend is a Vite, React, and TypeScript application for the Crown & Borough
map interface. It uses Tailwind CSS v4, shadcn/ui, and interactive SVG rendering.

## Commands

Run these commands from `web/`:

- `npm run dev` starts the Vite development server.
- `npm run build` runs the strict TypeScript build and production bundle.
- `npm run lint` runs ESLint.
- `npm run format` formats the frontend with Prettier.
- `npm run format:check` checks Prettier formatting without changing files.

The root `make web-dev` target starts the same development server from the
repository root.

## Structure

- `src/components/MapViewer.tsx` renders generic map and state data as an
  interactive SVG with terrain, live markers, control overlays, and tooltips.
- On the map, a left click selects or deselects a territory. Holding the left
  button and dragging past the movement threshold pans the view without
  selecting a territory.
- `src/components/ui/` contains the shadcn/ui components used by the application.
- `src/App.tsx` loads the static map and live state from `/api/map` and
  `/api/state`.
- `src/fixtures/` contains static data used by component tests.
- `src/types.ts` defines the map and state contracts shared by the frontend.

The Vite development server proxies `/api` to `http://localhost:8080`.
