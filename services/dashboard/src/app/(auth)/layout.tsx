export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 px-4">
      <div className="w-full max-w-md space-y-8">
        <div className="flex flex-col items-center gap-2">
          <h1 className="text-3xl font-bold tracking-tight">Docbiner</h1>
          <p className="text-sm text-muted-foreground">
            HTML to PDF &amp; Images
          </p>
        </div>

        <div className="rounded-xl border bg-card p-8 shadow-sm">
          {children}
        </div>
      </div>
    </div>
  );
}
