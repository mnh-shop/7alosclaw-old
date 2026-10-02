import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertTriangle, CheckCircle2, Loader2, PlugZap, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type {
  UpdateWorkstationParams,
  Workstation,
  WorkstationConnectionTest,
} from "./hooks/use-workstations";
import { hostTrustPatch, validateHostTrust, type HostTrustFormState } from "./workstation-create-dialog-helpers";
import { HostTrustFields } from "./host-trust-fields";

interface WorkstationEditDialogProps {
  /** The workstation being edited. Null when closed. */
  workstation: Workstation | null;
  onOpenChange: (open: boolean) => void;
  onSave: (id: string, params: UpdateWorkstationParams) => Promise<void>;
  /** Dials the remote host and reports what was verified. Rejects with the backend's own message. */
  onTest: (id: string) => Promise<WorkstationConnectionTest>;
}

/**
 * The engine's SanitizedView exposes the three non-secret host-verification
 * fields inside `metadata` (knownHostsFingerprint, hostCaKey,
 * allowInsecureHostKey — DIFFERENCES.md row 28); credentials are never returned.
 */
function trustFrom(ws: Workstation | null): HostTrustFormState {
  const summary = ws?.metadataSummary;
  return {
    knownHostsFingerprint: summary?.knownHostsFingerprint ?? "",
    hostCaKey: summary?.hostCaKey ?? "",
    allowInsecureHostKey: summary?.allowInsecureHostKey === true,
  };
}

/**
 * Whether the API actually returned the trust fields for this row. SanitizedView
 * populates them for the ssh backend and for canonical container rows; a legacy
 * container row omits them, so the console cannot read (or safely rewrite) its
 * anchor. When this is false the edit dialog must NOT send a trust patch: the
 * gateway merges the patch over stored metadata, and hostTrustPatch always
 * writes all three keys explicitly, so an empty read would silently wipe a
 * configured anchor.
 */
function trustIsExposed(ws: Workstation | null): boolean {
  const summary = ws?.metadataSummary;
  return summary != null && "knownHostsFingerprint" in summary;
}

/**
 * Success detail from ConnectionTestResult. Runtime/Image are absent for the
 * ssh backend, so they are rendered only when the probe resolved a container
 * runtime; ImagePresent tells the operator the pinned image is actually there.
 */
function TestSuccess({ result }: { result: WorkstationConnectionTest }) {
  const { t } = useTranslation("workstations");
  return (
    <div className="space-y-2 rounded-md border border-success/40 bg-success/5 px-3 py-2">
      <div className="flex items-center gap-2 text-sm font-medium text-success">
        <CheckCircle2 className="h-4 w-4 shrink-0" />
        {t("testResult.success")}
      </div>
      <dl className="space-y-1 text-xs">
        <div className="flex gap-1.5">
          <dt className="text-muted-foreground">{t("testResult.backendLabel")}</dt>
          <dd className="font-mono text-foreground">{result.backend}</dd>
        </div>
        {result.runtime && (
          <div className="flex gap-1.5">
            <dt className="text-muted-foreground">{t("testResult.runtimeLabel")}</dt>
            <dd className="font-mono text-foreground">{result.runtime}</dd>
          </div>
        )}
        {result.image && (
          <div className="flex gap-1.5">
            <dt className="text-muted-foreground">{t("testResult.imageLabel")}</dt>
            <dd className="break-all font-mono text-foreground">
              {result.image}
              <span className="ml-1 font-sans text-muted-foreground">
                — {result.imagePresent ? t("testResult.imagePresent") : t("testResult.imageAbsent")}
              </span>
            </dd>
          </div>
        )}
        {result.message && <p className="text-muted-foreground">{result.message}</p>}
      </dl>
      {result.rootfulSocket && (
        <div className="flex items-start gap-2 rounded border border-warning/40 bg-warning/10 px-2 py-1.5 text-xs text-warning">
          <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <p>{t("testResult.rootfulWarning")}</p>
        </div>
      )}
    </div>
  );
}

/**
 * Edits the settings that were previously hand-edit-only: the host trust anchor
 * and insecure-mode opt-in, plus the display name, and runs the gateway's
 * connection probe. Credentials are deliberately not shown or edited here — the
 * gateway merges this patch over the stored metadata, so the private key /
 * password are untouched.
 */
export function WorkstationEditDialog({ workstation, onOpenChange, onSave, onTest }: WorkstationEditDialogProps) {
  const { t } = useTranslation("workstations");

  const [name, setName] = useState("");
  const [trust, setTrust] = useState<HostTrustFormState>(trustFrom(null));
  const [submitting, setSubmitting] = useState(false);
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<WorkstationConnectionTest | null>(null);
  const [testError, setTestError] = useState<string | null>(null);

  // Re-seed the form each time a workstation is opened for editing. A result
  // from the previously opened row must never linger on screen.
  useEffect(() => {
    if (!workstation) return;
    setName(workstation.name);
    setTrust(trustFrom(workstation));
    setFieldError(null);
    setTestResult(null);
    setTestError(null);
  }, [workstation]);

  /**
   * A legacy row predates the SSH transport: it has no host, user or
   * credentials, so the gateway cannot open it. Offering the probe would only
   * produce a confusing failure, so it is disabled with the reason shown.
   */
  const legacy = workstation?.metadataSummary?.legacy === true;

  async function handleTest() {
    if (!workstation || legacy) return;
    setTesting(true);
    // Clear the previous attempt first: a stale result must never sit next to
    // an in-flight probe.
    setTestResult(null);
    setTestError(null);
    try {
      setTestResult(await onTest(workstation.id));
    } catch (err) {
      setTestError(err instanceof Error ? err.message : t("testResult.failed"));
    } finally {
      setTesting(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!workstation) return;

    if (!name.trim()) {
      setFieldError(t("editDialog.errors.nameRequired"));
      return;
    }

    // Only edit trust when the API gave us the current values. Sending the
    // always-explicit trust patch blind would wipe an anchor we cannot see.
    const trustEditable = trustIsExposed(workstation);
    if (trustEditable) {
      const trustError = validateHostTrust(trust);
      if (trustError) {
        setFieldError(t(`createDialog.errors.${trustError}`));
        return;
      }
    }

    setFieldError(null);
    setSubmitting(true);
    try {
      await onSave(workstation.id, {
        name: name.trim(),
        ...(trustEditable ? { metadata: hostTrustPatch(trust) } : {}),
      });
      onOpenChange(false);
    } catch (err) {
      setFieldError(err instanceof Error ? err.message : t("editDialog.errors.saveFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  const trustEditable = trustIsExposed(workstation);
  const busy = submitting || testing;

  return (
    <Dialog open={workstation !== null} onOpenChange={(v) => { if (!busy) onOpenChange(v); }}>
      <DialogContent className="sm:max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t("editDialog.title")}</DialogTitle>
            <DialogDescription>{t("editDialog.description")}</DialogDescription>
          </DialogHeader>

          <div className="mt-4 space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="ws-edit-name">{t("createDialog.nameLabel")}</Label>
              <Input
                id="ws-edit-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("createDialog.namePlaceholder")}
                required
                className="text-base md:text-sm"
              />
            </div>

            {trustEditable ? (
              <HostTrustFields value={trust} onChange={setTrust} />
            ) : (
              <p className="text-xs text-muted-foreground">{t("editDialog.trustUnavailable")}</p>
            )}

            <div className="space-y-3 rounded-md border border-border bg-muted/20 p-3">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-0.5">
                  <h4 className="text-sm font-medium">{t("testResult.title")}</h4>
                  <p className="text-xs text-muted-foreground">{t("testResult.hint")}</p>
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={handleTest}
                  disabled={testing || legacy || submitting}
                  title={legacy ? t("testResult.legacyReason") : undefined}
                  className="shrink-0 gap-1.5 max-sm:h-11"
                >
                  {testing ? (
                    <>
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      {t("testResult.running")}
                    </>
                  ) : (
                    <>
                      <PlugZap className="h-3.5 w-3.5" />
                      {t("testResult.action")}
                    </>
                  )}
                </Button>
              </div>

              {legacy && (
                <div className="flex items-start gap-2 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-warning">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                  <div className="space-y-0.5">
                    <p className="text-sm font-medium">{t("testResult.legacyTitle")}</p>
                    <p className="text-xs">{t("testResult.legacyReason")}</p>
                  </div>
                </div>
              )}

              {testError && (
                <div className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                  <XCircle className="mt-0.5 h-4 w-4 shrink-0" />
                  <p className="break-words">{testError}</p>
                </div>
              )}

              {testResult && <TestSuccess result={testResult} />}
            </div>

            {fieldError && <p className="text-sm text-destructive">{fieldError}</p>}
          </div>

          <DialogFooter className="mt-6">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
              {t("createDialog.cancel")}
            </Button>
            <Button type="submit" disabled={busy || !name.trim()}>
              {t("editDialog.save")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
