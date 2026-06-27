# isobox Landing Page

This is the landing page and developer site for **isobox** (`PeeeBrain/isobox`), built using Vite, React, TypeScript, and Tailwind CSS.

## Getting Started

To run the landing page development server locally:

```sh
cd site
bun install
bun run dev
```

The site will be available locally at `http://localhost:5173`.

## Production Build

To build the static application assets:

```sh
bun run build
```

This compiles TypeScript and outputs the production bundle to the `site/dist/` directory.

## Cloudflare Pages Deployment

This project is optimized for deployment to **Cloudflare Pages**. Use the following configurations in your Cloudflare dashboard:

- **Framework Preset**: `Vite`
- **Root Directory**: `site`
- **Build Command**: `bun run build`
- **Build Output Directory**: `dist`
- **Install Command**: `bun install`
- **Build System Version**: Cloudflare Pages modern builder environment (ensuring Bun runtime is supported, or install via environment setup).

No GitHub Pages routing or sub-path prefix configuration is required; the application relies on standard root-level resolution (`base: '/'`).
