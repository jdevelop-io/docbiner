'use client';

import { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Briefcase, Loader2 } from 'lucide-react';
import { useAuth } from '@/lib/auth-context';
import { api } from '@/lib/api';
import type { Job, PaginatedResponse } from '@/lib/types';
import { JobsTable } from '@/components/jobs-table';
import { JobDetailDialog } from '@/components/job-detail-dialog';

type StatusFilter = 'all' | 'pending' | 'processing' | 'completed' | 'failed';
type FormatFilter = 'all' | 'pdf' | 'png' | 'jpeg' | 'webp';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export default function JobsPage() {
  const { token } = useAuth();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [formatFilter, setFormatFilter] = useState<FormatFilter>('all');
  const [selectedJob, setSelectedJob] = useState<Job | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const perPage = 20;

  const queryParams = new URLSearchParams({
    page: page.toString(),
    per_page: perPage.toString(),
  });
  if (statusFilter !== 'all') queryParams.set('status', statusFilter);
  if (formatFilter !== 'all') queryParams.set('format', formatFilter);

  const { data, isLoading, error } = useQuery({
    queryKey: ['jobs', page, statusFilter, formatFilter],
    queryFn: () => api.get<PaginatedResponse<Job>>(`/v1/jobs?${queryParams.toString()}`),
  });

  const handlePageChange = useCallback((newPage: number) => {
    setPage(newPage);
  }, []);

  const handleRowClick = useCallback((job: Job) => {
    setSelectedJob(job);
    setDialogOpen(true);
  }, []);

  const handleDownload = useCallback(async (jobId: string) => {
    try {
      const res = await fetch(`${API_BASE}/v1/jobs/${jobId}/download`, {
        headers: { Authorization: `Bearer ${token}` },
        redirect: 'follow',
      });
      if (!res.ok) return;
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      const ext = blob.type.includes('pdf') ? 'pdf' : blob.type.includes('png') ? 'png' : blob.type.includes('jpeg') ? 'jpeg' : 'webp';
      a.download = `job-${jobId.slice(0, 8)}.${ext}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // silently fail
    }
  }, [token]);

  const handleCloseDialog = useCallback(() => {
    setDialogOpen(false);
    setSelectedJob(null);
  }, []);

  const handleStatusChange = useCallback((value: string) => {
    setStatusFilter(value as StatusFilter);
    setPage(1);
  }, []);

  const handleFormatChange = useCallback((value: string) => {
    setFormatFilter(value as FormatFilter);
    setPage(1);
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Jobs</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          View and manage your conversion jobs history.
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <label htmlFor="status-filter" className="text-sm font-medium text-muted-foreground">
            Status
          </label>
          <select
            id="status-filter"
            value={statusFilter}
            onChange={(e) => handleStatusChange(e.target.value)}
            className="rounded-md border border-input bg-background px-3 py-1.5 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="all">All</option>
            <option value="pending">Pending</option>
            <option value="processing">Processing</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
          </select>
        </div>

        <div className="flex items-center gap-2">
          <label htmlFor="format-filter" className="text-sm font-medium text-muted-foreground">
            Format
          </label>
          <select
            id="format-filter"
            value={formatFilter}
            onChange={(e) => handleFormatChange(e.target.value)}
            className="rounded-md border border-input bg-background px-3 py-1.5 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="all">All</option>
            <option value="pdf">PDF</option>
            <option value="png">PNG</option>
            <option value="jpeg">JPEG</option>
            <option value="webp">WebP</option>
          </select>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/10">
          <p className="text-sm text-red-700 dark:text-red-400">
            Failed to load jobs. Please try again.
          </p>
        </div>
      ) : data ? (
        <JobsTable
          jobs={data.data}
          pagination={data.pagination}
          onPageChange={handlePageChange}
          onRowClick={handleRowClick}
          onDownload={handleDownload}
        />
      ) : (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <Briefcase className="h-10 w-10 text-muted-foreground/50" />
          <p className="mt-4 text-sm text-muted-foreground">No jobs yet</p>
        </div>
      )}

      <JobDetailDialog
        job={selectedJob}
        open={dialogOpen}
        onClose={handleCloseDialog}
        onDownload={handleDownload}
      />
    </div>
  );
}
