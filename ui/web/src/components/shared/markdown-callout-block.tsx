/**
 * CalloutBlock — styled callout/admonition block rendered from [!type] blockquotes.
 * Supports: note, info, warning, tip, danger, important
 * Colours come from the semantic status tokens; the console is dark-only.
 */
import { Info, AlertTriangle, Lightbulb, AlertOctagon, Star } from "lucide-react";
import type { CalloutType } from "@/lib/remark-callouts";

interface CalloutConfig {
  icon: React.ElementType;
  borderColor: string;
  bgColor: string;
  titleColor: string;
  iconColor: string;
}

const CALLOUT_CONFIG: Record<CalloutType, CalloutConfig> = {
  note: {
    icon: Info,
    borderColor: "border-info/40",
    bgColor: "bg-info/10",
    titleColor: "text-info",
    iconColor: "text-info",
  },
  info: {
    icon: Info,
    borderColor: "border-info/40",
    bgColor: "bg-info/10",
    titleColor: "text-info",
    iconColor: "text-info",
  },
  warning: {
    icon: AlertTriangle,
    borderColor: "border-warning/40",
    bgColor: "bg-warning/10",
    titleColor: "text-warning",
    iconColor: "text-warning",
  },
  tip: {
    icon: Lightbulb,
    borderColor: "border-success/40",
    bgColor: "bg-success/10",
    titleColor: "text-success",
    iconColor: "text-success",
  },
  danger: {
    icon: AlertOctagon,
    borderColor: "border-destructive/40",
    bgColor: "bg-destructive/10",
    titleColor: "text-destructive",
    iconColor: "text-destructive",
  },
  important: {
    icon: Star,
    borderColor: "border-brand-secondary/40",
    bgColor: "bg-brand-secondary/10",
    titleColor: "text-brand-secondary",
    iconColor: "text-brand-secondary",
  },
};

interface CalloutBlockProps {
  calloutType?: string;
  calloutTitle?: string;
  children?: React.ReactNode;
}

export function CalloutBlock({ calloutType, calloutTitle, children }: CalloutBlockProps) {
  const type = (calloutType as CalloutType) ?? "note";
  const config = CALLOUT_CONFIG[type] ?? CALLOUT_CONFIG.note;
  const Icon = config.icon;

  return (
    <div
      className={`not-prose my-4 rounded-r-md border-l-4 px-4 py-3 ${config.borderColor} ${config.bgColor}`}
    >
      <div className={`mb-1 flex items-center gap-1.5 text-sm font-semibold ${config.titleColor}`}>
        <Icon className={`h-4 w-4 shrink-0 ${config.iconColor}`} />
        <span>{calloutTitle}</span>
      </div>
      <div className="prose prose-sm prose-invert max-w-none text-sm">
        {children}
      </div>
    </div>
  );
}
