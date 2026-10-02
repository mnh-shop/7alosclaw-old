import type { CreateWorkstationParams, WorkstationBackendType } from "./hooks/use-workstations";

/** Auth method for the SSH backend. The gateway accepts inline PEM or a password. */
export type SshAuthMethod = "privateKey" | "password";

/**
 * Host-identity / trust settings. These map 1:1 onto store.SSHMetadata's
 * KnownHostsFingerprint, HostCAKey and AllowInsecureHostKey (json tags
 * knownHostsFingerprint / hostCaKey / allowInsecureHostKey).
 *
 * The dial fails CLOSED: with none of these set the gateway refuses to connect
 * (backends/ssh_dial.go buildHostKeyCallback). The dialog therefore has to state
 * that consequence at the point of configuration rather than let it surface as a
 * connection error later.
 *
 * Both backends need this anchor: the container backend (wire value "docker") is
 * podman-or-docker on a REMOTE host, reached over the SAME SSH transport, so
 * store.ContainerMetadata embeds store.SSHMetadata and the trust controls are
 * rendered for the ssh and container backends alike.
 */
export interface HostTrustFormState {
  /** SHA256 pin of one host key, e.g. "SHA256:abc…". */
  knownHostsFingerprint: string;
  /** authorized_keys line for a CA that signs host certificates. */
  hostCaKey: string;
  /** Dev-only escape hatch. Accepts any host key; logs a warning per dial. */
  allowInsecureHostKey: boolean;
}

/**
 * Container runtime choice. "auto" is the form's name for store's empty runtime
 * (""), which resolves on the remote host and always prefers podman; the Select
 * cannot carry an empty string value, hence the sentinel. The sentinel is never
 * sent on the wire — "auto" omits the runtime member so store's default applies.
 */
export type ContainerRuntime = "auto" | "podman" | "docker";

export interface WorkstationCreateFormState extends HostTrustFormState {
  key: string;
  name: string;
  backend: WorkstationBackendType;
  // SSH transport (both backends: the container backend reaches its remote host
  // over the same SSH metadata).
  host: string;
  port: string;
  user: string;
  authMethod: SshAuthMethod;
  privateKey: string;
  password: string;
  // Container backend (wire value "docker")
  image: string;
  runtime: ContainerRuntime;
  allowRootfulSocket: boolean;
}

export type BuildCreatePayloadResult =
  | { kind: "ok"; payload: CreateWorkstationParams }
  | { kind: "error"; errorKey: string };

// ssh.FingerprintSHA256 emits "SHA256:" + unpadded base64 of a 32-byte digest (43 chars).
const FINGERPRINT_RE = /^SHA256:[A-Za-z0-9+/]{43}=?$/;

// One authorized_keys line: key type, base64 blob, optional comment. Covers the
// host-certificate CA types OpenSSH ships today.
const CA_KEY_RE =
  /^(ssh-ed25519|ssh-rsa|ecdsa-sha2-nistp256|ecdsa-sha2-nistp384|ecdsa-sha2-nistp521|sk-ssh-ed25519@openssh\.com|ssh-dss)\s+[A-Za-z0-9+/]+={0,3}(\s+.+)?$/;

/**
 * Validates the trust fields and returns an i18n error key, or null when valid.
 * An empty fingerprint and empty CA key are allowed (the UI warns instead of
 * blocking — allowInsecureHostKey is a legitimate, if discouraged, answer), but
 * anything non-empty must be the right shape so a typo does not reach the dial.
 */
export function validateHostTrust(t: HostTrustFormState): string | null {
  const fp = t.knownHostsFingerprint.trim();
  const ca = t.hostCaKey.trim();

  if (fp && !FINGERPRINT_RE.test(fp)) return "hostFingerprintShape";

  if (ca) {
    if (/PRIVATE KEY/.test(ca)) return "hostCaKeyIsPrivateKey";
    if (ca.includes("\n")) return "hostCaKeyNotSingleLine";
    if (!CA_KEY_RE.test(ca)) return "hostCaKeyShape";
  }
  return null;
}

/**
 * Trust members for a CREATE payload. Empty/off values are omitted so the stored
 * metadata stays clean (Go's omitempty fields decode the same either way).
 */
export function hostTrustMetadata(t: HostTrustFormState): Record<string, unknown> {
  const fp = t.knownHostsFingerprint.trim();
  const ca = t.hostCaKey.trim();
  return {
    ...(fp ? { knownHostsFingerprint: fp } : {}),
    ...(ca ? { hostCaKey: ca } : {}),
    ...(t.allowInsecureHostKey ? { allowInsecureHostKey: true } : {}),
  };
}

/**
 * Trust members for an UPDATE patch. Values are always explicit (including
 * empty string / false) so clearing a pin or turning insecure mode back off
 * actually takes effect: the gateway merges this object over the stored
 * metadata, where a present key replaces and an absent key is preserved.
 */
export function hostTrustPatch(t: HostTrustFormState): Record<string, unknown> {
  return {
    knownHostsFingerprint: t.knownHostsFingerprint.trim(),
    hostCaKey: t.hostCaKey.trim(),
    allowInsecureHostKey: t.allowInsecureHostKey,
  };
}

/**
 * Validates and builds the shared SSH transport members (store.SSHMetadata).
 * Both backends carry them: the container backend embeds SSHMetadata, so the
 * same host/user/auth rules and the same fail-closed host-trust check apply.
 * Empty/off values stay out of the create payload; the trust trio is folded in
 * by the caller via hostTrustMetadata.
 */
function buildSshTransport(
  form: WorkstationCreateFormState,
): { kind: "ok"; transport: Record<string, unknown> } | { kind: "error"; errorKey: string } {
  const host = form.host.trim();
  const user = form.user.trim();
  if (!host || !user) return { kind: "error", errorKey: "sshHostUserRequired" };

  const port = parseInt(form.port, 10) || 22;
  if (port < 1 || port > 65535) return { kind: "error", errorKey: "sshPortRange" };

  const privateKey = form.privateKey.trim();
  const password = form.password;
  if (form.authMethod === "privateKey" && !privateKey) {
    return { kind: "error", errorKey: "sshPrivateKeyRequired" };
  }
  if (form.authMethod === "password" && !password) {
    return { kind: "error", errorKey: "sshPasswordRequired" };
  }

  // Host identity travels in the metadata too: the gateway fails closed without
  // it, so a workstation created here cannot connect until it is set.
  const trustError = validateHostTrust(form);
  if (trustError) return { kind: "error", errorKey: trustError };

  return {
    kind: "ok",
    transport: {
      host,
      port,
      user,
      ...(form.authMethod === "privateKey" ? { privateKey } : { password }),
    },
  };
}

/**
 * Builds the workstations.create RPC payload from dialog form state.
 *
 * Field names here must match the gateway wire contract exactly — the WS client
 * serializes params verbatim with no case conversion, and Go's json decoder
 * silently drops unknown members rather than erroring. A mismatched name
 * therefore surfaces as "<field> is required" rather than as a decode failure.
 *
 * Metadata shape mirrors store.SSHMetadata / store.ContainerMetadata. The
 * container backend (wire value "docker") is NOT a local daemon: it is
 * podman-or-docker on a remote host reached over SSH, so it carries the full
 * SSH transport (host, port, user, privateKey or password) plus an image. The
 * legacy DockerMetadata shape with a local socketPath is dead for creation —
 * the validator now rejects it.
 */
export function buildWorkstationCreatePayload(
  form: WorkstationCreateFormState,
): BuildCreatePayloadResult {
  const key = form.key.trim();
  const name = form.name.trim();

  if (!key) return { kind: "error", errorKey: "keyRequired" };
  if (!name) return { kind: "error", errorKey: "nameRequired" };

  const transport = buildSshTransport(form);
  if (transport.kind === "error") return transport;

  let metadata: Record<string, unknown>;

  if (form.backend === "ssh") {
    metadata = { ...transport.transport, ...hostTrustMetadata(form) };
  } else {
    const image = form.image.trim();
    if (!image) return { kind: "error", errorKey: "dockerImageRequired" };
    // docker drives a rootful daemon socket, which delegates host root even to
    // a non-root uid. store refuses it without the explicit opt-in; refuse in
    // the form first so the trade-off is a visible choice, not a server error.
    if (form.runtime === "docker" && !form.allowRootfulSocket) {
      return { kind: "error", errorKey: "dockerRootfulRequired" };
    }

    metadata = {
      ...transport.transport,
      image,
      // "auto" omits runtime so store's resolver (podman preferred) applies.
      ...(form.runtime !== "auto" ? { runtime: form.runtime } : {}),
      ...(form.allowRootfulSocket ? { allowRootfulSocket: true } : {}),
      ...hostTrustMetadata(form),
    };
  }

  return {
    kind: "ok",
    payload: {
      workstationKey: key,
      name,
      backendType: form.backend,
      metadata,
    },
  };
}
