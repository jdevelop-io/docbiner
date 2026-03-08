'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Users, Loader2, CheckCircle } from 'lucide-react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth-context';
import { api } from '@/lib/api';

function SaveFeedback({ show }: { show: boolean }) {
  if (!show) return null;
  return (
    <span className="inline-flex items-center gap-1 text-sm text-emerald-600 dark:text-emerald-400">
      <CheckCircle className="h-3.5 w-3.5" />
      Saved
    </span>
  );
}

export default function SettingsPage() {
  const { user, organization } = useAuth();

  const [displayName, setDisplayName] = useState(user?.display_name || '');
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url || '');
  const [orgName, setOrgName] = useState(organization?.name || '');
  const [profileSaved, setProfileSaved] = useState(false);
  const [orgSaved, setOrgSaved] = useState(false);

  const profileMutation = useMutation({
    mutationFn: (data: { display_name: string; avatar_url: string }) =>
      api.put('/v1/auth/profile', data),
    onSuccess: () => {
      setProfileSaved(true);
      setTimeout(() => setProfileSaved(false), 3000);
    },
  });

  const orgMutation = useMutation({
    mutationFn: (data: { name: string }) =>
      api.put('/v1/organization', data),
    onSuccess: () => {
      setOrgSaved(true);
      setTimeout(() => setOrgSaved(false), 3000);
    },
  });

  function handleProfileSubmit(e: React.FormEvent) {
    e.preventDefault();
    profileMutation.mutate({
      display_name: displayName.trim(),
      avatar_url: avatarUrl.trim(),
    });
  }

  function handleOrgSubmit(e: React.FormEvent) {
    e.preventDefault();
    orgMutation.mutate({ name: orgName.trim() });
  }

  return (
    <div className="mx-auto max-w-2xl space-y-8">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your profile and organization settings.
        </p>
      </div>

      {/* Profile Settings */}
      <div className="rounded-xl border border-border bg-card">
        <div className="border-b border-border px-6 py-4">
          <h2 className="text-base font-semibold">Profile</h2>
          <p className="text-sm text-muted-foreground">
            Your personal account information.
          </p>
        </div>
        <form onSubmit={handleProfileSubmit} className="space-y-4 p-6">
          <div className="space-y-2">
            <label htmlFor="display-name" className="text-sm font-medium">
              Display name
            </label>
            <input
              id="display-name"
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Your name"
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium">
              Email
            </label>
            <input
              id="email"
              type="email"
              value={user?.email || ''}
              disabled
              className="w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-muted-foreground cursor-not-allowed"
            />
            <p className="text-xs text-muted-foreground">
              Your email address cannot be changed.
            </p>
          </div>

          <div className="space-y-2">
            <label htmlFor="avatar-url" className="text-sm font-medium">
              Avatar URL
            </label>
            <input
              id="avatar-url"
              type="url"
              value={avatarUrl}
              onChange={(e) => setAvatarUrl(e.target.value)}
              placeholder="https://example.com/avatar.png"
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          {profileMutation.isError && (
            <p className="text-sm text-destructive">
              {profileMutation.error?.message || 'Failed to update profile.'}
            </p>
          )}

          <div className="flex items-center gap-3 pt-2">
            <button
              type="submit"
              disabled={profileMutation.isPending}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {profileMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Save
            </button>
            <SaveFeedback show={profileSaved} />
          </div>
        </form>
      </div>

      {/* Organization Settings */}
      <div className="rounded-xl border border-border bg-card">
        <div className="border-b border-border px-6 py-4">
          <h2 className="text-base font-semibold">Organization</h2>
          <p className="text-sm text-muted-foreground">
            Settings for your organization.
          </p>
        </div>
        <form onSubmit={handleOrgSubmit} className="space-y-4 p-6">
          <div className="space-y-2">
            <label htmlFor="org-name" className="text-sm font-medium">
              Organization name
            </label>
            <input
              id="org-name"
              type="text"
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              placeholder="My Organization"
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="org-slug" className="text-sm font-medium">
              Slug
            </label>
            <input
              id="org-slug"
              type="text"
              value={organization?.slug || ''}
              disabled
              className="w-full rounded-md border border-input bg-muted px-3 py-2 text-sm text-muted-foreground cursor-not-allowed"
            />
            <p className="text-xs text-muted-foreground">
              The organization slug cannot be changed.
            </p>
          </div>

          {orgMutation.isError && (
            <p className="text-sm text-destructive">
              {orgMutation.error?.message ||
                'Failed to update organization.'}
            </p>
          )}

          <div className="flex items-center gap-3 pt-2">
            <button
              type="submit"
              disabled={orgMutation.isPending}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              {orgMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Save
            </button>
            <SaveFeedback show={orgSaved} />
          </div>
        </form>
      </div>

      {/* Members link */}
      <div className="rounded-xl border border-border bg-card">
        <div className="flex items-center justify-between px-6 py-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
              <Users className="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <h2 className="text-base font-semibold">Members</h2>
              <p className="text-sm text-muted-foreground">
                Invite and manage team members.
              </p>
            </div>
          </div>
          <Link
            href="/settings/members"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium hover:bg-muted transition-colors"
          >
            Manage Members
          </Link>
        </div>
      </div>
    </div>
  );
}
