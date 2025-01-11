// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

import fs from 'fs';
import path from 'path';

const docsPath = './docs/';

// Get all category directories
function getCategories() {
  return fs.readdirSync(docsPath)
    .filter(file => fs.statSync(path.join(docsPath, file)).isDirectory());
}

// Get all markdown files in a directory
function getMdFiles(dir) {
  return fs.readdirSync(dir)
    .filter(file => file.endsWith('.md'))
    .map(file => path.basename(file, '.md'));
}

// Create hierarchical menu with category groups
export function createSidebar() {
  // Get categories
  const categories = getCategories();

  // Create menu structure
  return categories.map(category => ({
    label: category.charAt(0).toUpperCase() + category.slice(1),
    items: getMdFiles(path.join(docsPath, category))
      .map(file => `${category}/${file}`)
  }));
}

// https://astro.build/config
export default defineConfig({
  site: "https://patterns.mkbrechtel.dev",
  trailingSlash: "never",
  build: {
    format: "preserve",
  },
  integrations: [
    starlight({
      title: "Cute Patterns!",
      logo: {
        src: "./public/emoji_u1f4a0.svg",
      },
      favicon: "/emoji_u1f4a0.svg",
      social: {
        github: "https://github.com/mkbrechtel/patterns",
      },
      sidebar: createSidebar(),
      editLink: {
        baseUrl: 'https://github.com/mkbrechtel/patterns/edit/main/',
      },
    }),
  ],
});
