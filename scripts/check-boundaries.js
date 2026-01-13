#!/usr/bin/env node
/**
 * Component Boundary Checker
 *
 * Ensures components only import from allowed packages:
 * - backend: can import from shared
 * - frontend: can import from shared
 * - mcp-server: can import from shared
 * - shared: cannot import from other packages
 *
 * Run: node scripts/check-boundaries.js
 */

import { readFileSync, readdirSync, statSync } from 'fs';
import { join, relative } from 'path';
import { fileURLToPath } from 'url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));
const ROOT = join(__dirname, '..');

// Define allowed imports for each package
const ALLOWED_IMPORTS = {
  backend: ['@travel-calendar/shared'],
  frontend: ['@travel-calendar/shared'],
  'mcp-server': ['@travel-calendar/shared'],
  shared: [], // Cannot import from other workspace packages
};

// Forbidden patterns (regex)
const FORBIDDEN_PATTERNS = {
  backend: [
    /from\s+['"].*svelte/,  // No Svelte imports
    /from\s+['"]@travel-calendar\/frontend/,
    /from\s+['"]@travel-calendar\/mcp-server/,
  ],
  frontend: [
    /from\s+['"]@travel-calendar\/backend/,
    /from\s+['"]@travel-calendar\/mcp-server/,
    /from\s+['"]drizzle-orm/,  // No direct DB access
  ],
  'mcp-server': [
    /from\s+['"].*svelte/,
    /from\s+['"]@travel-calendar\/frontend/,
    /from\s+['"]@travel-calendar\/backend/,
    /from\s+['"]drizzle-orm/,
  ],
  shared: [
    /from\s+['"]@travel-calendar\/(backend|frontend|mcp-server)/,
    /^export\s+(function|class|const\s+\w+\s*=)/m,  // No runtime code
  ],
};

let violations = [];

function checkFile(filePath, packageName) {
  const content = readFileSync(filePath, 'utf-8');
  const relativePath = relative(ROOT, filePath);

  const patterns = FORBIDDEN_PATTERNS[packageName] || [];

  for (const pattern of patterns) {
    if (pattern.test(content)) {
      violations.push({
        file: relativePath,
        pattern: pattern.toString(),
        message: `Forbidden pattern found in ${packageName}`,
      });
    }
  }
}

function walkDir(dir, packageName) {
  const files = readdirSync(dir);

  for (const file of files) {
    const filePath = join(dir, file);
    const stat = statSync(filePath);

    if (stat.isDirectory()) {
      if (file === 'node_modules' || file === 'dist' || file === '.svelte-kit') {
        continue;
      }
      walkDir(filePath, packageName);
    } else if (file.endsWith('.ts') || file.endsWith('.js') || file.endsWith('.svelte')) {
      checkFile(filePath, packageName);
    }
  }
}

// Check each package
const packagesDir = join(ROOT, 'packages');

try {
  const packages = readdirSync(packagesDir);

  for (const pkg of packages) {
    const pkgDir = join(packagesDir, pkg);
    const srcDir = join(pkgDir, 'src');

    if (statSync(pkgDir).isDirectory()) {
      try {
        if (statSync(srcDir).isDirectory()) {
          console.log(`Checking ${pkg}...`);
          walkDir(srcDir, pkg);
        }
      } catch (e) {
        // src directory doesn't exist yet, skip
      }
    }
  }
} catch (e) {
  // packages directory doesn't exist yet
  console.log('No packages directory found, skipping boundary check');
  process.exit(0);
}

// Report violations
if (violations.length > 0) {
  console.error('\n❌ Component boundary violations found:\n');

  for (const v of violations) {
    console.error(`  ${v.file}`);
    console.error(`    ${v.message}`);
    console.error(`    Pattern: ${v.pattern}\n`);
  }

  process.exit(1);
} else {
  console.log('\n✓ All component boundaries respected\n');
  process.exit(0);
}
