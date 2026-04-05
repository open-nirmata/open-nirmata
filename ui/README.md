# Open Nirmata UI

<p align="center">
  <img src="../server/static/open-nirmata.png" alt="Open Nirmata logo" width="180" />
</p>

This directory contains the **Next.js** admin interface for **Open Nirmata**, the open-source platform for building AI agents easily.

## Local development

```bash
pnpm install
pnpm dev
```

The UI starts on:

- `http://localhost:4051`

By default it connects to the API at:

- `http://localhost:4050`

To override the backend URL, set:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:4050
```

## Available scripts

| Command      | Description                                 |
| ------------ | ------------------------------------------- |
| `pnpm dev`   | Start the development server on port `4051` |
| `pnpm build` | Build the production app                    |
| `pnpm start` | Start the production server on port `4051`  |
| `pnpm lint`  | Run ESLint                                  |

## Notes

- The main project overview lives in the repository root `README.md`.
- This UI is built with **Next.js**, **React**, **TypeScript**, and **TanStack Query**.
