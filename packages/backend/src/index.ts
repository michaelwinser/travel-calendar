/**
 * Travel Calendar - Backend API
 *
 * REST API for managing trips, items, and documents.
 * See ARCHITECTURE.md for patterns and conventions.
 */

import { serve } from '@hono/node-server';
import { Hono } from 'hono';
import { cors } from 'hono/cors';

const app = new Hono();

// Middleware
app.use('*', cors());

// Health check
app.get('/health', (c) => {
  return c.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// API placeholder
app.get('/api', (c) => {
  return c.json({
    name: 'Travel Calendar API',
    version: '0.1.0',
    endpoints: {
      trips: '/api/trips',
      items: '/api/items',
      documents: '/api/documents',
    },
  });
});

// Start server
const port = parseInt(process.env.PORT || '3000');

console.log(`Starting backend on port ${port}...`);

serve({
  fetch: app.fetch,
  port,
}, (info) => {
  console.log(`Backend listening on http://localhost:${info.port}`);
});
