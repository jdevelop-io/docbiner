'use client';

import { Download, Eye, ChevronLeft, ChevronRight } from 'lucide-react';
import type { Job, Pagination } from '@/lib/types';

const statusConfig = {
  pending: { label: 'Pending', className: 'bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300' },
  processing: { label: 'Processing', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  completed: { label: 'Completed', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  failed: { label: 'Failed', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
} as const;

const formatConfig = {
  pdf: { label: 'PDF', className: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400' },
  png: { label: 'PNG', className: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400' },
  jpeg: { label: 'JPEG', className: 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-400' },
  webp: { label: 'WebP', className: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400' },
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

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 30) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

interface JobsTableProps {
  jobs: Job[];
  pagination: Pagination;
  onPageChange: (page: number) => void;
  onRowClick: (job: Job) => void;
  onDownload?: (jobId: string) => void;
}

export function JobsTable({ jobs, pagination, onPageChange, onRowClick, onDownload }: JobsTableProps) {
  return (
    <div className="space-y-4">
      <div className="overflow-x-auto rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50">
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">ID</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Format</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Input</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Duration</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Size</th>
              <th className="px-4 py-3 text-left font-medium text-muted-foreground">Created</th>
              <th className="px-4 py-3 text-right font-medium text-muted-foreground">Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-4 py-12 text-center text-muted-foreground">
                  No jobs found
                </td>
              </tr>
            ) : (
              jobs.map((job) => {
                const status = statusConfig[job.status];
                const format = formatConfig[job.output_format];

                return (
                  <tr
                    key={job.id}
                    className="border-b border-border last:border-0 cursor-pointer transition-colors hover:bg-muted/50"
                    onClick={() => onRowClick(job)}
                  >
                    <td className="px-4 py-3 font-mono text-xs">
                      {job.id.slice(0, 8)}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${status.className}`}>
                        {status.label}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${format.className}`}>
                        {format.label}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground capitalize">
                      {job.input_type}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {formatDuration(job.duration_ms)}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {formatSize(job.result_size)}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {formatRelativeTime(job.created_at)}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        {job.status === 'completed' && job.result_url && onDownload && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              onDownload(job.id);
                            }}
                            className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                            title="Download"
                          >
                            <Download className="h-4 w-4" />
                          </button>
                        )}
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onRowClick(job);
                          }}
                          className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                          title="View details"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {pagination.total_pages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Page {pagination.page} of {pagination.total_pages} ({pagination.total} total)
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => onPageChange(pagination.page - 1)}
              disabled={pagination.page <= 1}
              className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm font-medium transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-50"
            >
              <ChevronLeft className="h-4 w-4" />
              Previous
            </button>
            <button
              onClick={() => onPageChange(pagination.page + 1)}
              disabled={pagination.page >= pagination.total_pages}
              className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm font-medium transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-50"
            >
              Next
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
