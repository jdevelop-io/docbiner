'use client';

import { useState, useCallback } from 'react';
import { Play, Clipboard, Check, AlertTriangle } from 'lucide-react';
import { useAuth } from '@/lib/auth-context';
import { PlaygroundEditor, DEFAULT_HTML } from '@/components/playground-editor';
import { PdfViewer } from '@/components/pdf-viewer';

type OutputFormat = 'pdf' | 'png' | 'jpeg' | 'webp';
type PageSize = 'A4' | 'Letter' | 'Legal';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function PlaygroundPage() {
  const { token } = useAuth();

  const [htmlContent, setHtmlContent] = useState(DEFAULT_HTML);
  const [format, setFormat] = useState<OutputFormat>('pdf');
  const [pageSize, setPageSize] = useState<PageSize>('A4');
  const [landscape, setLandscape] = useState(false);

  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const isPdf = format === 'pdf';

  const buildRequestBody = useCallback(() => {
    const body: Record<string, unknown> = {
      source: htmlContent,
      format,
    };

    if (isPdf) {
      body.options = {
        page_size: pageSize,
        landscape,
      };
    }

    return body;
  }, [htmlContent, format, isPdf, pageSize, landscape]);

  const handleGenerate = useCallback(async () => {
    if (!token) return;

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/v1/convert`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(buildRequestBody()),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Conversion failed' }));
        throw new Error(errorData.message || `Error ${response.status}`);
      }

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      setBlobUrl(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An unexpected error occurred');
      setBlobUrl(null);
    } finally {
      setIsLoading(false);
    }
  }, [token, buildRequestBody]);

  const handleCopyCurl = useCallback(async () => {
    const body = buildRequestBody();
    const jsonBody = JSON.stringify(body);

    const curlCommand = [
      'curl -X POST https://api.docbiner.com/v1/convert \\',
      '  -H "Authorization: Bearer YOUR_API_KEY" \\',
      '  -H "Content-Type: application/json" \\',
      `  -d '${jsonBody}'`,
    ].join('\n');

    try {
      await navigator.clipboard.writeText(curlCommand);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for clipboard API failure
      const textarea = document.createElement('textarea');
      textarea.value = curlCommand;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [buildRequestBody]);

  return (
    <div className="flex h-[calc(100vh-3.5rem-3rem)] flex-col lg:h-[calc(100vh-3rem)]">
      {/* Header */}
      <div className="shrink-0 space-y-1 pb-4">
        <h1 className="text-2xl font-bold tracking-tight">Playground</h1>
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />
          <span>
            Playground uses test environment &mdash; output will be watermarked.
          </span>
        </div>
      </div>

      {/* Options bar */}
      <div className="shrink-0 flex flex-wrap items-center gap-3 rounded-lg border border-border bg-card px-4 py-3 mb-4">
        {/* Format */}
        <div className="flex items-center gap-2">
          <label htmlFor="format-select" className="text-xs font-medium text-muted-foreground">
            Format
          </label>
          <select
            id="format-select"
            value={format}
            onChange={(e) => setFormat(e.target.value as OutputFormat)}
            className="rounded-md border border-input bg-background px-3 py-1.5 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="pdf">PDF</option>
            <option value="png">PNG</option>
            <option value="jpeg">JPEG</option>
            <option value="webp">WebP</option>
          </select>
        </div>

        {/* Page size (PDF only) */}
        {isPdf && (
          <div className="flex items-center gap-2">
            <label htmlFor="page-size-select" className="text-xs font-medium text-muted-foreground">
              Page size
            </label>
            <select
              id="page-size-select"
              value={pageSize}
              onChange={(e) => setPageSize(e.target.value as PageSize)}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="A4">A4</option>
              <option value="Letter">Letter</option>
              <option value="Legal">Legal</option>
            </select>
          </div>
        )}

        {/* Orientation (PDF only) */}
        {isPdf && (
          <div className="flex items-center gap-2">
            <label htmlFor="orientation-select" className="text-xs font-medium text-muted-foreground">
              Orientation
            </label>
            <select
              id="orientation-select"
              value={landscape ? 'landscape' : 'portrait'}
              onChange={(e) => setLandscape(e.target.value === 'landscape')}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
            >
              <option value="portrait">Portrait</option>
              <option value="landscape">Landscape</option>
            </select>
          </div>
        )}

        {/* Spacer */}
        <div className="flex-1" />

        {/* Action buttons */}
        <button
          onClick={handleCopyCurl}
          className="inline-flex items-center gap-1.5 rounded-md border border-input bg-background px-3 py-1.5 text-sm font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
        >
          {copied ? (
            <>
              <Check className="h-3.5 w-3.5 text-emerald-500" />
              Copied!
            </>
          ) : (
            <>
              <Clipboard className="h-3.5 w-3.5" />
              Copy cURL
            </>
          )}
        </button>

        <button
          onClick={handleGenerate}
          disabled={isLoading || !htmlContent.trim()}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground shadow-sm transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Play className="h-3.5 w-3.5" />
          Generate
        </button>
      </div>

      {/* Split layout */}
      <div className="flex min-h-0 flex-1 gap-4">
        {/* Left panel - Editor */}
        <div className="flex w-1/2 flex-col overflow-hidden rounded-lg border border-border bg-card">
          <PlaygroundEditor value={htmlContent} onChange={setHtmlContent} />
        </div>

        {/* Right panel - Preview */}
        <div className="flex w-1/2 flex-col overflow-hidden rounded-lg border border-border bg-card">
          <PdfViewer
            blobUrl={blobUrl}
            format={format}
            isLoading={isLoading}
            error={error}
          />
        </div>
      </div>
    </div>
  );
}
