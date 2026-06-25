// Generate the GitHub repo social-preview card (1280×640) for knit — a proof-forward card
// showing the in-binary posting gate (WRITE_REFUSED), in the "thread" brand.
// Output: .github/social-preview.png (uploaded manually in repo Settings → Social preview).
// Run from the docs dir (has @fontsource fonts + sharp): node scripts/gen-social.mjs
import { execFileSync } from 'node:child_process';
import { readFileSync, mkdirSync, writeFileSync, globSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const DOCS = resolve(dirname(fileURLToPath(import.meta.url)), '..'); // docs
const REPO = resolve(DOCS, '..');
const OUT = join(REPO, '.github', 'social-preview.png');
const CHROME = process.env.CHROME || '/usr/bin/google-chrome';
const W = 1280, H = 640;

function fontB64(family) {
  const [file] = globSync(`node_modules/@fontsource-variable/${family}/files/*-latin-wght-normal.woff2`, { cwd: DOCS });
  if (!file) throw new Error(`font not found: ${family}`);
  return readFileSync(join(DOCS, file)).toString('base64');
}
const grotesk = fontB64('space-grotesk');
const mono = fontB64('jetbrains-mono');

const html = `<!doctype html><html><head><meta charset="utf-8"><style>
@font-face{font-family:'Grot';src:url(data:font/woff2;base64,${grotesk}) format('woff2');font-weight:300 700}
@font-face{font-family:'Mono';src:url(data:font/woff2;base64,${mono}) format('woff2');font-weight:100 800}
*{margin:0;box-sizing:border-box}
html,body{width:${W}px;height:${H}px}
body{background:#0e0e16;color:#f4efe6;font-family:'Grot',sans-serif;position:relative;overflow:hidden}
.bg{position:absolute;inset:0;background:
  radial-gradient(900px 520px at 8% -12%, rgba(255,90,54,.26), transparent 60%),
  radial-gradient(760px 520px at 112% 122%, rgba(255,90,54,.16), transparent 55%),
  radial-gradient(620px 460px at 95% 6%, rgba(120,40,20,.20), transparent 60%)}
.frame{position:absolute;inset:22px;border:1px solid rgba(255,90,54,.18);border-radius:18px}
.wrap{position:absolute;inset:0;padding:52px 60px;display:flex;flex-direction:column;justify-content:space-between}
.top{display:flex;align-items:center;justify-content:space-between}
.brand{display:flex;align-items:center;gap:14px;font-family:'Mono';font-weight:600;font-size:25px}
.dot{width:15px;height:15px;border-radius:50%;background:#ff5a36;box-shadow:0 0 22px 4px rgba(255,90,54,.7)}
.meta{font-family:'Mono';color:#8a8676;letter-spacing:.12em;text-transform:uppercase;font-size:19px}
.title{font-weight:700;font-size:64px;line-height:1.04;letter-spacing:-.02em;max-width:1100px}
.title .ac{color:#ff5a36}
.term{background:#08080d;border:1px solid rgba(255,90,54,.18);border-radius:14px;padding:22px 26px;font-family:'Mono';font-size:23px;line-height:1.5}
.dots{display:flex;gap:8px;margin-bottom:14px}
.dots i{width:13px;height:13px;border-radius:50%;display:inline-block}
.cmd{color:#f4efe6}.cmd .p{color:#ff5a36}.k{color:#ffb59e}.s{color:#cdd6a0}.muted{color:#8a8676}.refuse{color:#ff5a36;font-weight:700}
.bottom{display:flex;align-items:center;justify-content:space-between;gap:24px;font-family:'Mono';font-size:21px}
.tags{display:flex;gap:10px;flex-shrink:0}
.tag{border:1px solid rgba(255,90,54,.32);color:#ffc7b6;border-radius:999px;padding:6px 13px;font-size:18px;white-space:nowrap}
.install{color:#ff5a36;white-space:nowrap}
</style></head><body>
<div class="bg"></div><div class="frame"></div>
<div class="wrap">
  <div class="top">
    <div class="brand"><span class="dot"></span>knit</div>
    <div class="meta">Threads · official API</div>
  </div>
  <div class="title">The <span class="ac">agent-safe</span> CLI for Threads.</div>
  <div class="term">
    <div class="dots"><i style="background:#ff5f56"></i><i style="background:#ffbd2e"></i><i style="background:#27c93f"></i></div>
    <div class="cmd"><span class="p">$</span> knit post "shipped a thing"</div>
    <div class="cmd"><span class="muted">{</span> <span class="k">"error"</span>: <span class="s">"posting is a write"</span>, <span class="k">"code"</span>: <span class="refuse">"WRITE_REFUSED"</span>,</div>
    <div class="cmd">  <span class="k">"remediation"</span>: <span class="s">"re-run with --allow-mutations"</span> <span class="muted">}</span></div>
  </div>
  <div class="bottom">
    <div class="tags"><span class="tag">read-only</span><span class="tag">gated in binary</span><span class="tag">injection-fenced</span><span class="tag">MIT</span></div>
    <div class="install">$ brew install rnwolfe/tap/knit</div>
  </div>
</div></body></html>`;

mkdirSync(dirname(OUT), { recursive: true });
const tmp = join(tmpdir(), 'knit-social.html');
const raw = join(tmpdir(), 'knit-social-raw.png');
writeFileSync(tmp, html);
// This headless Chrome paints ~half the window height → render at 2× and crop the top.
execFileSync(CHROME, [
  '--headless=new', '--no-sandbox', '--hide-scrollbars', '--force-device-scale-factor=1',
  `--window-size=${W},${H * 2}`, '--default-background-color=00000000',
  '--virtual-time-budget=1500', `--screenshot=${raw}`, `file://${tmp}`,
], { stdio: 'ignore' });
await sharp(raw).extract({ left: 0, top: 0, width: W, height: H }).toFile(OUT);
console.log('wrote', OUT);
