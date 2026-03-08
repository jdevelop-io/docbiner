'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Dialog } from 'radix-ui';
import { X, Mail } from 'lucide-react';
import { api } from '@/lib/api';
import type { MemberRole } from '@/lib/types';

interface MemberInviteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MemberInviteDialog({ open, onOpenChange }: MemberInviteDialogProps) {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<Exclude<MemberRole, 'owner'>>('member');
  const [success, setSuccess] = useState(false);
  const queryClient = useQueryClient();

  const inviteMutation = useMutation({
    mutationFn: (data: { email: string; role: string }) =>
      api.post<{ message: string }>('/v1/organization/members/invite', data),
    onSuccess: () => {
      setSuccess(true);
      queryClient.invalidateQueries({ queryKey: ['members'] });
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!email.trim()) return;
    inviteMutation.mutate({ email: email.trim(), role });
  }

  function handleClose(isOpen: boolean) {
    if (!isOpen) {
      setEmail('');
      setRole('member');
      setSuccess(false);
      inviteMutation.reset();
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
              Invite Member
            </Dialog.Title>
            <Dialog.Close className="rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors">
              <X className="h-4 w-4" />
            </Dialog.Close>
          </div>

          {success ? (
            <div className="space-y-4">
              <div className="flex items-start gap-3 rounded-lg border border-emerald-200 bg-emerald-50 p-3 dark:border-emerald-900/50 dark:bg-emerald-900/20">
                <Mail className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
                <p className="text-sm text-emerald-800 dark:text-emerald-300">
                  Invitation sent to <strong>{email}</strong>. They will receive
                  an email with instructions to join.
                </p>
              </div>
              <button
                onClick={() => handleClose(false)}
                className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                Done
              </button>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="invite-email" className="text-sm font-medium">
                  Email address
                </label>
                <input
                  id="invite-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="colleague@example.com"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  required
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="invite-role" className="text-sm font-medium">
                  Role
                </label>
                <select
                  id="invite-role"
                  value={role}
                  onChange={(e) => setRole(e.target.value as Exclude<MemberRole, 'owner'>)}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
                >
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                </select>
                <p className="text-xs text-muted-foreground">
                  Admins can manage members and settings. Members can use the API and view jobs.
                </p>
              </div>

              {inviteMutation.isError && (
                <p className="text-sm text-destructive">
                  {inviteMutation.error?.message || 'Failed to send invitation.'}
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
                  disabled={inviteMutation.isPending || !email.trim()}
                  className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
                >
                  {inviteMutation.isPending ? 'Sending...' : 'Send Invite'}
                </button>
              </div>
            </form>
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
