'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Check, Minus, ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';

/* ------------------------------------------------------------------ */
/*  Plan data                                                         */
/* ------------------------------------------------------------------ */

interface PricingPlan {
  name: string;
  description: string;
  priceMonthly: number;
  priceYearly: number;
  quota: string;
  popular?: boolean;
  features: string[];
}

const pricingPlans: PricingPlan[] = [
  {
    name: 'Free',
    description: 'Perfect for testing and personal projects.',
    priceMonthly: 0,
    priceYearly: 0,
    quota: '100',
    features: [
      '100 conversions/mo',
      '5MB max file size',
      '15s timeout',
      'PDF & image output',
      'Community support',
    ],
  },
  {
    name: 'Starter',
    description: 'For small teams and growing applications.',
    priceMonthly: 19,
    priceYearly: 190,
    quota: '2,500',
    features: [
      '2,500 conversions/mo',
      '10MB max file size',
      '30s timeout',
      'PDF & image output',
      'Templates (Handlebars & Liquid)',
      'Custom headers & cookies',
      'Email support',
    ],
  },
  {
    name: 'Pro',
    description: 'For production workloads and scaling teams.',
    priceMonthly: 49,
    priceYearly: 490,
    quota: '15,000',
    popular: true,
    features: [
      '15,000 conversions/mo',
      '25MB max file size',
      '60s timeout',
      'PDF & image output',
      'Templates (Handlebars & Liquid)',
      'Custom headers & cookies',
      'Webhooks & async jobs',
      'Priority queue',
      'PDF encryption',
      'PDF merge',
      'Priority support',
    ],
  },
  {
    name: 'Business',
    description: 'For high-volume and enterprise use cases.',
    priceMonthly: 149,
    priceYearly: 1490,
    quota: '100,000',
    features: [
      '100,000 conversions/mo',
      '50MB max file size',
      '120s timeout',
      'Everything in Pro',
      'Dedicated support',
      'Custom SLA',
      'Overage at $0.001/conversion',
    ],
  },
];

/* ------------------------------------------------------------------ */
/*  Feature comparison table data                                     */
/* ------------------------------------------------------------------ */

interface ComparisonRow {
  feature: string;
  free: string | boolean;
  starter: string | boolean;
  pro: string | boolean;
  business: string | boolean;
}

const comparisonRows: ComparisonRow[] = [
  { feature: 'Monthly conversions', free: '100', starter: '2,500', pro: '15,000', business: '100,000' },
  { feature: 'Max file size', free: '5MB', starter: '10MB', pro: '25MB', business: '50MB' },
  { feature: 'Timeout', free: '15s', starter: '30s', pro: '60s', business: '120s' },
  { feature: 'PDF output', free: true, starter: true, pro: true, business: true },
  { feature: 'Image output (PNG, JPEG, WebP)', free: true, starter: true, pro: true, business: true },
  { feature: 'Templates', free: false, starter: true, pro: true, business: true },
  { feature: 'Custom headers & cookies', free: false, starter: true, pro: true, business: true },
  { feature: 'Webhooks', free: false, starter: false, pro: true, business: true },
  { feature: 'Async jobs', free: false, starter: false, pro: true, business: true },
  { feature: 'Priority queue', free: false, starter: false, pro: true, business: true },
  { feature: 'PDF encryption', free: false, starter: false, pro: true, business: true },
  { feature: 'PDF merge', free: false, starter: false, pro: true, business: true },
  { feature: 'Dedicated support', free: false, starter: false, pro: false, business: true },
  { feature: 'Custom SLA', free: false, starter: false, pro: false, business: true },
];

/* ------------------------------------------------------------------ */
/*  FAQ data                                                          */
/* ------------------------------------------------------------------ */

interface FaqItem {
  question: string;
  answer: string;
}

const faqs: FaqItem[] = [
  {
    question: 'What counts as a conversion?',
    answer:
      'Each API call to the /v1/convert or /v1/convert/async endpoint counts as one conversion. Merging multiple PDFs via /v1/merge also counts as one conversion. Test conversions (using test API keys) do not count toward your quota.',
  },
  {
    question: 'What happens when I exceed my monthly quota?',
    answer:
      'On the Free plan, additional conversions are blocked until the next billing cycle. On paid plans (Starter, Pro, Business), overage conversions are billed at $0.005, $0.003, and $0.001 per conversion respectively.',
  },
  {
    question: 'Can I switch plans at any time?',
    answer:
      'Yes. Upgrades take effect immediately and you are billed the prorated difference. Downgrades take effect at the start of your next billing cycle.',
  },
  {
    question: 'Is there a discount for yearly billing?',
    answer:
      'Yes, yearly billing gives you two months free compared to monthly billing. You can switch between monthly and yearly at any time from your billing settings.',
  },
  {
    question: 'Do you offer a free trial for paid plans?',
    answer:
      'We offer a generous free tier with 100 conversions per month. This lets you fully test the API before committing to a paid plan. No credit card required.',
  },
  {
    question: 'What payment methods do you accept?',
    answer:
      'We accept all major credit and debit cards (Visa, Mastercard, American Express) via Stripe. Invoicing is available for Business plan customers on yearly billing.',
  },
];

/* ------------------------------------------------------------------ */
/*  FAQ Accordion Item                                                */
/* ------------------------------------------------------------------ */

function FaqAccordionItem({ item }: { item: FaqItem }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="border-b border-border">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between py-5 text-left"
      >
        <span className="text-sm font-medium">{item.question}</span>
        <ChevronDown
          className={cn(
            'h-4 w-4 shrink-0 text-muted-foreground transition-transform',
            open && 'rotate-180',
          )}
        />
      </button>
      {open && (
        <div className="pb-5 text-sm leading-relaxed text-muted-foreground">
          {item.answer}
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Cell renderer for comparison table                                */
/* ------------------------------------------------------------------ */

function ComparisonCell({ value }: { value: string | boolean }) {
  if (typeof value === 'string') {
    return <span className="text-sm">{value}</span>;
  }
  if (value) {
    return <Check className="mx-auto h-4 w-4 text-foreground" />;
  }
  return <Minus className="mx-auto h-4 w-4 text-muted-foreground/40" />;
}

/* ------------------------------------------------------------------ */
/*  Page                                                              */
/* ------------------------------------------------------------------ */

export default function PricingPage() {
  const [yearly, setYearly] = useState(false);

  return (
    <div className="py-16 sm:py-24">
      {/* Header */}
      <div className="mx-auto max-w-7xl px-4 text-center sm:px-6 lg:px-8">
        <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl">
          Pricing
        </h1>
        <p className="mx-auto mt-4 max-w-xl text-lg text-muted-foreground">
          Start free, scale as you grow. All plans include core conversion features.
        </p>

        {/* Monthly / Yearly toggle */}
        <div className="mt-8 flex items-center justify-center gap-3">
          <span
            className={cn(
              'text-sm font-medium',
              !yearly ? 'text-foreground' : 'text-muted-foreground',
            )}
          >
            Monthly
          </span>
          <button
            onClick={() => setYearly(!yearly)}
            className={cn(
              'relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors',
              yearly ? 'bg-foreground' : 'bg-muted-foreground/30',
            )}
            role="switch"
            aria-checked={yearly}
          >
            <span
              className={cn(
                'pointer-events-none inline-block h-5 w-5 rounded-full bg-background shadow-sm ring-0 transition-transform',
                yearly ? 'translate-x-5' : 'translate-x-0',
              )}
            />
          </button>
          <span
            className={cn(
              'text-sm font-medium',
              yearly ? 'text-foreground' : 'text-muted-foreground',
            )}
          >
            Yearly
            <span className="ml-1.5 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-semibold text-foreground">
              2 months free
            </span>
          </span>
        </div>
      </div>

      {/* Plan cards */}
      <div className="mx-auto mt-16 max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {pricingPlans.map((plan) => {
            const price = yearly ? plan.priceYearly : plan.priceMonthly;
            const period = yearly ? '/year' : '/month';

            return (
              <div
                key={plan.name}
                className={cn(
                  'relative flex flex-col rounded-xl border bg-card p-6',
                  plan.popular ? 'border-foreground shadow-md' : 'border-border',
                )}
              >
                {plan.popular && (
                  <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                    <span className="rounded-full bg-foreground px-3 py-1 text-xs font-semibold text-background">
                      Most Popular
                    </span>
                  </div>
                )}

                <div className="mb-4">
                  <h3 className="text-lg font-semibold">{plan.name}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {plan.description}
                  </p>
                  <div className="mt-4 flex items-baseline gap-1">
                    <span className="text-4xl font-bold">${price}</span>
                    <span className="text-sm text-muted-foreground">{period}</span>
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {plan.quota} conversions/mo
                  </p>
                </div>

                <ul className="mb-6 flex-1 space-y-2.5">
                  {plan.features.map((feature) => (
                    <li key={feature} className="flex items-start gap-2 text-sm">
                      <Check className="mt-0.5 h-4 w-4 shrink-0 text-foreground" />
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>

                <Link
                  href="/register"
                  className={cn(
                    'rounded-md px-4 py-2.5 text-center text-sm font-medium transition-colors',
                    plan.popular
                      ? 'bg-foreground text-background hover:bg-foreground/90'
                      : 'border border-border bg-background hover:bg-accent hover:text-accent-foreground',
                  )}
                >
                  {plan.priceMonthly === 0 ? 'Get Started Free' : 'Start Free Trial'}
                </Link>
              </div>
            );
          })}
        </div>
      </div>

      {/* Feature comparison table */}
      <div className="mx-auto mt-24 max-w-7xl px-4 sm:px-6 lg:px-8">
        <h2 className="text-center text-2xl font-bold tracking-tight sm:text-3xl">
          Feature comparison
        </h2>

        <div className="mt-12 overflow-x-auto">
          <table className="w-full min-w-[640px] text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="pb-4 pr-4 text-left font-medium text-muted-foreground">
                  Feature
                </th>
                <th className="pb-4 px-4 text-center font-medium">Free</th>
                <th className="pb-4 px-4 text-center font-medium">Starter</th>
                <th className="pb-4 px-4 text-center font-medium">Pro</th>
                <th className="pb-4 pl-4 text-center font-medium">Business</th>
              </tr>
            </thead>
            <tbody>
              {comparisonRows.map((row) => (
                <tr key={row.feature} className="border-b border-border/50">
                  <td className="py-3.5 pr-4 text-sm">{row.feature}</td>
                  <td className="py-3.5 px-4 text-center">
                    <ComparisonCell value={row.free} />
                  </td>
                  <td className="py-3.5 px-4 text-center">
                    <ComparisonCell value={row.starter} />
                  </td>
                  <td className="py-3.5 px-4 text-center">
                    <ComparisonCell value={row.pro} />
                  </td>
                  <td className="py-3.5 pl-4 text-center">
                    <ComparisonCell value={row.business} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* FAQ */}
      <div className="mx-auto mt-24 max-w-3xl px-4 sm:px-6 lg:px-8">
        <h2 className="text-center text-2xl font-bold tracking-tight sm:text-3xl">
          Frequently asked questions
        </h2>
        <div className="mt-10">
          {faqs.map((faq) => (
            <FaqAccordionItem key={faq.question} item={faq} />
          ))}
        </div>
      </div>
    </div>
  );
}
