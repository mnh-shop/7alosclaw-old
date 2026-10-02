import { useState, useEffect, useCallback } from "react";
import { useWs } from "@/hooks/use-ws";
import { useAuthStore } from "@/stores/use-auth-store";
import { Methods } from "@/api/protocol";

export type WorkstationBackendType = "ssh" | "docker";

/**
 * Subset of store.SanitizedWorkstation that this page consumes. Field names are
 * camelCase to match the Go json tags — the WS client sends and receives params
 * verbatim, so any divergence here silently yields undefined rather than a
 * decode error.
 */
export interface Workstation {
  id: string;
  workstationKey: string;
  name: string;
  backendType: WorkstationBackendType;
  active: boolean;
  createdAt: string;
  updatedAt: string;
  /**
   * Non-secret view of the stored metadata (store/workstation_store.go
   * SanitizedView, json "metadataSummary"): carries the host-verification
   * fields a workstation needs to connect, never credentials. The console
   * dropped it along with the trust UI; both are required, because the gateway
   * refuses the dial when no anchor is set.
   */
  metadataSummary?: {
    knownHostsFingerprint?: string;
    hostCaKey?: string;
    allowInsecureHostKey?: boolean;
    /**
     * True for a pre-SSH container row (store/legacyContainerSummary): the
     * shape written before the container backend moved onto the SSH transport.
     * Such a row has no host, user or credentials and cannot be opened or
     * tested; the console disables the test action for it with a reason.
     */
    legacy?: boolean;
    // The Go side is map[string]any and carries more than the trust fields
    // (host/port/user/hasKey for ssh, containerName for docker), so the
    // three above are named for the code that reads them, not exhaustive.
    // NOTE: SanitizedView populates the trust trio for ssh and for canonical
    // container rows; a legacy row's summary omits them, so the edit dialog
    // treats its stored anchor as unreadable and leaves it untouched.
    [key: string]: unknown;
  };
}

export interface CreateWorkstationParams {
  workstationKey: string;
  name: string;
  backendType: WorkstationBackendType;
  metadata?: Record<string, unknown>;
}

export interface UpdateWorkstationParams {
  name?: string;
  active?: boolean;
  metadata?: Record<string, unknown>;
}

/**
 * The sanitized report from workstation.TestConnection, wire-mirroring
 * store's workstation.ConnectionTestResult (internal/workstation/backend.go).
 * Runtime/Image are omitted for non-container backends; rootfulSocket and
 * message are omitempty. Never carries credentials or host paths.
 */
export interface WorkstationConnectionTest {
  backend: string;
  runtime?: string;
  image?: string;
  imagePresent: boolean;
  rootfulSocket?: boolean;
  message?: string;
}

export function useWorkstations() {
  const ws = useWs();
  const connected = useAuthStore((s) => s.connected);
  const [workstations, setWorkstations] = useState<Workstation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!connected) return;
    setLoading(true);
    setError(null);
    try {
      const res = await ws.call<{ workstations: Workstation[] }>(Methods.WORKSTATIONS_LIST);
      setWorkstations(res.workstations ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load workstations");
    } finally {
      setLoading(false);
    }
  }, [ws, connected]);

  useEffect(() => {
    load();
  }, [load]);

  const createWorkstation = useCallback(
    async (params: CreateWorkstationParams): Promise<Workstation> => {
      const res = await ws.call<{ workstation: Workstation }>(Methods.WORKSTATIONS_CREATE, params as unknown as Record<string, unknown>);
      await load();
      return res.workstation;
    },
    [ws, load],
  );

  const updateWorkstation = useCallback(
    async (id: string, params: UpdateWorkstationParams): Promise<void> => {
      // The handler reads a nested `updates` map; a flattened body decodes to an
      // empty map and is rejected with "no updates provided".
      await ws.call(Methods.WORKSTATIONS_UPDATE, { id, updates: params });
      await load();
    },
    [ws, load],
  );

  const deleteWorkstation = useCallback(
    async (id: string): Promise<void> => {
      await ws.call(Methods.WORKSTATIONS_DELETE, { id });
      await load();
    },
    [ws, load],
  );

  // Diagnostic probe: opens a transient backend on the gateway, dials the
  // remote host, and returns the sanitized result. A failure rejects with the
  // backend's own message (e.g. the image is not present), which callers show
  // verbatim because it is the actionable part.
  const testWorkstation = useCallback(
    async (id: string): Promise<WorkstationConnectionTest> => {
      return ws.call<WorkstationConnectionTest>(Methods.WORKSTATIONS_TEST, { id });
    },
    [ws],
  );

  return {
    workstations,
    loading,
    error,
    refresh: load,
    createWorkstation,
    updateWorkstation,
    deleteWorkstation,
    testWorkstation,
  };
}
