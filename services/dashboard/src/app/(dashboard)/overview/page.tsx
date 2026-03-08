'use client';

import { useQuery } from '@tanstack/react-query';
import { BarChart3, Clock, CheckCircle, Zap } from 'lucide-react';
import { api } from '@/lib/api';
import { UsageChart } from '@/components/usage-chart';
import type {
  UsageSummary,
  MonthlyUsage,
  Job,
  PaginatedResponse,
} from '@/lib/types';

function StatCard({
  title,
  value,
  subtitle,
  icon: Icon,
}: {
  title: string;
  value: string;
  subtitle?: string;
  icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-muted-foreground">{title}</p>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </div>
      <p className="mt-2 text-2xl font-bold">{value}</p>
      {subtitle && (
        <p className="mt-1 text-xs text-muted-foreground">{subtitle}</p>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: Job['status'] }) {
  const styles: Record<Job['status'], string> = {
    completed:
      'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400',
    failed: 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-400',
    processing:
      'bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-400',
    pending:
      'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-400',
  };

  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${styles[status]}`}
    >
      {status}
    </span>
  );
}

function formatDuration(ms?: number): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function OverviewPage() {
  const {
    data: usage,
    isLoading: usageLoading,
  } = useQuery({
    queryKey: ['usage'],
    queryFn: () => api.get<UsageSummary>('/v1/usage'),
  });

  const {
    data: history,
    isLoading: historyLoading,
  } = useQuery({
    queryKey: ['usage-history'],
    queryFn: () => api.get<MonthlyUsage[]>('/v1/usage/history'),
  });

  const {
    data: jobsResponse,
    isLoading: jobsLoading,
  } = useQuery({
    queryKey: ['recent-jobs'],
    queryFn: () => api.get<PaginatedResponse<Job>>('/v1/jobs?per_page=5'),
  });

  return (
    <div className="mx-auto max-w-6xl space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Overview</h1>
        <p className="text-sm text-muted-foreground">
          Your conversion activity at a glance
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {usageLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-[108px] animate-pulse rounded-xl border border-border bg-muted"
            />
          ))
        ) : (
          <>
            <StatCard
              title="Conversions this month"
              value={
                usage?.current_month?.conversions?.toLocaleString() || '0'
              }
              subtitle={`${usage?.current_month?.test_conversions || 0} test conversions`}
              icon={BarChart3}
            />
            <StatCard
              title="Remaining quota"
              value={usage?.quota?.remaining?.toLocaleString() || '0'}
              subtitle={`of ${usage?.quota?.limit?.toLocaleString() || '0'} total`}
              icon={Zap}
            />
            <StatCard
              title="Avg. conversion time"
              value={formatDuration(usage?.avg_duration_ms)}
              icon={Clock}
            />
            <StatCard
              title="Success rate"
              value={
                usage?.success_rate != null
                  ? `${(usage.success_rate * 100).toFixed(1)}%`
                  : '-'
              }
              icon={CheckCircle}
            />
          </>
        )}
      </div>

      {/* Usage chart */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-base font-semibold">Usage history</h2>
        {historyLoading ? (
          <div className="h-[300px] animate-pulse rounded-lg bg-muted" />
        ) : (
          <UsageChart data={history || []} />
        )}
      </div>

      {/* Recent jobs */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-base font-semibold">Recent jobs</h2>
        {jobsLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div
                key={i}
                className="h-12 animate-pulse rounded-lg bg-muted"
              />
            ))}
          </div>
        ) : !jobsResponse?.data?.length ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            No jobs yet. Create your first conversion via the API or Playground.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-muted-foreground">
                  <th className="pb-3 pr-4 font-medium">Status</th>
                  <th className="pb-3 pr-4 font-medium">Format</th>
                  <th className="pb-3 pr-4 font-medium">Input</th>
                  <th className="pb-3 pr-4 font-medium">Duration</th>
                  <th className="pb-3 font-medium">Created</th>
                </tr>
              </thead>
              <tbody>
                {jobsResponse.data.map((job) => (
                  <tr
                    key={job.id}
                    className="border-b border-border last:border-0"
                  >
                    <td className="py-3 pr-4">
                      <StatusBadge status={job.status} />
                    </td>
                    <td className="py-3 pr-4 font-mono text-xs uppercase">
                      {job.output_format}
                    </td>
                    <td className="py-3 pr-4 text-muted-foreground">
                      {job.input_type}
                    </td>
                    <td className="py-3 pr-4 font-mono text-xs">
                      {formatDuration(job.duration_ms)}
                    </td>
                    <td className="py-3 text-muted-foreground">
                      {formatDate(job.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
