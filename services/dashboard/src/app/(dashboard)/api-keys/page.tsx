'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { api } from '@/lib/api';
import type { ApiKey } from '@/lib/types';
import { ApiKeyTable } from '@/components/api-key-table';
import { CreateKeyDialog } from '@/components/create-key-dialog';

export default function ApiKeysPage() {
  const [dialogOpen, setDialogOpen] = useState(false);

  const { data: keys = [], isLoading, error } = useQuery({
    queryKey: ['api-keys'],
    queryFn: () => api.get<ApiKey[]>('/v1/api-keys'),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">API Keys</h1>
          <p className="text-muted-foreground">
            Manage your API keys for accessing the Docbiner API.
          </p>
        </div>
        <button
          onClick={() => setDialogOpen(true)}
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Create Key
        </button>
      </div>

      {isLoading ? (
        <div className="rounded-lg border border-border bg-card p-12 text-center">
          <p className="text-muted-foreground">Loading API keys...</p>
        </div>
      ) : error ? (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-12 text-center">
          <p className="text-destructive">
            Failed to load API keys. Please try again.
          </p>
        </div>
      ) : (
        <ApiKeyTable keys={keys} />
      )}

      <CreateKeyDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  );
}
