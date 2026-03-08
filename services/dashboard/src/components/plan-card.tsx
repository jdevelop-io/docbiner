'use client';

import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface PlanInfo {
  name: string;
  price: number;
  quota: number;
  features: string[];
  popular?: boolean;
  price_id?: string;
}

interface PlanCardProps {
  plan: PlanInfo;
  isCurrent: boolean;
  onUpgrade: (plan: PlanInfo) => void;
  isLoading?: boolean;
}

export const plans: PlanInfo[] = [
  {
    name: 'Free',
    price: 0,
    quota: 100,
    features: ['100 conversions/mo', '5MB max file', '15s timeout'],
  },
  {
    name: 'Starter',
    price: 19,
    quota: 2500,
    features: [
      '2,500 conversions/mo',
      '10MB max file',
      '30s timeout',
      'Templates',
      'Custom headers',
    ],
  },
  {
    name: 'Pro',
    price: 49,
    quota: 15000,
    popular: true,
    features: [
      '15,000 conversions/mo',
      '25MB max file',
      '60s timeout',
      'Templates',
      'Webhooks',
      'Priority queue',
    ],
  },
  {
    name: 'Business',
    price: 149,
    quota: 100000,
    features: [
      '100,000 conversions/mo',
      '50MB max file',
      '120s timeout',
      'Everything in Pro',
      'Dedicated support',
    ],
  },
];

export function PlanCard({ plan, isCurrent, onUpgrade, isLoading }: PlanCardProps) {
  return (
    <div
      className={cn(
        'relative flex flex-col rounded-xl border bg-card p-6',
        plan.popular
          ? 'border-primary shadow-md'
          : 'border-border',
      )}
    >
      {plan.popular && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <span className="rounded-full bg-primary px-3 py-1 text-xs font-semibold text-primary-foreground">
            Popular
          </span>
        </div>
      )}

      <div className="mb-4">
        <h3 className="text-lg font-semibold">{plan.name}</h3>
        <div className="mt-2 flex items-baseline gap-1">
          <span className="text-3xl font-bold">
            ${plan.price}
          </span>
          <span className="text-sm text-muted-foreground">/month</span>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          {plan.quota.toLocaleString()} conversions/mo
        </p>
      </div>

      <ul className="mb-6 flex-1 space-y-2.5">
        {plan.features.map((feature) => (
          <li key={feature} className="flex items-start gap-2 text-sm">
            <Check className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
            <span>{feature}</span>
          </li>
        ))}
      </ul>

      {isCurrent ? (
        <div className="rounded-md border border-border bg-muted px-4 py-2 text-center text-sm font-medium text-muted-foreground">
          Current Plan
        </div>
      ) : (
        <button
          onClick={() => onUpgrade(plan)}
          disabled={isLoading}
          className={cn(
            'rounded-md px-4 py-2 text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',
            plan.popular
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'border border-border bg-background hover:bg-accent hover:text-accent-foreground',
          )}
        >
          {isLoading ? 'Redirecting...' : plan.price === 0 ? 'Downgrade' : 'Upgrade'}
        </button>
      )}
    </div>
  );
}
