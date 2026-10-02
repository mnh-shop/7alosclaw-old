import { describe, expect, it } from "vitest";
import {
  buildWorkstationCreatePayload,
  type WorkstationCreateFormState,
} from "../workstation-create-dialog-helpers";
import type { Workstation } from "../hooks/use-workstations";
import { Methods } from "@/api/protocol";

function form(overrides: Partial<WorkstationCreateFormState> = {}): WorkstationCreateFormState {
  return {
    key: "dev-server",
    name: "Dev Server",
    backend: "ssh",
    host: "192.168.1.100",
    port: "22",
    user: "ubuntu",
    authMethod: "privateKey",
    privateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----",
    password: "",
    image: "",
    runtime: "auto",
    allowRootfulSocket: false,
    knownHostsFingerprint: "",
    hostCaKey: "",
    allowInsecureHostKey: false,
    ...overrides,
  };
}

describe("workstation create payload contract", () => {
  it("submits camelCase params the gateway handler actually decodes", () => {
    const result = buildWorkstationCreatePayload(form());

    expect(result.kind).toBe("ok");
    if (result.kind !== "ok") return;

    // Regression guard for #1258: snake_case here decodes to the zero value in
    // Go and the handler rejects the request with "workstationKey is required".
    expect(Object.keys(result.payload).sort()).toEqual([
      "backendType",
      "metadata",
      "name",
      "workstationKey",
    ]);
    expect(result.payload.workstationKey).toBe("dev-server");
    expect(result.payload.backendType).toBe("ssh");
    expect(result.payload).not.toHaveProperty("workstation_key");
    expect(result.payload).not.toHaveProperty("backend_type");
  });

  it("sends SSH credentials as privateKey, not the unsupported identity_file", () => {
    const result = buildWorkstationCreatePayload(form());
    if (result.kind !== "ok") throw new Error("expected ok");

    // store.SSHMetadata accepts privateKey or password only; identity_file was
    // silently dropped and left the request failing metadata validation.
    expect(result.payload.metadata).toEqual({
      host: "192.168.1.100",
      port: 22,
      user: "ubuntu",
      privateKey: expect.stringContaining("BEGIN OPENSSH PRIVATE KEY"),
    });
    expect(result.payload.metadata).not.toHaveProperty("identity_file");
  });

  it("sends a password when password auth is selected", () => {
    const result = buildWorkstationCreatePayload(
      form({ authMethod: "password", privateKey: "", password: "hunter2" }),
    );
    if (result.kind !== "ok") throw new Error("expected ok");

    expect(result.payload.metadata).toMatchObject({ password: "hunter2" });
    expect(result.payload.metadata).not.toHaveProperty("privateKey");
  });

  it("rejects SSH auth with no credential rather than letting the backend 400", () => {
    expect(
      buildWorkstationCreatePayload(form({ privateKey: "" })),
    ).toEqual({ kind: "error", errorKey: "sshPrivateKeyRequired" });

    expect(
      buildWorkstationCreatePayload(form({ authMethod: "password", privateKey: "", password: "" })),
    ).toEqual({ kind: "error", errorKey: "sshPasswordRequired" });
  });

  it("sends the container backend as a full SSH transport plus an image", () => {
    const ok = buildWorkstationCreatePayload(
      form({ backend: "docker", host: "10.0.0.9", user: "deploy", image: "ubuntu:24.04" }),
    );
    if (ok.kind !== "ok") throw new Error("expected ok");

    // store.ContainerMetadata embeds store.SSHMetadata: the container backend
    // runs podman/docker on a REMOTE host over SSH, so it needs host, port, user
    // and a credential, plus the image. The old payload sent {host, image} with
    // no transport and the tightened validator rejects it.
    expect(ok.payload.metadata).toEqual({
      host: "10.0.0.9",
      port: 22,
      user: "deploy",
      privateKey: expect.stringContaining("BEGIN OPENSSH PRIVATE KEY"),
      image: "ubuntu:24.04",
    });
    // socketPath is dead for creation: there is no local daemon.
    expect(ok.payload.metadata).not.toHaveProperty("socketPath");

    // transport rules are shared with the ssh backend
    expect(
      buildWorkstationCreatePayload(form({ backend: "docker", image: "ubuntu:24.04", user: "" })),
    ).toEqual({ kind: "error", errorKey: "sshHostUserRequired" });

    expect(
      buildWorkstationCreatePayload(form({ backend: "docker", image: "" })),
    ).toEqual({ kind: "error", errorKey: "dockerImageRequired" });
  });

  it("omits runtime for auto and sends podman/docker when pinned", () => {
    const auto = buildWorkstationCreatePayload(form({ backend: "docker", image: "ubuntu:24.04" }));
    if (auto.kind !== "ok") throw new Error("expected ok");
    // "" resolves on the remote host and always prefers podman, so auto sends
    // no runtime member at all.
    expect(auto.payload.metadata).not.toHaveProperty("runtime");

    const podman = buildWorkstationCreatePayload(
      form({ backend: "docker", image: "ubuntu:24.04", runtime: "podman" }),
    );
    if (podman.kind !== "ok") throw new Error("expected ok");
    expect(podman.payload.metadata).toMatchObject({ runtime: "podman" });
    expect(podman.payload.metadata).not.toHaveProperty("allowRootfulSocket");
  });

  it("refuses docker unless the rootful socket opt-in is explicit", () => {
    expect(
      buildWorkstationCreatePayload(
        form({ backend: "docker", image: "ubuntu:24.04", runtime: "docker" }),
      ),
    ).toEqual({ kind: "error", errorKey: "dockerRootfulRequired" });

    const accepted = buildWorkstationCreatePayload(
      form({
        backend: "docker",
        image: "ubuntu:24.04",
        runtime: "docker",
        allowRootfulSocket: true,
      }),
    );
    if (accepted.kind !== "ok") throw new Error("expected ok");
    expect(accepted.payload.metadata).toMatchObject({
      runtime: "docker",
      allowRootfulSocket: true,
    });
  });

  it("carries the trust anchor for the container backend too", () => {
    const fp = "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";
    const ok = buildWorkstationCreatePayload(
      form({ backend: "docker", image: "ubuntu:24.04", knownHostsFingerprint: fp }),
    );
    if (ok.kind !== "ok") throw new Error("expected ok");
    expect(ok.payload.metadata).toMatchObject({ knownHostsFingerprint: fp });
  });

  it("requires a workstation key before hitting the network", () => {
    expect(buildWorkstationCreatePayload(form({ key: "  " }))).toEqual({
      kind: "error",
      errorKey: "keyRequired",
    });
  });
});

describe("workstation RPC contract", () => {
  // The constants with no call site are the dangerous ones: a drifted value
  // cannot surface as a failing page, only as an unknown-method error the first
  // time somebody wires the feature up. WORKSTATIONS_TEST drifted exactly that
  // way ("workstations.test" against the gateway's "workstations.testConnection")
  // and, having zero call sites, nothing caught it. Values must match
  // pkg/protocol/methods.go verbatim.
  it.each([
    ["WORKSTATIONS_TEST", Methods.WORKSTATIONS_TEST, "workstations.testConnection"],
    ["WORKSTATIONS_LINK_AGENT", Methods.WORKSTATIONS_LINK_AGENT, "workstations.linkAgent"],
    ["WORKSTATIONS_UNLINK_AGENT", Methods.WORKSTATIONS_UNLINK_AGENT, "workstations.unlinkAgent"],
  ])("pins %s to the method the gateway registers", (_name, actual, want) => {
    expect(actual).toBe(want);
  });
});

describe("workstation list contract", () => {
  // Verbatim body of store.Workstation.SanitizedView() as serialized by the
  // gateway, kept complete even though the UI type is a narrower subset — that
  // is what proves the subset is actually assignable from a real response.
  // If the TS interface drifts back to snake_case this stops compiling.
  const apiResponse = {
    id: "8f2b0f7e-1c4a-4c9e-9f1a-2b3c4d5e6f70",
    workstationKey: "aether",
    tenantId: "11111111-2222-3333-4444-555555555555",
    name: "Aether",
    backendType: "ssh",
    defaultCwd: "/home/ubuntu",
    active: true,
    createdAt: "2026-06-22T15:06:38Z",
    updatedAt: "2026-06-22T15:06:38Z",
    createdBy: "admin",
    metadataSummary: { host: "10.0.0.5", port: 22, user: "ubuntu", hasKey: true },
  } as const;

  it("resolves every field the workstations table renders", () => {
    const ws: Workstation = apiResponse;

    // Regression guard for #1258: these read undefined under the old snake_case
    // interface, rendering a blank key, "backend.undefined" and "Invalid Date".
    expect(ws.workstationKey).toBe("aether");
    expect(ws.backendType).toBe("ssh");
    expect(Number.isNaN(new Date(ws.createdAt).getTime())).toBe(false);
    expect(ws.name).toBe("Aether");
    expect(ws.active).toBe(true);
  });

  it("carries the host trust anchor, because the gateway fails closed without it", () => {
    const fp = "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";
    const ok = buildWorkstationCreatePayload(form({ knownHostsFingerprint: fp }));
    if (ok.kind !== "ok") throw new Error("expected ok");
    expect(ok.payload.metadata).toMatchObject({ knownHostsFingerprint: fp });

    // empty and off values are omitted, so the stored metadata stays clean
    const clean = buildWorkstationCreatePayload(form());
    if (clean.kind !== "ok") throw new Error("expected ok");
    expect(clean.payload.metadata).not.toHaveProperty("knownHostsFingerprint");
    expect(clean.payload.metadata).not.toHaveProperty("allowInsecureHostKey");

    // a typo must not reach the dial
    expect(buildWorkstationCreatePayload(form({ knownHostsFingerprint: "not-a-fingerprint" })))
      .toEqual({ kind: "error", errorKey: "hostFingerprintShape" });
  });
});
