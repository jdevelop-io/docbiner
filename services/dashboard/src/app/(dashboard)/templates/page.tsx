'use client';

import { useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Code, Trash2, Loader2 } from 'lucide-react';
import { api } from '@/lib/api';
import type { Template } from '@/lib/types';
import { cn } from '@/lib/utils';

export default function TemplatesPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const { data: templates = [], isLoading, error } = useQuery({
    queryKey: ['templates'],
    queryFn: () => api.get<Template[]>('/v1/templates'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/v1/templates/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      setConfirmDeleteId(null);
      setDeletingId(null);
    },
    onError: () => {
      setDeletingId(null);
    },
  });

  const handleDelete = useCallback(
    (id: string) => {
      setDeletingId(id);
      deleteMutation.mutate(id);
    },
    [deleteMutation],
  );

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Templates</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Create and manage your HTML templates for document generation.
          </p>
        </div>
        <button
          onClick={() => router.push('/templates/new')}
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-4 w-4" />
          New Template
        </button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/10">
          <p className="text-sm text-red-700 dark:text-red-400">
            Failed to load templates. Please try again.
          </p>
        </div>
      ) : templates.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-16 text-center">
          <Code className="h-10 w-10 text-muted-foreground/50" />
          <h3 className="mt-4 text-sm font-semibold">No templates</h3>
          <p className="mt-1 text-sm text-muted-foreground">
            Get started by creating your first template.
          </p>
          <button
            onClick={() => router.push('/templates/new')}
            className="mt-4 inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
          >
            <Plus className="h-4 w-4" />
            New Template
          </button>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {templates.map((template) => (
            <div
              key={template.id}
              className="group relative rounded-lg border border-border bg-card transition-colors hover:border-foreground/20"
            >
              <button
                onClick={() => router.push(`/templates/${template.id}`)}
                className="block w-full p-4 text-left"
              >
                <div className="flex items-start justify-between">
                  <h3 className="font-semibold text-sm truncate pr-2">
                    {template.name}
                  </h3>
                  <span
                    className={cn(
                      'shrink-0 rounded-full px-2 py-0.5 text-xs font-medium',
                      template.engine === 'handlebars'
                        ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                        : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
                    )}
                  >
                    {template.engine}
                  </span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  Updated {formatDate(template.updated_at)}
                </p>
              </button>

              {/* Delete button */}
              <div className="absolute bottom-3 right-3">
                {confirmDeleteId === template.id ? (
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => handleDelete(template.id)}
                      disabled={deletingId === template.id}
                      className="rounded px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 disabled:opacity-50"
                    >
                      {deletingId === template.id ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        'Confirm'
                      )}
                    </button>
                    <button
                      onClick={() => setConfirmDeleteId(null)}
                      className="rounded px-2 py-1 text-xs font-medium text-muted-foreground hover:bg-accent"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => setConfirmDeleteId(template.id)}
                    className="rounded p-1 text-muted-foreground opacity-0 transition-opacity hover:text-red-600 group-hover:opacity-100 dark:hover:text-red-400"
                    title="Delete template"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
