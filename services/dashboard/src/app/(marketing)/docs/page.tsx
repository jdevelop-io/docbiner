'use client';

import { useState } from 'react';
import Link from 'next/link';
import { cn } from '@/lib/utils';

/* ------------------------------------------------------------------ */
/*  Code Block component                                              */
/* ------------------------------------------------------------------ */

function CodeBlock({ children, title }: { children: string; title?: string }) {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(children).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="group relative overflow-hidden rounded-lg border border-border bg-muted/50">
      {title && (
        <div className="border-b border-border bg-muted/80 px-4 py-2">
          <span className="text-xs font-medium text-muted-foreground">{title}</span>
        </div>
      )}
      <div className="relative">
        <pre className="overflow-x-auto p-4 text-sm leading-relaxed">
          <code className="font-mono text-foreground/90">{children}</code>
        </pre>
        <button
          onClick={handleCopy}
          className="absolute right-2 top-2 rounded-md bg-background/80 px-2 py-1 text-xs font-medium text-muted-foreground opacity-0 backdrop-blur-sm transition-opacity group-hover:opacity-100 hover:text-foreground"
        >
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Endpoint badge                                                    */
/* ------------------------------------------------------------------ */

function MethodBadge({ method }: { method: string }) {
  const colors: Record<string, string> = {
    GET: 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-400',
    POST: 'bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-400',
    PUT: 'bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-400',
    PATCH: 'bg-amber-100 text-amber-800 dark:bg-amber-950 dark:text-amber-400',
    DELETE: 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-400',
  };

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-bold uppercase',
        colors[method] || 'bg-muted text-muted-foreground',
      )}
    >
      {method}
    </span>
  );
}

function Endpoint({ method, path }: { method: string; path: string }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-muted/50 px-4 py-2.5">
      <MethodBadge method={method} />
      <code className="font-mono text-sm">{path}</code>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Sidebar Navigation                                                */
/* ------------------------------------------------------------------ */

const sections = [
  { id: 'authentication', label: 'Authentication' },
  { id: 'convert', label: 'Convert' },
  { id: 'async-convert', label: 'Async Convert' },
  { id: 'jobs', label: 'Jobs' },
  { id: 'templates', label: 'Templates' },
  { id: 'merge', label: 'Merge' },
  { id: 'usage', label: 'Usage' },
];

function DocsSidebar() {
  return (
    <nav className="sticky top-20 hidden lg:block">
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        On this page
      </h3>
      <ul className="space-y-1">
        {sections.map((section) => (
          <li key={section.id}>
            <a
              href={`#${section.id}`}
              className="block rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              {section.label}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}

/* ------------------------------------------------------------------ */
/*  Documentation Sections                                            */
/* ------------------------------------------------------------------ */

function AuthenticationSection() {
  return (
    <section id="authentication" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Authentication</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        All API requests require authentication via an API key. Include your key
        in the <code className="rounded bg-muted px-1.5 py-0.5 text-sm font-mono">Authorization</code> header
        as a Bearer token.
      </p>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        API keys are created in the{' '}
        <Link href="/api-keys" className="font-medium text-foreground underline underline-offset-4 hover:text-foreground/80">
          Dashboard
        </Link>
        . Each key is scoped to an organization and has an environment:
      </p>
      <ul className="mt-3 list-disc space-y-1 pl-6 text-sm text-muted-foreground">
        <li>
          <strong className="text-foreground">Live keys</strong> (<code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">db_live_...</code>) &mdash; Count toward your quota
        </li>
        <li>
          <strong className="text-foreground">Test keys</strong> (<code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">db_test_...</code>) &mdash; Free, watermarked output
        </li>
      </ul>

      <div className="mt-6">
        <CodeBlock title="Example request header">{`Authorization: Bearer db_live_abc123def456`}</CodeBlock>
      </div>

      <div className="mt-4">
        <CodeBlock title="Error response (401)">{`{
  "error": {
    "code": "unauthorized",
    "message": "Invalid or missing API key."
  }
}`}</CodeBlock>
      </div>
    </section>
  );
}

function ConvertSection() {
  return (
    <section id="convert" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Convert</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        Convert HTML content or a URL to PDF or image format synchronously. The
        response body contains the generated file.
      </p>

      <div className="mt-6">
        <Endpoint method="POST" path="/v1/convert" />
      </div>

      <div className="mt-6 space-y-4">
        <CodeBlock title="Request body">{`{
  "source": "<h1>Hello World</h1>",
  "format": "pdf",
  "options": {
    "page_size": "A4",
    "margin": { "top": "20mm", "bottom": "20mm" },
    "print_background": true,
    "landscape": false
  }
}`}</CodeBlock>

        <CodeBlock title="Request body (URL source)">{`{
  "source": "https://example.com",
  "source_type": "url",
  "format": "png",
  "options": {
    "viewport": { "width": 1280, "height": 720 },
    "full_page": true
  }
}`}</CodeBlock>

        <CodeBlock title="Response headers">{`Content-Type: application/pdf
Content-Disposition: attachment; filename="document.pdf"
X-Docbiner-Job-Id: job_abc123
X-Docbiner-Duration-Ms: 1250
X-Docbiner-Pages: 3`}</CodeBlock>
      </div>

      <div className="mt-4">
        <h3 className="text-base font-semibold">Parameters</h3>
        <div className="mt-3 overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left">
                <th className="pb-2 pr-4 font-medium">Field</th>
                <th className="pb-2 pr-4 font-medium">Type</th>
                <th className="pb-2 pr-4 font-medium">Required</th>
                <th className="pb-2 font-medium">Description</th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr className="border-b border-border/50">
                <td className="py-2 pr-4 font-mono text-xs text-foreground">source</td>
                <td className="py-2 pr-4">string</td>
                <td className="py-2 pr-4">Yes</td>
                <td className="py-2">HTML string or URL to convert</td>
              </tr>
              <tr className="border-b border-border/50">
                <td className="py-2 pr-4 font-mono text-xs text-foreground">source_type</td>
                <td className="py-2 pr-4">string</td>
                <td className="py-2 pr-4">No</td>
                <td className="py-2">&quot;html&quot; (default) or &quot;url&quot;</td>
              </tr>
              <tr className="border-b border-border/50">
                <td className="py-2 pr-4 font-mono text-xs text-foreground">format</td>
                <td className="py-2 pr-4">string</td>
                <td className="py-2 pr-4">Yes</td>
                <td className="py-2">&quot;pdf&quot;, &quot;png&quot;, &quot;jpeg&quot;, or &quot;webp&quot;</td>
              </tr>
              <tr className="border-b border-border/50">
                <td className="py-2 pr-4 font-mono text-xs text-foreground">options</td>
                <td className="py-2 pr-4">object</td>
                <td className="py-2 pr-4">No</td>
                <td className="py-2">Format-specific options (page size, margins, etc.)</td>
              </tr>
              <tr className="border-b border-border/50">
                <td className="py-2 pr-4 font-mono text-xs text-foreground">template_id</td>
                <td className="py-2 pr-4">string</td>
                <td className="py-2 pr-4">No</td>
                <td className="py-2">Use a saved template instead of raw HTML</td>
              </tr>
              <tr>
                <td className="py-2 pr-4 font-mono text-xs text-foreground">data</td>
                <td className="py-2 pr-4">object</td>
                <td className="py-2 pr-4">No</td>
                <td className="py-2">Template variables (required when using template_id)</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

function AsyncConvertSection() {
  return (
    <section id="async-convert" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Async Convert</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        Queue a conversion job for asynchronous processing. Ideal for long-running
        conversions or when you want to receive results via webhook.
      </p>

      <div className="mt-6">
        <Endpoint method="POST" path="/v1/convert/async" />
      </div>

      <div className="mt-6 space-y-4">
        <CodeBlock title="Request body">{`{
  "source": "<h1>Large Report</h1><p>...</p>",
  "format": "pdf",
  "webhook_url": "https://your-app.com/webhooks/docbiner",
  "delivery_method": "webhook",
  "options": {
    "page_size": "A4",
    "print_background": true
  }
}`}</CodeBlock>

        <CodeBlock title="Response (202 Accepted)">{`{
  "job_id": "job_abc123def456",
  "status": "pending",
  "created_at": "2026-03-04T12:00:00Z"
}`}</CodeBlock>

        <CodeBlock title="Webhook payload (on completion)">{`{
  "event": "job.completed",
  "job_id": "job_abc123def456",
  "status": "completed",
  "result_url": "https://cdn.docbiner.com/results/job_abc123def456.pdf",
  "duration_ms": 3200,
  "pages_count": 12,
  "completed_at": "2026-03-04T12:00:03Z"
}`}</CodeBlock>
      </div>
    </section>
  );
}

function JobsSection() {
  return (
    <section id="jobs" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Jobs</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        Retrieve the status of async jobs or cancel pending ones.
      </p>

      <div className="mt-6 space-y-4">
        <div>
          <Endpoint method="GET" path="/v1/jobs/:id" />
          <div className="mt-4">
            <CodeBlock title="Response">{`{
  "id": "job_abc123def456",
  "status": "completed",
  "input_type": "html",
  "output_format": "pdf",
  "delivery_method": "webhook",
  "result_url": "https://cdn.docbiner.com/results/job_abc123def456.pdf",
  "result_size": 245120,
  "pages_count": 12,
  "duration_ms": 3200,
  "is_test": false,
  "created_at": "2026-03-04T12:00:00Z",
  "completed_at": "2026-03-04T12:00:03Z"
}`}</CodeBlock>
          </div>
        </div>

        <div>
          <Endpoint method="GET" path="/v1/jobs?page=1&per_page=20" />
          <div className="mt-4">
            <CodeBlock title="Response (paginated list)">{`{
  "data": [
    { "id": "job_abc123", "status": "completed", "output_format": "pdf", ... },
    { "id": "job_def456", "status": "failed", "output_format": "png", ... }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 142,
    "total_pages": 8
  }
}`}</CodeBlock>
          </div>
        </div>

        <div>
          <Endpoint method="DELETE" path="/v1/jobs/:id" />
          <p className="mt-3 text-sm text-muted-foreground">
            Cancel a pending or processing job. Completed and failed jobs cannot be cancelled.
          </p>
          <div className="mt-4">
            <CodeBlock title="Response (204 No Content)">{`// Empty response body`}</CodeBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

function TemplatesSection() {
  return (
    <section id="templates" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Templates</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        Create and manage reusable HTML templates with Handlebars or Liquid
        syntax. Templates let you separate your document layout from the data
        that populates it.
      </p>

      <div className="mt-6 space-y-6">
        {/* Create */}
        <div>
          <Endpoint method="POST" path="/v1/templates" />
          <div className="mt-4">
            <CodeBlock title="Request body">{`{
  "name": "Invoice",
  "engine": "handlebars",
  "html_content": "<h1>Invoice #` + '{{invoice_number}}' + `</h1><p>Amount: $` + '{{amount}}' + `</p>",
  "css_content": "body { font-family: sans-serif; padding: 40px; }",
  "sample_data": {
    "invoice_number": "INV-001",
    "amount": "99.00"
  }
}`}</CodeBlock>
          </div>
          <div className="mt-4">
            <CodeBlock title="Response (201 Created)">{`{
  "id": "tpl_abc123",
  "name": "Invoice",
  "engine": "handlebars",
  "html_content": "<h1>Invoice #` + '{{invoice_number}}' + `</h1>...",
  "css_content": "body { font-family: sans-serif; ... }",
  "sample_data": { "invoice_number": "INV-001", "amount": "99.00" },
  "created_at": "2026-03-04T12:00:00Z",
  "updated_at": "2026-03-04T12:00:00Z"
}`}</CodeBlock>
          </div>
        </div>

        {/* List */}
        <div>
          <Endpoint method="GET" path="/v1/templates" />
          <p className="mt-3 text-sm text-muted-foreground">
            Returns a paginated list of all templates in your organization.
          </p>
        </div>

        {/* Get */}
        <div>
          <Endpoint method="GET" path="/v1/templates/:id" />
          <p className="mt-3 text-sm text-muted-foreground">
            Retrieve a single template by ID.
          </p>
        </div>

        {/* Update */}
        <div>
          <Endpoint method="PUT" path="/v1/templates/:id" />
          <p className="mt-3 text-sm text-muted-foreground">
            Update an existing template. Accepts the same fields as creation.
          </p>
        </div>

        {/* Delete */}
        <div>
          <Endpoint method="DELETE" path="/v1/templates/:id" />
          <p className="mt-3 text-sm text-muted-foreground">
            Permanently delete a template. Returns 204 No Content on success.
          </p>
        </div>

        {/* Using templates */}
        <div>
          <h3 className="text-base font-semibold">Using a template for conversion</h3>
          <div className="mt-3">
            <CodeBlock title="POST /v1/convert">{`{
  "template_id": "tpl_abc123",
  "format": "pdf",
  "data": {
    "invoice_number": "INV-042",
    "amount": "250.00"
  }
}`}</CodeBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

function MergeSection() {
  return (
    <section id="merge" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Merge</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        Merge multiple PDF sources into a single document. Each source can be
        raw HTML, a URL, or a template reference.
      </p>

      <div className="mt-6">
        <Endpoint method="POST" path="/v1/merge" />
      </div>

      <div className="mt-6 space-y-4">
        <CodeBlock title="Request body">{`{
  "sources": [
    {
      "source": "<h1>Cover Page</h1>",
      "options": { "page_size": "A4" }
    },
    {
      "template_id": "tpl_abc123",
      "data": { "invoice_number": "INV-042", "amount": "250.00" }
    },
    {
      "source": "https://example.com/report",
      "source_type": "url"
    }
  ],
  "options": {
    "password": "secret123"
  }
}`}</CodeBlock>

        <CodeBlock title="Response headers">{`Content-Type: application/pdf
Content-Disposition: attachment; filename="merged.pdf"
X-Docbiner-Job-Id: job_merge_abc123
X-Docbiner-Duration-Ms: 4500
X-Docbiner-Pages: 24`}</CodeBlock>
      </div>
    </section>
  );
}

function UsageSection() {
  return (
    <section id="usage" className="scroll-mt-24">
      <h2 className="text-2xl font-bold tracking-tight">Usage</h2>
      <p className="mt-3 text-muted-foreground leading-relaxed">
        Retrieve your current usage statistics and quota information.
      </p>

      <div className="mt-6 space-y-6">
        <div>
          <Endpoint method="GET" path="/v1/usage" />
          <div className="mt-4">
            <CodeBlock title="Response">{`{
  "quota": {
    "used": 1234,
    "limit": 15000,
    "remaining": 13766
  },
  "current_month": {
    "month": "2026-03",
    "conversions": 1234,
    "test_conversions": 56,
    "overage_amount": 0
  },
  "avg_duration_ms": 1850,
  "success_rate": 0.987
}`}</CodeBlock>
          </div>
        </div>

        <div>
          <Endpoint method="GET" path="/v1/usage/history" />
          <div className="mt-4">
            <CodeBlock title="Response">{`[
  {
    "month": "2026-03",
    "conversions": 1234,
    "test_conversions": 56,
    "overage_amount": 0
  },
  {
    "month": "2026-02",
    "conversions": 8921,
    "test_conversions": 120,
    "overage_amount": 0
  }
]`}</CodeBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Page                                                              */
/* ------------------------------------------------------------------ */

export default function DocsPage() {
  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
      {/* Page header */}
      <div className="mb-16">
        <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl">
          API Documentation
        </h1>
        <p className="mt-4 max-w-2xl text-lg text-muted-foreground">
          Complete reference for the Docbiner REST API. Base URL:{' '}
          <code className="rounded bg-muted px-2 py-0.5 font-mono text-sm">
            https://api.docbiner.com
          </code>
        </p>
      </div>

      <div className="flex gap-12">
        {/* Sidebar */}
        <div className="hidden w-48 shrink-0 lg:block">
          <DocsSidebar />
        </div>

        {/* Content */}
        <div className="min-w-0 flex-1 space-y-20">
          <AuthenticationSection />
          <ConvertSection />
          <AsyncConvertSection />
          <JobsSection />
          <TemplatesSection />
          <MergeSection />
          <UsageSection />
        </div>
      </div>
    </div>
  );
}
