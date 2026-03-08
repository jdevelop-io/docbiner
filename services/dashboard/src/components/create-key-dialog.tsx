'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Dialog } from 'radix-ui';
import { Copy, X, AlertTriangle } from 'lucide-react';
import { api } from '@/lib/api';
import type { ApiKey } from '@/lib/types';

interface CreateKeyResponse extends ApiKey {
  full_key: string;
}

interface CreateKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text);
}

export function CreateKeyDialog({ open, onOpenChange }: CreateKeyDialogProps) {
  const [name, setName] = useState('');
  const [environment, setEnvironment] = useState<'live' | 'test'>('test');
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (data: { name: string; environment: 'live' | 'test' }) =>
      api.post<CreateKeyResponse>('/v1/api-keys', data),
    onSuccess: (data) => {
      setCreatedKey(data.full_key);
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });

  function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    createMutation.mutate({ name: name.trim(), environment });
  }

  function handleCopy() {
    if (createdKey) {
      copyToClipboard(createdKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  function handleClose(isOpen: boolean) {
    if (!isOpen) {
      setName('');
      setEnvironment('test');
      setCreatedKey(null);
      setCopied(false);
      createMutation.reset();
    }
    onOpenChange(isOpen);
  }

  return (
    <Dialog.Root open={open} onOpenChange={handleClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-6 shadow-lg data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95">
          <div className="flex items-center justify-between mb-4">
            <Dialog.Title className="text-lg font-semibold">
              {createdKey ? 'API Key Created' : 'Create API Key'}
            </Dialog.Title>
            <Dialog.Close className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors">
              <X className="h-4 w-4" />
            </Dialog.Close>
          </div>

          {createdKey ? (
            <div className="space-y-4">
              <div className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/50 dark:bg-amber-900/20">
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" />
                <p className="text-sm text-amber-800 dark:text-amber-300">
                  Save this key. You won&apos;t be able to see it again.
                </p>
              </div>

              <div className="flex items-center gap-2">
                <code className="flex-1 overflow-x-auto rounded-md border border-border bg-muted px-3 py-2 font-mono text-sm">
                  {createdKey}
                </code>
                <button
                  onClick={handleCopy}
                  className="shrink-0 rounded-md border border-border px-3 py-2 text-sm font-medium hover:bg-muted transition-colors"
                >
                  {copied ? 'Copied!' : <Copy className="h-4 w-4" />}
                </button>
              </div>

              <button
                onClick={() => handleClose(false)}
                className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                Done
              </button>
            </div>
          ) : (
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="key-name" className="text-sm font-medium">
                  Name
                </label>
                <input
                  id="key-name"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g. Production API Key"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  required
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="key-environment" className="text-sm font-medium">
                  Environment
                </label>
                <select
                  id="key-environment"
                  value={environment}
                  onChange={(e) => setEnvironment(e.target.value as 'live' | 'test')}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                >
                  <option value="test">Test</option>
                  <option value="live">Live</option>
                </select>
              </div>

              {createMutation.isError && (
                <p className="text-sm text-destructive">
                  {createMutation.error?.message || 'Failed to create API key.'}
                </p>
              )}

              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => handleClose(false)}
                  className="rounded-md border border-border px-4 py-2 text-sm font-medium hover:bg-muted transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || !name.trim()}
                  className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {createMutation.isPending ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
