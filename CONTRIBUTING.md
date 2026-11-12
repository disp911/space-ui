## Local Development Setup

- Create a directory named `space-ui` in the project root
- Rename `.env.example` to `.env`
- Build the frontend once: `npm ci --prefix frontend && npm --prefix frontend run build`
- Run `main.go`

While working on the frontend, run the Vite dev server against the running
panel instead of rebuilding: `PANEL_URL=http://127.0.0.1:2053 npm --prefix frontend run dev`.
