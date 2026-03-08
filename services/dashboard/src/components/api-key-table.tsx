'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Copy, Trash2 } from 'lucide-react';
import { api } from '@/lib/api';
import type { ApiKey } from '@/lib/types';

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return 'Never';

  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  const diffHours = Math.floor(diffMinutes / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSeconds < 60) return 'Just now';
  if (diffMinutes < 60) return `${diffMinutes} minute${diffMinutes > 1 ? 's' : ''} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
  if (diffDays < 30) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;

  return date.toLocaleDateString();
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
}

interface ApiKeyTableProps {
  keys: ApiKey[];
}

export function ApiKeyTable({ keys }: ApiKeyTableProps) {
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/v1/api-keys/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });

  function handleDelete(id: string, name: string) {
    if (confirm(`Are you sure you want to delete the key "${name}"? This action cannot be undone.`)) {
      deleteMutation.mutate(id);
    }
  }

  if (keys.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card p-12 text-center">
        <p className="text-muted-foreground">
          No API keys yet. Create one to get started.
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/50">
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Key</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Environment</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Last used</th>
            <th className="px-4 py-3 text-left font-medium text-muted-foreground">Created</th>
            <th className="px-4 py-3 text-right font-medium text-muted-foreground">Actions</th>
          </tr>
        </thead>
        <tbody>
          {keys.map((key) => (
            <tr key={key.id} className="border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors">
              <td className="px-4 py-3 font-medium">{key.name}</td>
              <td className="px-4 py-3">
                <code className="rounded bg-muted px-2 py-1 font-mono text-xs">
                  {key.key_prefix}...
                </code>
              </td>
              <td className="px-4 py-3">
                <span
                  className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                    key.environment === 'live'
                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400'
                      : 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400'
                  }`}
                >
                  {key.environment === 'live' ? 'Live' : 'Test'}
                </span>
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {formatRelativeTime(key.last_used_at)}
              </td>
              <td className="px-4 py-3 text-muted-foreground">
                {formatDate(key.created_at)}
              </td>
              <td className="px-4 py-3">
                <div className="flex items-center justify-end gap-1">
                  <button
                    onClick={() => copyToClipboard(key.key_prefix)}
                    className="rounded-md p-2 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                    title="Copy key prefix"
                  >
                    <Copy className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(key.id, key.name)}
                    disabled={deleteMutation.isPending}
                    className="rounded-md p-2 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors disabled:opacity-50"
                    title="Delete key"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
