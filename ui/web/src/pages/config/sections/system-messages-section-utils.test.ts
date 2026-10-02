import { describe, expect, it } from "vitest";
import {
  SYSTEM_MESSAGE_LOCALES,
  buildSystemMessagesPatch,
  localizedSystemMessageDescription,
  localizedSystemMessageLabel,
  normalizeSystemMessagesDraft,
  systemMessageDefinitionsFromSchema,
} from "./system-messages-section-utils";

describe("system-messages-section-utils", () => {
  const definitions = [
    {
      key: "pairing.group_required",
      template: "Group {{code}}",
      description: "Sent to an unpaired group chat.",
      labels: { en: "Group pairing required" },
      descriptions: { en: "Sent to an unpaired group chat." },
      variables: ["code"],
    },
    {
      key: "pairing.approved",
      template: "Approved {{app_name}}",
      description: "Sent after approval.",
      variables: ["app_name"],
    },
  ];

  it("reads localized labels and descriptions from config schema definitions", () => {
    const schema = {
      properties: {
        system_messages: {
          definitions: [definitions[0]],
        },
      },
    };

    const parsed = systemMessageDefinitionsFromSchema(schema);
    const first = parsed[0];

    expect(first).toBeDefined();
    // A region-tagged locale still resolves to its base code.
    expect(localizedSystemMessageLabel(first!, "en-US")).toBe("Group pairing required");
    expect(localizedSystemMessageDescription(first!, "en")).toBe("Sent to an unpaired group chat.");
    // A definition with no label for the locale falls back to the key.
    expect(localizedSystemMessageLabel({ key: "pairing.account_required", template: "" }, "en")).toBe("pairing.account_required");
  });

  it("normalizes the supported locales for each known message definition", () => {
    // English is the only locale the engine can render for these messages, so it
    // is the only one an operator may author.
    expect(SYSTEM_MESSAGE_LOCALES.map((locale) => locale.code)).toEqual(["en"]);

    const draft = normalizeSystemMessagesDraft(
      {
        default_locale: "en",
        messages: {
          "pairing.group_required": {
            en: "Group {{code}}",
          },
        },
      },
      definitions,
    );

    expect(draft["pairing.group_required"]).toMatchObject({ en: "Group {{code}}" });
    // A definition with no override still gets the locale key, empty.
    expect(draft["pairing.approved"]).toMatchObject({ en: "" });
  });

  it("builds a default delivery locale into the config.patch payload", () => {
    // A real override, so the assertion covers the message payload and not just
    // the default locale: an all-empty draft legitimately produces no messages.
    const patch = buildSystemMessagesPatch(
      { "pairing.group_required": { en: "Group {{code}}" } },
      undefined,
      "en",
    );

    expect(patch).toEqual({
      default_locale: "en",
      messages: {
        "pairing.group_required": { en: "Group {{code}}" },
      },
    });
  });

  it("builds a config.patch payload and preserves cleared overrides", () => {
    const patch = buildSystemMessagesPatch({
      "pairing.group_required": { en: "Group {{code}}" },
    });

    expect(patch).toEqual({
      messages: {
        "pairing.group_required": { en: "Group {{code}}" },
      },
    });
  });

  it("keeps an existing key with an empty locale so config.patch can clear old overrides", () => {
    const patch = buildSystemMessagesPatch(
      {
        "pairing.group_required": { en: "" },
      },
      {
        messages: {
          "pairing.group_required": { en: "Old" },
        },
      },
    );

    expect(patch).toEqual({
      messages: {
        "pairing.group_required": { en: "" },
      },
    });
  });
});
