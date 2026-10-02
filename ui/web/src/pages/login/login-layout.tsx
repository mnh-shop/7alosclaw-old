import { getRuntimeBranding } from "@/lib/branding";

interface LoginLayoutProps {
  children: React.ReactNode;
  subtitle?: string;
}

export function LoginLayout({ children, subtitle }: LoginLayoutProps) {
  const branding = getRuntimeBranding();

  return (
    <div className="relative flex min-h-dvh items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 rounded-lg border bg-card p-6 shadow-lg backdrop-blur-xl sm:p-8">
        <div className="text-center">
          <img src={branding.logoUrl} alt={branding.appName} className="mx-auto mb-3 h-20 w-20" />
          <h1 className="text-3xl font-bold tracking-tight">{branding.appName}</h1>
          {subtitle && (
            <p className="mt-2 text-sm text-muted-foreground">{subtitle}</p>
          )}
        </div>
        {children}
      </div>
    </div>
  );
}
