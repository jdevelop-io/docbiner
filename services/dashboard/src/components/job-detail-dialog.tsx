'use client';

import { X, Download, AlertCircle, Clock, FileOutput, Globe, Code, LayoutTemplate } from 'lucide-react';
import type { Job } from '@/lib/types';

const statusConfig = {
  pending: { label: 'Pending', className: 'bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300' },
  processing: { label: 'Processing', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  completed: { label: 'Completed', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  failed: { label: 'Failed', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
} as const;

const inputTypeIcons = {
  url: Globe,
  html: Code,
  template: LayoutTemplate,
} as const;

function formatDuration(ms?: number): string {
  if (ms == null) return '-';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatSize(bytes?: number): string {
  if (bytes == null) return '-';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

interface JobDetailDialogProps {
  job: Job | null;
  open: boolean;
  onClose: () => void;
  onDownload?: (jobId: string) => void;
}

export function JobDetailDialog({ job, open, onClose, onDownload }: JobDetailDialogProps) {
  if (!open || !job) return null;

  const status = statusConfig[job.status];
  const InputIcon = inputTypeIcons[job.input_type];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="fixed inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />
      <div className="relative z-10 w-full max-w-lg rounded-lg border border-border bg-card p-6 shadow-xl mx-4">
        <div className="mb-6 flex items-start justify-between">
          <div>
            <h2 className="text-lg font-semibold">Job Details</h2>
            <p className="mt-1 font-mono text-xs text-muted-foreground">{job.id}</p>
          </div>
          <button
            onClick={onClose}
            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <DetailField label="Status">
              <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${status.className}`}>
                {status.label}
              </span>
            </DetailField>

            <DetailField label="Format">
              <div className="flex items-center gap-1.5">
                <FileOutput className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-sm uppercase">{job.output_format}</span>
              </div>
            </DetailField>

            <DetailField label="Input Type">
              <div className="flex items-center gap-1.5">
                <InputIcon className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-sm capitalize">{job.input_type}</span>
              </div>
            </DetailField>

            <DetailField label="Delivery">
              <span className="text-sm capitalize">{job.delivery_method}</span>
            </DetailField>

            <DetailField label="Duration">
              <div className="flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="font-mono text-sm">{formatDuration(job.duration_ms)}</span>
              </div>
            </DetailField>

            <DetailField label="Size">
              <span className="text-sm">{formatSize(job.result_size)}</span>
            </DetailField>

            {job.pages_count != null && (
              <DetailField label="Pages">
                <span className="text-sm">{job.pages_count}</span>
              </DetailField>
            )}

            <DetailField label="Test">
              <span className="text-sm">{job.is_test ? 'Yes' : 'No'}</span>
            </DetailField>
          </div>

          <div className="border-t border-border pt-4">
            <div className="grid grid-cols-2 gap-4">
              <DetailField label="Created">
                <span className="text-sm">{formatDate(job.created_at)}</span>
              </DetailField>
              {job.completed_at && (
                <DetailField label="Completed">
                  <span className="text-sm">{formatDate(job.completed_at)}</span>
                </DetailField>
              )}
            </div>
          </div>

          {job.result_url && (
            <div className="border-t border-border pt-4">
              <DetailField label="Result URL">
                <p className="truncate font-mono text-xs text-muted-foreground" title={job.result_url}>
                  {job.result_url}
                </p>
              </DetailField>
            </div>
          )}

          {job.status === 'failed' && job.error_message && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-900/10">
              <div className="flex items-start gap-2">
                <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-red-600 dark:text-red-400" />
                <div>
                  <p className="text-sm font-medium text-red-800 dark:text-red-300">Error</p>
                  <p className="mt-1 text-sm text-red-700 dark:text-red-400">{job.error_message}</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {job.status === 'completed' && job.result_url && onDownload && (
          <div className="mt-6 flex justify-end border-t border-border pt-4">
            <button
              onClick={() => onDownload(job.id)}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            >
              <Download className="h-4 w-4" />
              Download Result
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function DetailField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <div className="mt-1">{children}</div>
    </div>
  );
}
