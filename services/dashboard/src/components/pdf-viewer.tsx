'use client';

import { useEffect, useRef } from 'react';
import { Download, Loader2, AlertCircle } from 'lucide-react';

type OutputFormat = 'pdf' | 'png' | 'jpeg' | 'webp';

interface PdfViewerProps {
  blobUrl: string | null;
  format: OutputFormat;
  isLoading: boolean;
  error: string | null;
}

function getFileExtension(format: OutputFormat): string {
  return format === 'jpeg' ? 'jpg' : format;
}

export function PdfViewer({ blobUrl, format, isLoading, error }: PdfViewerProps) {
  const previousBlobUrl = useRef<string | null>(null);

  // Revoke previous blob URL to avoid memory leaks
  useEffect(() => {
    return () => {
      if (previousBlobUrl.current) {
        URL.revokeObjectURL(previousBlobUrl.current);
      }
    };
  }, []);

  useEffect(() => {
    if (previousBlobUrl.current && previousBlobUrl.current !== blobUrl) {
      URL.revokeObjectURL(previousBlobUrl.current);
    }
    previousBlobUrl.current = blobUrl;
  }, [blobUrl]);

  const handleDownload = () => {
    if (!blobUrl) return;
    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = `document.${getFileExtension(format)}`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-muted-foreground">
        <Loader2 className="h-8 w-8 animate-spin" />
        <p className="text-sm">Generating {format.toUpperCase()}...</p>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
          <AlertCircle className="h-6 w-6 text-destructive" />
        </div>
        <div>
          <p className="text-sm font-medium text-destructive">Conversion failed</p>
          <p className="mt-1 text-xs text-muted-foreground">{error}</p>
        </div>
      </div>
    );
  }

  // Empty state
  if (!blobUrl) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
        <p className="text-sm">Click &quot;Generate&quot; to preview the output</p>
        <p className="text-xs">The result will appear here</p>
      </div>
    );
  }

  // Preview state
  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b border-border px-4 py-2">
        <p className="text-xs font-medium text-muted-foreground uppercase">
          {format} Preview
        </p>
        <button
          onClick={handleDownload}
          className="inline-flex items-center gap-1.5 rounded-md border border-input bg-background px-3 py-1.5 text-xs font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
        >
          <Download className="h-3.5 w-3.5" />
          Download
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto bg-muted/30 p-4">
        {format === 'pdf' ? (
          <iframe
            src={blobUrl}
            className="h-full w-full rounded-md border border-border bg-white"
            title="PDF Preview"
          />
        ) : (
          <div className="flex items-start justify-center">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={blobUrl}
              alt="Generated preview"
              className="max-w-full rounded-md border border-border shadow-sm"
            />
          </div>
        )}
      </div>
    </div>
  );
}
