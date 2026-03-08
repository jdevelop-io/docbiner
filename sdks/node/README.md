# @docbiner/sdk

Official Node.js/TypeScript SDK for the [Docbiner](https://docbiner.com) HTML-to-PDF/Images API.

## Installation

```bash
npm install @docbiner/sdk
```

## Quick start

```typescript
import { Docbiner } from '@docbiner/sdk';

const docbiner = new Docbiner({ apiKey: 'db_live_...' });

// Convert HTML to PDF (synchronous)
const pdf = await docbiner.convert({
  source: '<h1>Hello World</h1>',
  format: 'pdf',
  options: {
    page_size: 'A4',
    margin_top: '20mm',
    margin_bottom: '20mm',
  },
});
// pdf is a Buffer — write to file, stream, etc.

// Convert a URL to PNG screenshot
const screenshot = await docbiner.convert({
  source: 'https://example.com',
  format: 'png',
  options: { full_page: true },
});

// Async conversion with webhook delivery
const job = await docbiner.convertAsync({
  source: '<h1>Hello</h1>',
  delivery: {
    method: 'webhook',
    config: { url: 'https://myapp.com/hooks/docbiner' },
  },
});
console.log(job.id, job.status); // => "pending"
```

## Jobs

```typescript
// List jobs
const { data, pagination } = await docbiner.jobs.list({ page: 1, per_page: 10 });

// Get a single job
const job = await docbiner.jobs.get('job-id');

// Download result
const file = await docbiner.jobs.download('job-id');

// Delete a job
await docbiner.jobs.delete('job-id');
```

## Templates

```typescript
// Create a template
const tpl = await docbiner.templates.create({
  name: 'Invoice',
  engine: 'handlebars',
  html_content: '<h1>Invoice #{{number}}</h1>',
});

// List templates
const templates = await docbiner.templates.list();

// Preview with data
const html = await docbiner.templates.preview(tpl.id, { number: 42 });

// Update / Delete
await docbiner.templates.update(tpl.id, { name: 'Invoice v2' });
await docbiner.templates.delete(tpl.id);
```

## Merge PDFs

```typescript
const merged = await docbiner.merge({
  sources: [
    { source: '<h1>Page 1</h1>' },
    { source: '<h1>Page 2</h1>' },
    { source: 'https://example.com' },
  ],
});
```

## Usage

```typescript
const current = await docbiner.usage();
console.log(current.conversions, current.quota.remaining);

const history = await docbiner.usageHistory();
```

## Error handling

```typescript
import { Docbiner, DocbinerError } from '@docbiner/sdk';

try {
  await docbiner.convert({ source: '' });
} catch (err) {
  if (err instanceof DocbinerError) {
    console.error(err.status, err.code, err.message);
  }
}
```

## Configuration

```typescript
const docbiner = new Docbiner({
  apiKey: 'db_live_...',
  baseURL: 'https://api.docbiner.com',  // default
  maxRetries: 3,                         // retry on 5xx (1s, 2s, 4s backoff)
  signal: abortController.signal,        // cancellation support
});
```

## License

MIT
