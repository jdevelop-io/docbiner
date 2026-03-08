'use client';

import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { CreditCard, ExternalLink, Loader2 } from 'lucide-react';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth-context';
import { PlanCard, plans } from '@/components/plan-card';
import { UsageTable } from '@/components/usage-table';
import type { PlanInfo } from '@/components/plan-card';
import type { Plan, MonthlyUsage } from '@/lib/types';

interface BillingStatus {
  plan: Plan;
  subscription_status?: string;
  current_period_end?: string;
}

interface CheckoutResponse {
  url: string;
}

interface PortalResponse {
  url: string;
}

export default function BillingPage() {
  const { organization } = useAuth();
  const [upgradingPlan, setUpgradingPlan] = useState<string | null>(null);

  const {
    data: billingStatus,
    isLoading: billingLoading,
  } = useQuery({
    queryKey: ['billing-status'],
    queryFn: () => api.get<BillingStatus>('/v1/billing/status'),
  });

  const {
    data: usageHistory,
    isLoading: historyLoading,
  } = useQuery({
    queryKey: ['usage-history'],
    queryFn: () => api.get<MonthlyUsage[]>('/v1/usage/history'),
  });

  const checkoutMutation = useMutation({
    mutationFn: (priceId: string) =>
      api.post<CheckoutResponse>('/v1/billing/checkout', {
        price_id: priceId,
        success_url: `${window.location.origin}/billing?success=true`,
        cancel_url: `${window.location.origin}/billing?canceled=true`,
      }),
    onSuccess: (data) => {
      window.location.href = data.url;
    },
    onSettled: () => {
      setUpgradingPlan(null);
    },
  });

  const portalMutation = useMutation({
    mutationFn: () =>
      api.post<PortalResponse>('/v1/billing/portal', {
        return_url: `${window.location.origin}/billing`,
      }),
    onSuccess: (data) => {
      window.location.href = data.url;
    },
  });

  function handleUpgrade(plan: PlanInfo) {
    if (!plan.price_id && plan.price > 0) {
      // For plans without a price_id, use the plan name as identifier
      setUpgradingPlan(plan.name);
      checkoutMutation.mutate(plan.name.toLowerCase());
    } else if (plan.price_id) {
      setUpgradingPlan(plan.name);
      checkoutMutation.mutate(plan.price_id);
    }
  }

  const currentPlanName = billingStatus?.plan?.name || organization?.plan_id || 'Free';

  return (
    <div className="mx-auto max-w-6xl space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Billing</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage your subscription plan and view usage history.
          </p>
        </div>
        <button
          onClick={() => portalMutation.mutate()}
          disabled={portalMutation.isPending || currentPlanName === 'Free'}
          className="inline-flex items-center gap-2 rounded-md border border-border bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
        >
          {portalMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ExternalLink className="h-4 w-4" />
          )}
          Manage Billing
        </button>
      </div>

      {/* Current Plan */}
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <CreditCard className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 className="text-base font-semibold">Current Plan</h2>
            <p className="text-sm text-muted-foreground">
              Your active subscription details
            </p>
          </div>
        </div>

        {billingLoading ? (
          <div className="mt-4 h-20 animate-pulse rounded-lg bg-muted" />
        ) : (
          <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <p className="text-sm text-muted-foreground">Plan</p>
              <p className="text-lg font-semibold">{currentPlanName}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Monthly Price</p>
              <p className="text-lg font-semibold">
                ${billingStatus?.plan?.price_monthly ?? 0}/mo
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Monthly Quota</p>
              <p className="text-lg font-semibold">
                {billingStatus?.plan?.monthly_quota?.toLocaleString() ?? '100'} conversions
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              <p className="text-lg font-semibold capitalize">
                {billingStatus?.subscription_status || 'Active'}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Plan Cards */}
      <div>
        <h2 className="mb-4 text-lg font-semibold">Available Plans</h2>
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {plans.map((plan) => (
            <PlanCard
              key={plan.name}
              plan={plan}
              isCurrent={currentPlanName.toLowerCase() === plan.name.toLowerCase()}
              onUpgrade={handleUpgrade}
              isLoading={upgradingPlan === plan.name && checkoutMutation.isPending}
            />
          ))}
        </div>
      </div>

      {/* Usage History */}
      <div>
        <h2 className="mb-4 text-lg font-semibold">Usage History</h2>
        <UsageTable
          data={usageHistory || []}
          isLoading={historyLoading}
        />
      </div>
    </div>
  );
}
