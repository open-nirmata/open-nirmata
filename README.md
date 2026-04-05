# Open Nirmata

<p align="center">
  <img src="server/static/open-nirmata.png" alt="Open Nirmata logo" width="220" />
</p>

> **Nirmata** (**निर्माता**) is a Sanskrit word meaning **creator** or **maker**.  
> **Open Nirmata** is an open-source project for building AI agents easily.

Open Nirmata brings the core building blocks of agent systems into one place so you can model, connect, and manage them with less friction.

## ✨ What it helps you build

- **LLM provider integrations** for connecting models and backends
- **Tools** that agents can call, including `http`, `mcp`, and `llm` types
- **Knowledge bases** for grounding and contextual retrieval
- **Prompt flows** for multi-step agent orchestration
- **Operational visibility** through health, docs, and metrics endpoints

## 🏗️ Project structure

| Path      | Stack                        | Purpose                                    |
| --------- | ---------------------------- | ------------------------------------------ |
| `server/` | Go + Fiber + MongoDB         | API, orchestration, and persistence        |
| `ui/`     | Next.js + React + TypeScript | Web interface for managing agent resources |

## 🚀 Quick start

### Prerequisites

- **Go** `1.25+`
- **Node.js** and **pnpm**
- A running **MongoDB** instance

### 1) Start the API server

```bash
cd server
cp sample.env .env
go run .
```

By default the API runs at:

- `http://localhost:4050`
- OpenAPI/docs: `http://localhost:4050/docs`
- Metrics: `http://localhost:4050/metrics`

### 2) Start the web UI

```bash
cd ui
pnpm install
pnpm dev
```

Then open:

- `http://localhost:4051`

> The UI talks to `http://localhost:4050` by default. You can override this with `NEXT_PUBLIC_API_BASE_URL` if needed.

## 🧩 Core resources

Open Nirmata currently exposes management endpoints for:

- `GET /health`
- `/knowledgebases`
- `/llm-providers`
- `/tools`
- `/prompt-flows`

## 🛠️ Common development commands

### Backend

```bash
cd server
go run .
```

### Frontend

```bash
cd ui
pnpm dev     # start local development server
pnpm build   # create a production build
pnpm lint    # run lint checks
```

## 🌍 Why Open Nirmata?

Agent development should be more open, inspectable, and composable. Open Nirmata is designed to make it easier to:

1. connect models,
2. attach tools and knowledge,
3. define flows between agent steps, and
4. iterate in a developer-friendly open-source stack.

## 🤝 Contributing

Contributions, ideas, and improvements are welcome. If you want to help shape Open Nirmata, feel free to open an issue or submit a pull request.

## 📄 License

See `LICENSE` for licensing details.
