/**
 * MermaidBlock — lazy-loads mermaid and renders diagrams as SVG.
 * Shows skeleton while loading, error fallback on failure. The console is dark
 * only, so diagrams always render with mermaid's dark theme.
 */
import { useEffect, useState, useId } from "react";

interface MermaidBlockProps {
  code: string;
}

export function MermaidBlock({ code }: MermaidBlockProps) {
  const uid = useId().replace(/:/g, "");
  const diagramId = `mermaid-${uid}`;
  const [svg, setSvg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    setSvg(null);
    setError(null);
    setLoading(true);

    let cancelled = false;

    (async () => {
      try {
        const mermaid = (await import("mermaid")).default;
        mermaid.initialize({
          startOnLoad: false,
          theme: "dark",
          securityLevel: "strict",
        });

        const { svg: renderedSvg } = await mermaid.render(diagramId, code);
        if (!cancelled) {
          setSvg(renderedSvg);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [code, diagramId]);

  if (loading) {
    return (
      <div className="not-prose my-3 overflow-x-auto rounded-lg border border-border/60">
        <div className="animate-pulse bg-muted/50 h-32 w-full rounded-lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="not-prose my-3 rounded-lg border border-destructive bg-destructive/30 p-3">
        <p className="mb-2 text-xs font-semibold text-destructive">
          Mermaid render error
        </p>
        <pre className="overflow-x-auto text-xs text-destructive whitespace-pre-wrap">
          {error}
        </pre>
        <hr className="my-2 border-destructive" />
        <pre className="overflow-x-auto text-xs text-muted-foreground whitespace-pre">{code}</pre>
      </div>
    );
  }

  return (
    <div
      className="not-prose my-3 overflow-x-auto rounded-lg border border-border/60 bg-background p-4"
      dangerouslySetInnerHTML={{ __html: svg ?? "" }}
    />
  );
}
