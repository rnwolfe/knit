# knit — brand mark

The `knit` wordmark is set in **Pacifico** — a loopy, low-contrast brush script chosen to read
as a continuous loop of thread (the Threads coil energy), in the brand coral. Font-based on
purpose: no hand-drawn paths to maintain.

## Tokens
| Token | Value |
|---|---|
| Wordmark font | **Pacifico** (Google Fonts / `@fontsource/pacifico`) |
| Coral (thread) | `#FF5A36` |
| Ink (dark bg) | `#0B0B0F` |
| Paper (light bg / docs) | `#F4EFE6` |
| Body / UI font | Space Grotesk |
| Mono / code | JetBrains Mono |

The wordmark is coral on both ink and paper. Don't add stroke, shadow, or a needle — the loopy
script carries the thread metaphor on its own.

## Usage

**Plain HTML (landing):**
```html
<link href="https://fonts.googleapis.com/css2?family=Pacifico&display=swap" rel="stylesheet">
<span style="font-family:'Pacifico';color:#FF5A36">knit</span>
```

**Self-hosted (docs / app — no CDN call):**
```bash
pnpm add @fontsource/pacifico
```
```css
@import '@fontsource/pacifico';
.wordmark { font-family: 'Pacifico'; color: #FF5A36; }
```

Minimum legible size ≈ 18px. For favicon / very small compact marks, prefer a simple `k` glyph
or a single coral loop rather than the full word.

See `wordmark.html` for a standalone preview/lockup.
