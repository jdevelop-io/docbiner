'use client';

import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Plus, Loader2, Users, Trash2 } from 'lucide-react';
import Link from 'next/link';
import { api } from '@/lib/api';
import type { Member, MemberRole } from '@/lib/types';
import { MemberInviteDialog } from '@/components/member-invite';

function RoleBadge({ role }: { role: MemberRole }) {
  const styles: Record<MemberRole, string> = {
    owner:
      'bg-purple-50 text-purple-700 dark:bg-purple-950 dark:text-purple-400',
    admin:
      'bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-400',
    member:
      'bg-gray-50 text-gray-700 dark:bg-gray-950 dark:text-gray-400',
  };

  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium capitalize ${styles[role]}`}
    >
      {role}
    </span>
  );
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export default function MembersPage() {
  const [inviteOpen, setInviteOpen] = useState(false);
  const [confirmRemoveId, setConfirmRemoveId] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data: members = [], isLoading, error } = useQuery({
    queryKey: ['members'],
    queryFn: () => api.get<Member[]>('/v1/organization/members'),
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ memberId, role }: { memberId: string; role: MemberRole }) =>
      api.put(`/v1/organization/members/${memberId}/role`, { role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['members'] });
    },
  });

  const removeMutation = useMutation({
    mutationFn: (memberId: string) =>
      api.delete(`/v1/organization/members/${memberId}`),
    onSuccess: () => {
      setConfirmRemoveId(null);
      queryClient.invalidateQueries({ queryKey: ['members'] });
    },
  });

  const handleRoleChange = useCallback(
    (memberId: string, newRole: string) => {
      updateRoleMutation.mutate({ memberId, role: newRole as MemberRole });
    },
    [updateRoleMutation],
  );

  const handleRemove = useCallback(
    (memberId: string) => {
      removeMutation.mutate(memberId);
    },
    [removeMutation],
  );

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href="/settings"
            className="rounded-md p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">Members</h1>
            <p className="text-sm text-muted-foreground">
              Manage your organization members and invitations.
            </p>
          </div>
        </div>
        <button
          onClick={() => setInviteOpen(true)}
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Plus className="h-4 w-4" />
          Invite Member
        </button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/10">
          <p className="text-sm text-red-700 dark:text-red-400">
            Failed to load members. Please try again.
          </p>
        </div>
      ) : members.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-card py-12 text-center">
          <Users className="h-10 w-10 text-muted-foreground/50" />
          <p className="mt-4 text-sm text-muted-foreground">
            No members yet. Invite someone to get started.
          </p>
        </div>
      ) : (
        <div className="rounded-xl border border-border bg-card">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-muted-foreground">
                  <th className="px-5 py-3 font-medium">Member</th>
                  <th className="px-5 py-3 font-medium">Role</th>
                  <th className="px-5 py-3 font-medium">Joined</th>
                  <th className="px-5 py-3 font-medium">
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {members.map((member) => (
                  <tr
                    key={member.id}
                    className="border-b border-border last:border-0"
                  >
                    <td className="px-5 py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-sm font-medium">
                          {member.display_name?.charAt(0).toUpperCase() ||
                            member.email.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <p className="font-medium">
                            {member.display_name || 'Unnamed'}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {member.email}
                          </p>
                        </div>
                      </div>
                    </td>
                    <td className="px-5 py-4">
                      {member.role === 'owner' ? (
                        <RoleBadge role={member.role} />
                      ) : (
                        <select
                          value={member.role}
                          onChange={(e) =>
                            handleRoleChange(member.id, e.target.value)
                          }
                          disabled={updateRoleMutation.isPending}
                          className="rounded-md border border-input bg-background px-2 py-1 text-xs font-medium focus:outline-none focus:ring-2 focus:ring-ring disabled:opacity-50"
                        >
                          <option value="admin">Admin</option>
                          <option value="member">Member</option>
                        </select>
                      )}
                    </td>
                    <td className="px-5 py-4 text-muted-foreground">
                      {formatDate(member.joined_at)}
                    </td>
                    <td className="px-5 py-4">
                      {member.role !== 'owner' && (
                        <>
                          {confirmRemoveId === member.id ? (
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => handleRemove(member.id)}
                                disabled={removeMutation.isPending}
                                className="rounded-md bg-destructive px-3 py-1 text-xs font-medium text-destructive-foreground hover:bg-destructive/90 transition-colors disabled:opacity-50"
                              >
                                {removeMutation.isPending
                                  ? 'Removing...'
                                  : 'Confirm'}
                              </button>
                              <button
                                onClick={() => setConfirmRemoveId(null)}
                                className="rounded-md border border-border px-3 py-1 text-xs font-medium hover:bg-muted transition-colors"
                              >
                                Cancel
                              </button>
                            </div>
                          ) : (
                            <button
                              onClick={() => setConfirmRemoveId(member.id)}
                              className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                              title="Remove member"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          )}
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {(updateRoleMutation.isError || removeMutation.isError) && (
            <div className="border-t border-border px-5 py-3">
              <p className="text-sm text-destructive">
                {updateRoleMutation.error?.message ||
                  removeMutation.error?.message ||
                  'An error occurred. Please try again.'}
              </p>
            </div>
          )}
        </div>
      )}

      <MemberInviteDialog open={inviteOpen} onOpenChange={setInviteOpen} />
    </div>
  );
}
