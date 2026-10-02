import { cn } from "@/lib/utils";

interface SidebarGroupProps {
  label: string;
  collapsed?: boolean;
  children: React.ReactNode;
}

/**
 * A labelled section of the sidebar. Deliberately NOT collapsible: every group
 * in this sidebar is always visible, and a single collapsing folder among flat
 * groups reads as a bug rather than as a feature. If a group ever needs to
 * collapse, it needs a reason that applies to all of them.
 */
export function SidebarGroup({ label, collapsed, children }: SidebarGroupProps) {
  return (
    <div className="space-y-1">
      {!collapsed && (
        <p className="brand-eyebrow flex items-center gap-2 px-3 py-1 text-primary/80">
          <span className="h-px w-4 shrink-0 bg-primary/50" aria-hidden="true" />
          {label}
        </p>
      )}
      {collapsed && <div className="mx-auto my-1 h-px w-6 bg-border" />}
      <div className={cn("space-y-0.5", collapsed && "flex flex-col items-center")}>
        {children}
      </div>
    </div>
  );
}
