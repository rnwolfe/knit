// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import { visit } from 'unist-util-visit';

// GitHub Pages project site lives at a sub-path: https://rnwolfe.github.io/knit
const SITE = 'https://rnwolfe.github.io';
const BASE = '/knit';

// Base-prefix hand-written absolute Markdown links so they resolve under /knit/.
// (Starlight base-prefixes nav/assets, but not in-content `/...` links.)
function rehypeBaseLinks() {
  return (tree) =>
    visit(tree, 'element', (n) => {
      const h = n.tagName === 'a' && n.properties && n.properties.href;
      if (
        typeof h === 'string' &&
        h.startsWith('/') &&
        !h.startsWith('//') &&
        !h.startsWith(BASE + '/') &&
        h !== BASE
      ) {
        n.properties.href = BASE + h;
      }
    });
}

export default defineConfig({
  site: SITE,
  base: BASE,
  markdown: { rehypePlugins: [rehypeBaseLinks] },
  integrations: [
    starlight({
      title: 'knit',
      tagline: "An agent-safe CLI for Instagram's Threads",
      description:
        "knit is an agent-friendly CLI for Instagram's Threads — read-only by default, posting gated in the binary, prompt-injection-fenced, machine-readable.",
      logo: { src: './src/assets/knit.svg', replacesTitle: true },
      social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/rnwolfe/knit' }],
      editLink: { baseUrl: 'https://github.com/rnwolfe/knit/edit/main/docs/' },
      lastUpdated: true,
      customCss: ['./src/styles/custom.css'],
      plugins: [
        starlightLlmsTxt({
          projectName: 'knit',
          description:
            "Agent-friendly CLI for Instagram's Threads (official API). Read-only by default, mutation-gated in the binary, prompt-injection-fenced, machine-readable.",
          exclude: ['changelog'],
        }),
      ],
      // Diátaxis: Tutorials / How-to / Reference / Explanation, physically separate.
      sidebar: [
        { label: 'Start here', items: [{ label: 'Quickstart', slug: 'start/quickstart' }] },
        { label: 'Tutorials', items: [{ autogenerate: { directory: 'tutorials' } }] },
        { label: 'How-to guides', items: [{ autogenerate: { directory: 'guides' } }] },
        { label: 'Reference', items: [{ autogenerate: { directory: 'reference' } }] },
        { label: 'Explanation', items: [{ autogenerate: { directory: 'explanation' } }] },
      ],
    }),
  ],
});
