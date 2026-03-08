'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from 'recharts';
import type { MonthlyUsage } from '@/lib/types';

interface UsageChartProps {
  data: MonthlyUsage[];
}

function formatMonth(month: string): string {
  const date = new Date(month + '-01');
  return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' });
}

interface TooltipPayloadItem {
  value: number;
  dataKey: string;
  payload: MonthlyUsage;
}

function CustomTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: TooltipPayloadItem[];
  label?: string;
}) {
  if (!active || !payload?.length) return null;

  return (
    <div className="rounded-lg border border-border bg-card px-3 py-2 shadow-md">
      <p className="mb-1 text-xs font-medium text-muted-foreground">
        {label}
      </p>
      <p className="text-sm font-semibold">
        {payload[0].value.toLocaleString()} conversions
      </p>
    </div>
  );
}

export function UsageChart({ data }: UsageChartProps) {
  const chartData = data.map((item) => ({
    ...item,
    label: formatMonth(item.month),
  }));

  if (!chartData.length) {
    return (
      <div className="flex h-[300px] items-center justify-center text-sm text-muted-foreground">
        No usage data yet
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={chartData} margin={{ top: 5, right: 5, bottom: 5, left: 5 }}>
        <CartesianGrid
          strokeDasharray="3 3"
          vertical={false}
          className="stroke-border"
        />
        <XAxis
          dataKey="label"
          tick={{ fontSize: 12 }}
          tickLine={false}
          axisLine={false}
          className="fill-muted-foreground"
        />
        <YAxis
          tick={{ fontSize: 12 }}
          tickLine={false}
          axisLine={false}
          className="fill-muted-foreground"
          allowDecimals={false}
        />
        <Tooltip content={<CustomTooltip />} cursor={{ fill: 'var(--accent)', opacity: 0.5 }} />
        <Bar
          dataKey="conversions"
          fill="var(--primary)"
          radius={[4, 4, 0, 0]}
          maxBarSize={48}
        />
      </BarChart>
    </ResponsiveContainer>
  );
}
