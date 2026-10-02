import { AlertTriangle, ShieldAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { HostTrustFormState } from "./workstation-create-dialog-helpers";

interface HostTrustFieldsProps {
  value: HostTrustFormState;
  onChange: (next: HostTrustFormState) => void;
}

/**
 * Host-identity settings shared by the create and edit dialogs.
 *
 * The gateway refuses (fails closed) to connect when no anchor is configured —
 * see backends/ssh_dial.go buildHostKeyCallback — so the warning below is shown
 * exactly when that will happen, at the point of configuration.
 */
export function HostTrustFields({ value, onChange }: HostTrustFieldsProps) {
  const { t } = useTranslation("workstations");

  const hasAnchor = value.knownHostsFingerprint.trim() !== "" || value.hostCaKey.trim() !== "";
  const bothSet = value.knownHostsFingerprint.trim() !== "" && value.hostCaKey.trim() !== "";

  return (
    <div className="space-y-4 rounded-md border border-border bg-muted/20 p-3">
      <div className="space-y-0.5">
        <h4 className="text-sm font-medium">{t("hostTrust.sectionTitle")}</h4>
        <p className="text-xs text-muted-foreground">{t("hostTrust.sectionHint")}</p>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="ws-known-host">{t("hostTrust.fingerprintLabel")}</Label>
        <Input
          id="ws-known-host"
          value={value.knownHostsFingerprint}
          onChange={(e) => onChange({ ...value, knownHostsFingerprint: e.target.value })}
          placeholder={t("hostTrust.fingerprintPlaceholder")}
          spellCheck={false}
          autoComplete="off"
          className="font-mono text-base md:text-sm"
        />
        <p className="text-xs text-muted-foreground">{t("hostTrust.fingerprintHint")}</p>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="ws-host-ca">{t("hostTrust.caKeyLabel")}</Label>
        <Textarea
          id="ws-host-ca"
          value={value.hostCaKey}
          onChange={(e) => onChange({ ...value, hostCaKey: e.target.value })}
          placeholder={t("hostTrust.caKeyPlaceholder")}
          rows={2}
          spellCheck={false}
          autoComplete="off"
          className="font-mono text-base md:text-sm"
        />
        <p className="text-xs text-muted-foreground">{t("hostTrust.caKeyHint")}</p>
      </div>

      {bothSet && (
        <p className="text-xs text-muted-foreground">{t("hostTrust.precedenceHint")}</p>
      )}

      {!hasAnchor && !value.allowInsecureHostKey && (
        <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-300">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          <p>{t("hostTrust.noAnchorWarning")}</p>
        </div>
      )}

      <div className="space-y-2 rounded-md border border-destructive/30 bg-destructive/5 p-3">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-0.5">
            <Label htmlFor="ws-insecure-host-key" className="text-destructive">
              {t("hostTrust.insecureLabel")}
            </Label>
            <p className="text-xs text-muted-foreground">{t("hostTrust.insecureHint")}</p>
          </div>
          <Switch
            id="ws-insecure-host-key"
            checked={value.allowInsecureHostKey}
            onCheckedChange={(checked) => onChange({ ...value, allowInsecureHostKey: checked })}
          />
        </div>
        {value.allowInsecureHostKey && (
          <div className="flex items-start gap-2 text-sm text-destructive">
            <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" />
            <p>{t("hostTrust.insecureWarning")}</p>
          </div>
        )}
      </div>
    </div>
  );
}
