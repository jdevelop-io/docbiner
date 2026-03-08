'use client';

import type { MonthlyUsage } from '@/lib/types';

function formatMonth(monthStr: string): string {
  const date = new Date(monthStr + '-01');
  return date.toLocaleDateString('en-US', {
    month: 'long',
    year: 'numeric',
  });
}

function formatCurrency(amount: number): string {
  if (amount === 0) return '-';
  return `$${amount.toFixed(2)}`;
}

interface UsageTableProps {
  data: MonthlyUsage[];
  isLoading?: boolean;
}

export function UsageTable({ data, isLoading }: UsageTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div
            key={i}
            className="h-12 animate-pulse rounded-lg bg-muted"
          />
        ))}
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-12 text-center">
        <p className="text-sm text-muted-foreground">
          No usage history yet.
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/50">
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">
              Month
            </th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">
              Conversions
            </th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">
              Test Conversions
            </th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">
              Overage
            </th>
          </tr>
        </thead>
        <tbody>
          {data.map((row) => (
            <tr
              key={row.month}
              className="border-b border-border last:border-0 transition-colors hover:bg-muted/30"
            >
              <td className="px-4 py-3 font-medium">
                {formatMonth(row.month)}
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {row.conversions.toLocaleString()}
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {row.test_conversions.toLocaleString()}
              </td>
              <td className="px-4 py-3">
                {row.overage_amount > 0 ? (
                  <span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/30 dark:text-amber-400">
                    {formatCurrency(row.overage_amount)}
                  </span>
                ) : (
                  <span className="text-muted-foreground">-</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
