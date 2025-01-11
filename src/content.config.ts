import { defineCollection } from 'astro:content';
import { glob, file, type Loader } from 'astro/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

const docs = defineCollection({
  loader: glob({ pattern: "**/*.md", base: "./docs" }),
  schema: docsSchema()
});

export const collections = { docs };
