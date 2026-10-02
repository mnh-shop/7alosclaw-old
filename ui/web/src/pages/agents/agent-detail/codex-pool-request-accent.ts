import type { CodexPoolRecentRequest } from "./hooks/use-codex-pool-activity";

export interface RequestAccentClasses {
  card: string;
  glow: string;
  stripe: string;
  marker: string;
  index: string;
  directBadge: string;
  pill: string;
  trace: string;
}

export const REQUEST_ACCENTS: RequestAccentClasses[] = [
  {
    card: "border-success/20",
    glow:
      "from-success/[0.12] via-success/[0.04] to-transparent dark:from-success/[0.16] dark:via-success/[0.06]",
    stripe: "from-success/90 via-success/75 to-success/35",
    marker: "bg-success",
    index:
      "border-success/30 bg-success/10 text-success dark:border-success/30 dark:bg-success/15 dark:text-success",
    directBadge:
      "border-success/20 bg-success/10 text-success dark:border-success/25 dark:bg-success/15 dark:text-success",
    pill:
      "border-success/15 bg-success/[0.07] text-success dark:border-success/20 dark:bg-success/[0.12] dark:text-success",
    trace:
      "text-success hover:border-success/25 hover:bg-success/10 hover:text-success dark:text-success dark:hover:border-success/30 dark:hover:bg-success/15 dark:hover:text-success",
  },
  {
    card: "border-info/20",
    glow:
      "from-info/[0.12] via-info/[0.04] to-transparent dark:from-info/[0.16] dark:via-info/[0.06]",
    stripe: "from-info/90 via-info/75 to-info/35",
    marker: "bg-info",
    index:
      "border-info/30 bg-info/10 text-info dark:border-info/30 dark:bg-info/15 dark:text-info",
    directBadge:
      "border-info/20 bg-info/10 text-info dark:border-info/25 dark:bg-info/15 dark:text-info",
    pill:
      "border-info/15 bg-info/[0.07] text-info dark:border-info/20 dark:bg-info/[0.12] dark:text-info",
    trace:
      "text-info hover:border-info/25 hover:bg-info/10 hover:text-info dark:text-info dark:hover:border-info/30 dark:hover:bg-info/15 dark:hover:text-info",
  },
  {
    card: "border-warning/20",
    glow:
      "from-warning/[0.12] via-warning/[0.04] to-transparent dark:from-warning/[0.16] dark:via-warning/[0.06]",
    stripe: "from-warning/90 via-warning/75 to-warning/35",
    marker: "bg-warning",
    index:
      "border-warning/30 bg-warning/10 text-warning dark:border-warning/30 dark:bg-warning/15 dark:text-warning",
    directBadge:
      "border-warning/20 bg-warning/10 text-warning dark:border-warning/25 dark:bg-warning/15 dark:text-warning",
    pill:
      "border-warning/15 bg-warning/[0.07] text-warning dark:border-warning/20 dark:bg-warning/[0.12] dark:text-warning",
    trace:
      "text-warning hover:border-warning/25 hover:bg-warning/10 hover:text-warning dark:text-warning dark:hover:border-warning/30 dark:hover:bg-warning/15 dark:hover:text-warning",
  },
  {
    card: "border-destructive/20",
    glow:
      "from-destructive/[0.12] via-destructive/[0.04] to-transparent dark:from-destructive/[0.16] dark:via-destructive/[0.06]",
    stripe: "from-destructive/90 via-destructive/75 to-warning/35",
    marker: "bg-destructive",
    index:
      "border-destructive/30 bg-destructive/10 text-destructive dark:border-destructive/30 dark:bg-destructive/15 dark:text-destructive",
    directBadge:
      "border-destructive/20 bg-destructive/10 text-destructive dark:border-destructive/25 dark:bg-destructive/15 dark:text-destructive",
    pill:
      "border-destructive/15 bg-destructive/[0.07] text-destructive dark:border-destructive/20 dark:bg-destructive/[0.12] dark:text-destructive",
    trace:
      "text-destructive hover:border-destructive/25 hover:bg-destructive/10 hover:text-destructive dark:text-destructive dark:hover:border-destructive/30 dark:hover:bg-destructive/15 dark:hover:text-destructive",
  },
];

export function requestProviderSummary(request: CodexPoolRecentRequest): string {
  const selected = request.selected_provider?.trim();
  const served = request.provider_name?.trim();
  if (request.used_failover && selected && served && selected !== served) {
    return `${selected} -> ${served}`;
  }
  return served || selected || "";
}

export function requestAccentSeed(request: CodexPoolRecentRequest): string {
  return (
    request.provider_name?.trim() ||
    request.selected_provider?.trim() ||
    requestProviderSummary(request) ||
    request.model ||
    request.span_id
  );
}

export function requestAccentClasses(seed: string): RequestAccentClasses {
  let hash = 0;
  for (const char of seed) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  }
  return REQUEST_ACCENTS[hash % REQUEST_ACCENTS.length] ?? REQUEST_ACCENTS[0]!;
}
