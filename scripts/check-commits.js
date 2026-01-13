#!/usr/bin/env node
/**
 * Commit Message Checker
 *
 * Validates commit messages follow the format:
 *   type(component): description (#issue)
 *
 * Types: feat, fix, refactor, test, docs, chore
 * Components: backend, frontend, mcp-server, shared, infra, cli
 *
 * Run: node scripts/check-commits.js [commit-message-file]
 *
 * In a pre-commit hook, pass .git/COMMIT_EDITMSG as argument
 */

import { readFileSync, existsSync } from 'fs';
import { execSync } from 'child_process';

const VALID_TYPES = ['feat', 'fix', 'refactor', 'test', 'docs', 'chore'];
const VALID_COMPONENTS = ['backend', 'frontend', 'mcp-server', 'shared', 'infra', 'cli'];

// Regex for commit message format
// type(component): description (#issue)
const COMMIT_REGEX = /^(feat|fix|refactor|test|docs|chore)\((backend|frontend|mcp-server|shared|infra|cli)\): .+ \(#\d+\)$/;

// Also allow merge commits and initial commits
const ALLOWED_PATTERNS = [
  /^Merge /,
  /^Initial commit$/,
  /^Revert "/,
];

function validateCommitMessage(message) {
  // Get first line (subject)
  const subject = message.split('\n')[0].trim();

  // Check allowed patterns first
  for (const pattern of ALLOWED_PATTERNS) {
    if (pattern.test(subject)) {
      return { valid: true };
    }
  }

  // Check main format
  if (COMMIT_REGEX.test(subject)) {
    return { valid: true };
  }

  // Provide helpful error
  const errors = [];

  if (!subject.includes('(')) {
    errors.push('Missing component in parentheses');
  } else {
    const match = subject.match(/\(([^)]+)\)/);
    if (match && !VALID_COMPONENTS.includes(match[1])) {
      errors.push(`Invalid component "${match[1]}". Valid: ${VALID_COMPONENTS.join(', ')}`);
    }
  }

  if (!subject.includes(':')) {
    errors.push('Missing colon after component');
  }

  if (!subject.includes('#')) {
    errors.push('Missing issue reference (#issue)');
  }

  const typeMatch = subject.match(/^([a-z]+)/);
  if (typeMatch && !VALID_TYPES.includes(typeMatch[1])) {
    errors.push(`Invalid type "${typeMatch[1]}". Valid: ${VALID_TYPES.join(', ')}`);
  }

  return {
    valid: false,
    subject,
    errors,
  };
}

// Main
const args = process.argv.slice(2);

if (args.length > 0 && existsSync(args[0])) {
  // Called with commit message file (e.g., from commit-msg hook)
  const message = readFileSync(args[0], 'utf-8');
  const result = validateCommitMessage(message);

  if (!result.valid) {
    console.error('\n❌ Invalid commit message format\n');
    console.error(`  Message: "${result.subject}"\n`);
    console.error('  Issues:');
    for (const error of result.errors) {
      console.error(`    - ${error}`);
    }
    console.error('\n  Expected format: type(component): description (#issue)');
    console.error(`  Example: feat(backend): add expense entity (#42)\n`);
    process.exit(1);
  }

  console.log('✓ Commit message format valid');
  process.exit(0);
} else {
  // Called without file, check recent commits on branch
  try {
    const mainBranch = execSync('git rev-parse --abbrev-ref origin/HEAD 2>/dev/null || echo origin/main', { encoding: 'utf-8' }).trim().replace('origin/', '');
    const commits = execSync(`git log ${mainBranch}..HEAD --format="%s" 2>/dev/null`, { encoding: 'utf-8' }).trim();

    if (!commits) {
      console.log('No commits to check');
      process.exit(0);
    }

    let allValid = true;
    const messages = commits.split('\n').filter(Boolean);

    for (const message of messages) {
      const result = validateCommitMessage(message);
      if (!result.valid) {
        console.error(`❌ ${message}`);
        for (const error of result.errors) {
          console.error(`   - ${error}`);
        }
        allValid = false;
      } else {
        console.log(`✓ ${message}`);
      }
    }

    process.exit(allValid ? 0 : 1);
  } catch (e) {
    // Not in a git repo or no commits
    console.log('No commits to check');
    process.exit(0);
  }
}
