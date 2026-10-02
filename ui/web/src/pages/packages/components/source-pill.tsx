import { cn } from "@/lib/utils";

interface Props {
  source: "github" | "pip" | "npm" | "apk" | string;
}

const SOURCE_CLASSES: Record<string, string> = {
  github:
    "bg-slate-100 text-slate-900 dark:bg-slate-800 dark:text-slate-100",
  pip: "bg-blue-100 text-blue-900 dark:bg-blue-900/40 dark:text-blue-200",
  npm: "bg-warning text-warning dark:bg-warning/40 dark:text-warning",
  apk: "bg-success text-success dark:bg-success/40 dark:text-success",
};

const NEUTRAL =
  "bg-muted text-muted-foreground";

/**
 * Small colored pill indicating a package source (github / pip / npm / apk / other).
 */
export function SourcePill({ source }: Props) {
  const classes = SOURCE_CLASSES[source] ?? NEUTRAL;
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium",
        classes,
      )}
    >
      {source}
    </span>
  );
}
