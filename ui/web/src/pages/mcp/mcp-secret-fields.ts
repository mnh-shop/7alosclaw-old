import type { MCPServerInput } from "@/types/mcp";

/**
 * Write-only secret handling for the MCP server form.
 *
 * `headers` and `env` are redacted from GET/list responses and are never
 * pre-filled. On update they are omitted from the payload unless the user
 * changed the field in this session, so a redacted (empty) response can never
 * overwrite a stored secret. Create always sends whatever the user entered.
 */
export interface SecretFieldChange {
  /** True when editing an existing server; create always sends entered values. */
  isEditing: boolean;
  /** react-hook-form dirty flag: user edited this field's value. */
  dirty: boolean;
  /** User explicitly opted to clear the stored value. */
  cleared: boolean;
}

export function includeSecretField({ isEditing, dirty, cleared }: SecretFieldChange): boolean {
  return !isEditing || dirty || cleared;
}

/** Build the headers/env part of an MCP server create/update payload. */
export function buildSecretFields(params: {
  isEditing: boolean;
  isStdio: boolean;
  headers: Record<string, string>;
  env: Record<string, string>;
  headersDirty: boolean;
  envDirty: boolean;
  headersCleared: boolean;
  envCleared: boolean;
}): Pick<MCPServerInput, "headers" | "env"> {
  const out: Pick<MCPServerInput, "headers" | "env"> = {};
  if (
    !params.isStdio &&
    includeSecretField({
      isEditing: params.isEditing,
      dirty: params.headersDirty,
      cleared: params.headersCleared,
    })
  ) {
    out.headers = params.headers;
  }
  if (
    includeSecretField({
      isEditing: params.isEditing,
      dirty: params.envDirty,
      cleared: params.envCleared,
    })
  ) {
    out.env = params.env;
  }
  return out;
}
