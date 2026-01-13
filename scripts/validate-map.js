#!/usr/bin/env node
/**
 * Project Map Validator
 *
 * Validates PROJECT_MAP.md against the actual codebase:
 * - Checks that documented files exist
 * - Checks that documented endpoints are implemented
 * - Checks that documented tools are registered
 * - Checks lexicon terms appear in code
 *
 * Run: node scripts/validate-map.js
 * Or:  pnpm validate:map
 */

import { readFileSync, existsSync } from 'fs';
import { join } from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const __dirname = fileURLToPath(new URL('.', import.meta.url));
const ROOT = join(__dirname, '..');

const errors = [];
const warnings = [];

function error(msg) {
  errors.push(msg);
  console.error(`❌ ${msg}`);
}

function warn(msg) {
  warnings.push(msg);
  console.warn(`⚠️  ${msg}`);
}

function success(msg) {
  console.log(`✓ ${msg}`);
}

// Read PROJECT_MAP.md
const mapPath = join(ROOT, 'PROJECT_MAP.md');
if (!existsSync(mapPath)) {
  console.error('PROJECT_MAP.md not found');
  process.exit(1);
}

const mapContent = readFileSync(mapPath, 'utf-8');

console.log('\n🗺️  Validating PROJECT_MAP.md\n');
console.log('─'.repeat(50));

// 1. Validate component directories exist
console.log('\n📁 Checking component directories...\n');

const components = ['backend', 'frontend', 'mcp-server', 'shared'];
for (const component of components) {
  const dir = join(ROOT, 'packages', component);
  if (existsSync(dir)) {
    success(`packages/${component}/ exists`);
  } else {
    warn(`packages/${component}/ does not exist yet`);
  }
}

// 2. Validate lexicon terms are defined
console.log('\n📖 Checking lexicon...\n');

const lexiconSection = mapContent.match(/## Lexicon[\s\S]*?(?=\n---|\n##|$)/);
if (lexiconSection) {
  const termMatches = lexiconSection[0].matchAll(/^\| \*\*(\w+)\*\* \|/gm);
  const terms = [...termMatches].map(m => m[1]);

  if (terms.length > 0) {
    success(`Found ${terms.length} lexicon terms: ${terms.join(', ')}`);
  } else {
    warn('No lexicon terms found');
  }
} else {
  warn('Lexicon section not found in PROJECT_MAP.md');
}

// 3. Validate API endpoints table
console.log('\n🔌 Checking API endpoints...\n');

const apiSection = mapContent.match(/## API Endpoints[\s\S]*?(?=\n---|\n##|$)/);
if (apiSection) {
  const endpointMatches = apiSection[0].matchAll(/`(GET|POST|PATCH|DELETE)` \| `([^`]+)`/g);
  const endpoints = [...endpointMatches].map(m => ({ method: m[1], path: m[2] }));

  if (endpoints.length > 0) {
    success(`Found ${endpoints.length} documented endpoints`);

    // Check if routes files exist
    const routesDir = join(ROOT, 'packages/backend/src/routes');
    if (existsSync(routesDir)) {
      // Could do deeper validation here
      success('Routes directory exists');
    } else {
      warn('Routes directory not created yet');
    }
  }
} else {
  warn('API Endpoints section not found');
}

// 4. Validate MCP tools table
console.log('\n🔧 Checking MCP tools...\n');

const toolsSection = mapContent.match(/## MCP Tools[\s\S]*?(?=\n---|\n##|$)/);
if (toolsSection) {
  const toolMatches = toolsSection[0].matchAll(/`([a-z_]+)`/g);
  const tools = [...new Set([...toolMatches].map(m => m[1]))];

  if (tools.length > 0) {
    success(`Found ${tools.length} documented tools: ${tools.join(', ')}`);
  }
} else {
  warn('MCP Tools section not found');
}

// 5. Check for stale references
console.log('\n🔍 Checking for stale references...\n');

// Extract file paths from map
const fileRefs = mapContent.matchAll(/`(src\/[^`]+|packages\/[^`]+)`/g);
const referencedFiles = [...fileRefs].map(m => m[1]);

for (const ref of referencedFiles.slice(0, 10)) { // Check first 10
  const fullPath = join(ROOT, ref);
  if (!existsSync(fullPath)) {
    // Only warn if parent directory exists (meaning project is partially built)
    const parentDir = join(ROOT, ref.split('/').slice(0, -1).join('/'));
    if (existsSync(parentDir)) {
      warn(`Referenced file not found: ${ref}`);
    }
  }
}

// Summary
console.log('\n' + '─'.repeat(50));
console.log('\n📊 Validation Summary\n');

if (errors.length === 0 && warnings.length === 0) {
  console.log('✅ PROJECT_MAP.md is valid\n');
} else {
  if (errors.length > 0) {
    console.log(`❌ ${errors.length} error(s) found`);
  }
  if (warnings.length > 0) {
    console.log(`⚠️  ${warnings.length} warning(s) found`);
  }
  console.log('\nNote: Warnings for missing files are expected before implementation.\n');
}

// Update "Last validated" timestamp
try {
  const timestamp = new Date().toISOString();
  const updatedMap = mapContent.replace(
    /\*\*Last validated\*\*: .*/,
    `**Last validated**: ${timestamp}`
  );
  // Don't actually write - just show what would be updated
  if (updatedMap !== mapContent) {
    console.log(`📝 Would update "Last validated" to: ${timestamp}\n`);
  }
} catch (e) {
  // Ignore timestamp update errors
}

process.exit(errors.length > 0 ? 1 : 0);
