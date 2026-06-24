import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';
import { z } from 'astro:content';

// Extend the docs schema with freshness-triage fields (optional, so builds never break;
// populate them on pages so the docs-freshness workflow can flag stale/unowned pages).
export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		schema: docsSchema({
			extend: z.object({
				owner: z.string().optional(),
				// YAML auto-parses bare dates to Date; coerce to a string for stable rendering.
				lastReviewed: z.coerce.string().optional(),
			}),
		}),
	}),
};
