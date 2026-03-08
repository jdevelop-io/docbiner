'use client';

import { useState } from 'react';
import Link from 'next/link';
import {
  FileText,
  Image,
  Code2,
  Webhook,
  Lock,
  Terminal,
  Check,
  ArrowRight,
} from 'lucide-react';
import { cn } from '@/lib/utils';

/* ------------------------------------------------------------------ */
/*  Hero Section                                                      */
/* ------------------------------------------------------------------ */

function HeroSection() {
  return (
    <section className="relative overflow-hidden">
      {/* Gradient background */}
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-gradient-to-b from-muted/50 to-background" />
        <div className="absolute left-1/2 top-0 h-[600px] w-[900px] -translate-x-1/2 rounded-full bg-gradient-to-r from-primary/5 via-primary/10 to-primary/5 blur-3xl" />
      </div>

      <div className="mx-auto max-w-7xl px-4 pb-20 pt-24 sm:px-6 sm:pt-32 lg:px-8 lg:pt-40">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl lg:text-6xl">
            Convert HTML to PDF &amp; Images{' '}
            <span className="bg-gradient-to-r from-foreground/80 to-foreground bg-clip-text text-transparent">
              at Scale
            </span>
          </h1>
          <p className="mt-6 text-lg leading-8 text-muted-foreground sm:text-xl">
            Production-ready API for high-quality document generation. Templates,
            webhooks, and enterprise features.
          </p>

          <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Link
              href="/register"
              className="inline-flex items-center gap-2 rounded-lg bg-foreground px-6 py-3 text-sm font-semibold text-background shadow-sm transition-colors hover:bg-foreground/90"
            >
              Get Started Free
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              href="/docs"
              className="inline-flex items-center gap-2 rounded-lg border border-border px-6 py-3 text-sm font-semibold transition-colors hover:bg-accent"
            >
              View Docs
            </Link>
          </div>
        </div>

        {/* Code snippet */}
        <div className="mx-auto mt-16 max-w-2xl">
          <div className="overflow-hidden rounded-xl border border-border bg-card shadow-lg">
            <div className="flex items-center gap-2 border-b border-border bg-muted/50 px-4 py-3">
              <div className="h-3 w-3 rounded-full bg-red-400/60" />
              <div className="h-3 w-3 rounded-full bg-yellow-400/60" />
              <div className="h-3 w-3 rounded-full bg-green-400/60" />
              <span className="ml-2 text-xs text-muted-foreground">Terminal</span>
            </div>
            <pre className="overflow-x-auto p-4 text-sm leading-relaxed">
              <code className="font-mono text-foreground/90">
                <span className="text-muted-foreground">$</span> curl -X POST https://api.docbiner.com/v1/convert \{'\n'}
                {'  '}-H &quot;Authorization: Bearer db_live_...&quot; \{'\n'}
                {'  '}-H &quot;Content-Type: application/json&quot; \{'\n'}
                {'  '}-d &apos;{'{'}&quot;source&quot;: &quot;&lt;h1&gt;Hello World&lt;/h1&gt;&quot;, &quot;format&quot;: &quot;pdf&quot;{'}'}&apos;
              </code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Features Section                                                  */
/* ------------------------------------------------------------------ */

const features = [
  {
    icon: FileText,
    title: 'Pixel-Perfect PDFs',
    description:
      'Powered by Chromium rendering for accurate, production-quality PDF output every time.',
  },
  {
    icon: Image,
    title: 'Multiple Formats',
    description:
      'Generate PDF, PNG, JPEG, and WebP outputs from a single API endpoint.',
  },
  {
    icon: Code2,
    title: 'Template Engine',
    description:
      'Use Handlebars or Liquid templates with dynamic data for reusable document generation.',
  },
  {
    icon: Webhook,
    title: 'Async & Webhooks',
    description:
      'Queue conversions and receive results via webhooks. Scale without waiting.',
  },
  {
    icon: Lock,
    title: 'Encryption & Merge',
    description:
      'Password-protect PDFs and merge multiple documents into one. Security built-in.',
  },
  {
    icon: Terminal,
    title: 'SDKs & CLI',
    description:
      'Official Node.js and Python SDKs plus a CLI tool for quick integrations.',
  },
];

function FeaturesSection() {
  return (
    <section id="features" className="scroll-mt-20 bg-muted/30 py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Everything you need for document generation
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            A complete platform for converting HTML to production-ready documents.
          </p>
        </div>

        <div className="mt-16 grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="group rounded-xl border border-border bg-card p-6 transition-shadow hover:shadow-md"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <feature.icon className="h-5 w-5 text-foreground" />
              </div>
              <h3 className="mt-4 text-base font-semibold">{feature.title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Code Examples Section                                             */
/* ------------------------------------------------------------------ */

const codeExamples: { label: string; language: string; code: string }[] = [
  {
    label: 'Node.js',
    language: 'javascript',
    code: `const docbiner = new Docbiner({ apiKey: 'db_live_...' });

const pdf = await docbiner.convert({
  source: '<h1>Hello World</h1>',
  format: 'pdf'
});

// Save to file
fs.writeFileSync('output.pdf', pdf);`,
  },
  {
    label: 'Python',
    language: 'python',
    code: `from docbiner import Docbiner

client = Docbiner(api_key="db_live_...")

pdf = client.convert(
    source="<h1>Hello World</h1>",
    format="pdf"
)

# Save to file
with open("output.pdf", "wb") as f:
    f.write(pdf)`,
  },
  {
    label: 'cURL',
    language: 'bash',
    code: `curl -X POST https://api.docbiner.com/v1/convert \\
  -H "Authorization: Bearer db_live_..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "source": "<h1>Hello World</h1>",
    "format": "pdf"
  }' \\
  --output output.pdf`,
  },
];

function CodeExamplesSection() {
  const [activeTab, setActiveTab] = useState(0);

  return (
    <section className="py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Simple, powerful API
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Get started in minutes with your language of choice.
          </p>
        </div>

        <div className="mx-auto mt-12 max-w-3xl">
          <div className="overflow-hidden rounded-xl border border-border bg-card shadow-lg">
            {/* Tabs */}
            <div className="flex border-b border-border bg-muted/50">
              {codeExamples.map((example, index) => (
                <button
                  key={example.label}
                  onClick={() => setActiveTab(index)}
                  className={cn(
                    'px-5 py-3 text-sm font-medium transition-colors',
                    activeTab === index
                      ? 'border-b-2 border-foreground bg-card text-foreground'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {example.label}
                </button>
              ))}
            </div>

            {/* Code */}
            <pre className="overflow-x-auto p-6 text-sm leading-relaxed">
              <code className="font-mono text-foreground/90">
                {codeExamples[activeTab].code}
              </code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Pricing Preview Section                                           */
/* ------------------------------------------------------------------ */

const previewPlans = [
  {
    name: 'Free',
    price: 0,
    quota: '100',
    features: ['100 conversions/mo', '5MB max file', '15s timeout'],
  },
  {
    name: 'Starter',
    price: 19,
    quota: '2,500',
    features: ['2,500 conversions/mo', '10MB max file', '30s timeout', 'Templates', 'Custom headers'],
  },
  {
    name: 'Pro',
    price: 49,
    quota: '15,000',
    popular: true,
    features: ['15,000 conversions/mo', '25MB max file', '60s timeout', 'Templates', 'Webhooks', 'Priority queue'],
  },
  {
    name: 'Business',
    price: 149,
    quota: '100,000',
    features: ['100,000 conversions/mo', '50MB max file', '120s timeout', 'Everything in Pro', 'Dedicated support'],
  },
];

function PricingPreviewSection() {
  return (
    <section className="bg-muted/30 py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Simple, transparent pricing
          </h2>
          <p className="mt-4 text-lg text-muted-foreground">
            Start free, scale as you grow. No hidden fees.
          </p>
        </div>

        <div className="mt-16 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {previewPlans.map((plan) => (
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
                    Popular
                  </span>
                </div>
              )}

              <div className="mb-4">
                <h3 className="text-lg font-semibold">{plan.name}</h3>
                <div className="mt-2 flex items-baseline gap-1">
                  <span className="text-3xl font-bold">${plan.price}</span>
                  <span className="text-sm text-muted-foreground">/month</span>
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
                  'rounded-md px-4 py-2 text-center text-sm font-medium transition-colors',
                  plan.popular
                    ? 'bg-foreground text-background hover:bg-foreground/90'
                    : 'border border-border bg-background hover:bg-accent hover:text-accent-foreground',
                )}
              >
                Get Started
              </Link>
            </div>
          ))}
        </div>

        <div className="mt-10 text-center">
          <Link
            href="/pricing"
            className="inline-flex items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
          >
            View full pricing details
            <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  CTA Banner Section                                                */
/* ------------------------------------------------------------------ */

function CtaBanner() {
  return (
    <section className="py-24">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="relative overflow-hidden rounded-2xl bg-foreground px-8 py-16 text-center shadow-lg sm:px-16">
          <h2 className="text-3xl font-bold tracking-tight text-background sm:text-4xl">
            Ready to get started?
          </h2>
          <p className="mx-auto mt-4 max-w-xl text-lg text-background/70">
            Start converting documents in minutes. Free plan includes 100
            conversions per month.
          </p>
          <div className="mt-8 flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Link
              href="/register"
              className="inline-flex items-center gap-2 rounded-lg bg-background px-6 py-3 text-sm font-semibold text-foreground shadow-sm transition-colors hover:bg-background/90"
            >
              Create Free Account
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              href="/docs"
              className="inline-flex items-center gap-2 rounded-lg border border-background/20 px-6 py-3 text-sm font-semibold text-background transition-colors hover:bg-background/10"
            >
              Read the Docs
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Page                                                              */
/* ------------------------------------------------------------------ */

export default function LandingPage() {
  return (
    <>
      <HeroSection />
      <FeaturesSection />
      <CodeExamplesSection />
      <PricingPreviewSection />
      <CtaBanner />
    </>
  );
}
