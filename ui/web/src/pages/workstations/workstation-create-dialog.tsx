import { useState } from "react";
import { useTranslation } from "react-i18next";
import { ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { CreateWorkstationParams, WorkstationBackendType } from "./hooks/use-workstations";
import {
  buildWorkstationCreatePayload,
  type ContainerRuntime,
  type SshAuthMethod,
} from "./workstation-create-dialog-helpers";
import { HostTrustFields } from "./host-trust-fields";
import type { HostTrustFormState } from "./workstation-create-dialog-helpers";

interface WorkstationCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (params: CreateWorkstationParams) => Promise<void>;
}

export function WorkstationCreateDialog({
  open,
  onOpenChange,
  onCreate,
}: WorkstationCreateDialogProps) {
  const { t } = useTranslation("workstations");

  const [name, setName] = useState("");
  const [key, setKey] = useState("");
  const [backend, setBackend] = useState<WorkstationBackendType>("ssh");
  // SSH transport — shared by both backends. The container backend reaches its
  // remote host over this same SSH metadata.
  const [host, setHost] = useState("");
  const [port, setPort] = useState("22");
  const [user, setUser] = useState("");
  const [authMethod, setAuthMethod] = useState<SshAuthMethod>("privateKey");
  const [privateKey, setPrivateKey] = useState("");
  const [password, setPassword] = useState("");
  // SSH host identity. The gateway fails closed without one of these, so the
  // form states the consequence at the point of configuration.
  const [trust, setTrust] = useState<HostTrustFormState>({
    knownHostsFingerprint: "",
    hostCaKey: "",
    allowInsecureHostKey: false,
  });
  // Container backend (wire value "docker")
  const [image, setImage] = useState("");
  const [runtime, setRuntime] = useState<ContainerRuntime>("auto");
  const [allowRootfulSocket, setAllowRootfulSocket] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [fieldError, setFieldError] = useState<string | null>(null);

  function resetForm() {
    setName("");
    setKey("");
    setBackend("ssh");
    setHost("");
    setPort("22");
    setUser("");
    setAuthMethod("privateKey");
    setPrivateKey("");
    setPassword("");
    setTrust({ knownHostsFingerprint: "", hostCaKey: "", allowInsecureHostKey: false });
    setImage("");
    setRuntime("auto");
    setAllowRootfulSocket(false);
    setFieldError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    const built = buildWorkstationCreatePayload({
      key,
      name,
      backend,
      host,
      port,
      user,
      authMethod,
      privateKey,
      password,
      image,
      runtime,
      allowRootfulSocket,
      ...trust,
    });
    if (built.kind === "error") {
      setFieldError(t(`createDialog.errors.${built.errorKey}`));
      return;
    }

    setFieldError(null);
    setSubmitting(true);
    try {
      await onCreate(built.payload);
      resetForm();
      onOpenChange(false);
    } catch (err) {
      setFieldError(err instanceof Error ? err.message : t("createDialog.errors.createFailed"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!submitting) { resetForm(); onOpenChange(v); } }}>
      <DialogContent className="sm:max-w-lg">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{t("createDialog.title")}</DialogTitle>
            <DialogDescription>{t("createDialog.description")}</DialogDescription>
          </DialogHeader>

          <div className="mt-4 space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="ws-name">{t("createDialog.nameLabel")}</Label>
              <Input
                id="ws-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("createDialog.namePlaceholder")}
                required
                className="text-base md:text-sm"
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="ws-key">{t("createDialog.keyLabel")}</Label>
              <Input
                id="ws-key"
                value={key}
                onChange={(e) => setKey(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                placeholder={t("createDialog.keyPlaceholder")}
                required
                className="text-base md:text-sm"
              />
              <p className="text-xs text-muted-foreground">{t("createDialog.keyHint")}</p>
            </div>

            <div className="space-y-1.5">
              <Label>{t("createDialog.backendLabel")}</Label>
              <Select value={backend} onValueChange={(v) => setBackend(v as WorkstationBackendType)}>
                <SelectTrigger className="text-base md:text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ssh">{t("createDialog.sshOption")}</SelectItem>
                  <SelectItem value="docker">{t("createDialog.dockerOption")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Transport. For the container backend this is the SSH connection to
                the remote host that runs podman/docker — there is no local socket. */}
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div className="space-y-1.5 sm:col-span-2">
                <Label htmlFor="ws-host">{t("createDialog.hostLabel")}</Label>
                <Input
                  id="ws-host"
                  value={host}
                  onChange={(e) => setHost(e.target.value)}
                  placeholder={t("createDialog.hostPlaceholder")}
                  className="text-base md:text-sm"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="ws-port">{t("createDialog.portLabel")}</Label>
                <Input
                  id="ws-port"
                  type="number"
                  min={1}
                  max={65535}
                  value={port}
                  onChange={(e) => setPort(e.target.value)}
                  className="text-base md:text-sm"
                />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ws-user">{t("createDialog.userLabel")}</Label>
              <Input
                id="ws-user"
                value={user}
                onChange={(e) => setUser(e.target.value)}
                placeholder={t("createDialog.userPlaceholder")}
                className="text-base md:text-sm"
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t("createDialog.authMethodLabel")}</Label>
              <Select value={authMethod} onValueChange={(v) => setAuthMethod(v as SshAuthMethod)}>
                <SelectTrigger className="text-base md:text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="privateKey">{t("createDialog.authPrivateKeyOption")}</SelectItem>
                  <SelectItem value="password">{t("createDialog.authPasswordOption")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {authMethod === "privateKey" ? (
              <div className="space-y-1.5">
                <Label htmlFor="ws-private-key">{t("createDialog.privateKeyLabel")}</Label>
                <Textarea
                  id="ws-private-key"
                  value={privateKey}
                  onChange={(e) => setPrivateKey(e.target.value)}
                  placeholder={t("createDialog.privateKeyPlaceholder")}
                  rows={5}
                  spellCheck={false}
                  autoComplete="off"
                  className="font-mono text-base md:text-sm"
                />
                <p className="text-xs text-muted-foreground">{t("createDialog.privateKeyHint")}</p>
              </div>
            ) : (
              <div className="space-y-1.5">
                <Label htmlFor="ws-password">{t("createDialog.passwordLabel")}</Label>
                <Input
                  id="ws-password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t("createDialog.passwordPlaceholder")}
                  autoComplete="new-password"
                  className="text-base md:text-sm"
                />
                <p className="text-xs text-muted-foreground">{t("createDialog.passwordHint")}</p>
              </div>
            )}

            {backend === "docker" && (
              <>
                <div className="space-y-1.5">
                  <Label htmlFor="ws-image">{t("createDialog.imageLabel")}</Label>
                  <Input
                    id="ws-image"
                    value={image}
                    onChange={(e) => setImage(e.target.value)}
                    placeholder={t("createDialog.imagePlaceholder")}
                    className="text-base md:text-sm"
                  />
                  <p className="text-xs text-muted-foreground">{t("createDialog.imageHint")}</p>
                </div>

                <div className="space-y-1.5">
                  <Label>{t("createDialog.runtimeLabel")}</Label>
                  <Select value={runtime} onValueChange={(v) => setRuntime(v as ContainerRuntime)}>
                    <SelectTrigger className="text-base md:text-sm">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="auto">{t("createDialog.runtimeAutoOption")}</SelectItem>
                      <SelectItem value="podman">{t("createDialog.runtimePodmanOption")}</SelectItem>
                      <SelectItem value="docker">{t("createDialog.runtimeDockerOption")}</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">{t("createDialog.runtimeHint")}</p>
                </div>

                {runtime !== "podman" && (
                  <div className="space-y-2 rounded-md border border-destructive/30 bg-destructive/5 p-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="space-y-0.5">
                        <Label htmlFor="ws-rootful-docker" className="text-destructive">
                          {t("createDialog.allowRootfulLabel")}
                        </Label>
                        <p className="text-xs text-muted-foreground">
                          {t("createDialog.allowRootfulHint")}
                        </p>
                      </div>
                      <Switch
                        id="ws-rootful-docker"
                        checked={allowRootfulSocket}
                        onCheckedChange={setAllowRootfulSocket}
                      />
                    </div>
                    {runtime === "docker" && !allowRootfulSocket && (
                      <div className="flex items-start gap-2 text-sm text-destructive">
                        <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" />
                        <p>{t("createDialog.rootfulRequiredWarning")}</p>
                      </div>
                    )}
                  </div>
                )}
              </>
            )}

            <HostTrustFields value={trust} onChange={setTrust} />

            {fieldError && (
              <p className="text-sm text-destructive">{fieldError}</p>
            )}
          </div>

          <DialogFooter className="mt-6">
            <Button type="button" variant="outline" onClick={() => { resetForm(); onOpenChange(false); }} disabled={submitting}>
              {t("createDialog.cancel")}
            </Button>
            <Button type="submit" disabled={submitting || !name.trim() || !key.trim()}>
              {t("createDialog.create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
