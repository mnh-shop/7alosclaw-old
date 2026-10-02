import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// --- EN namespaces ---
import enCommon from "./locales/en/common.json";
import enSidebar from "./locales/en/sidebar.json";
import enTopbar from "./locales/en/topbar.json";
import enLogin from "./locales/en/login.json";
import enOverview from "./locales/en/overview.json";
import enChat from "./locales/en/chat.json";
import enAgents from "./locales/en/agents.json";
import enTeams from "./locales/en/teams.json";
import enSessions from "./locales/en/sessions.json";
import enSkills from "./locales/en/skills.json";
import enCron from "./locales/en/cron.json";
import enConfig from "./locales/en/config.json";
import enChannels from "./locales/en/channels.json";
import enProviders from "./locales/en/providers.json";
import enTraces from "./locales/en/traces.json";
import enEvents from "./locales/en/events.json";
import enUsage from "./locales/en/usage.json";
import enApprovals from "./locales/en/approvals.json";
import enNodes from "./locales/en/nodes.json";
import enLogs from "./locales/en/logs.json";
import enTools from "./locales/en/tools.json";
import enMcp from "./locales/en/mcp.json";
import enTts from "./locales/en/tts.json";
import enSetup from "./locales/en/setup.json";
import enMemory from "./locales/en/memory.json";
import enVault from "./locales/en/vault.json";
import enStorage from "./locales/en/storage.json";
import enPendingMessages from "./locales/en/pending-messages.json";
import enContacts from "./locales/en/contacts.json";
import enActivity from "./locales/en/activity.json";
import enApiKeys from "./locales/en/api-keys.json";
import enCliCredentials from "./locales/en/cli-credentials.json";
import enPackages from "./locales/en/packages.json";
import enTenants from "./locales/en/tenants.json";
import enSystemSettings from "./locales/en/system-settings.json";
import enImportExport from "./locales/en/import-export.json";
import enV3Capabilities from "./locales/en/v3-capabilities.json";
import enBackup from "./locales/en/backup.json";
import enHooks from "./locales/en/hooks.json";
import enWebhooks from "./locales/en/webhooks.json";
import enWorkstations from "./locales/en/workstations.json";
import enOptions from "./locales/en/options.json";

const ns = [
  "common", "sidebar", "topbar", "login", "overview", "chat",
  "agents", "teams", "sessions", "skills", "cron", "config",
  "channels", "providers", "traces", "events",
  "usage", "approvals", "nodes", "logs", "tools", "mcp", "tts",
  "setup", "memory", "vault", "storage", "pending-messages", "contacts", "activity", "api-keys",
  "cli-credentials", "packages", "tenants", "system-settings", "import-export",
  "v3-capabilities",
  "backup",
  "hooks",
  "webhooks",
  "workstations",
  "options",
] as const;

i18n.use(initReactI18next).init({
  resources: {
    en: {
      common: enCommon, sidebar: enSidebar, topbar: enTopbar, login: enLogin,
      overview: enOverview, chat: enChat, agents: enAgents, teams: enTeams,
      sessions: enSessions, skills: enSkills, cron: enCron, config: enConfig,
      channels: enChannels, providers: enProviders, traces: enTraces,
      events: enEvents, usage: enUsage,
      approvals: enApprovals, nodes: enNodes, logs: enLogs, tools: enTools,
      mcp: enMcp, tts: enTts, setup: enSetup, memory: enMemory, vault: enVault, storage: enStorage,
      "pending-messages": enPendingMessages,
      contacts: enContacts, activity: enActivity, "api-keys": enApiKeys,
      "cli-credentials": enCliCredentials,
      packages: enPackages,
      tenants: enTenants,
      "system-settings": enSystemSettings,
      "import-export": enImportExport,
      "v3-capabilities": enV3Capabilities,
      backup: enBackup,
      hooks: enHooks,
      webhooks: enWebhooks,
      workstations: enWorkstations,
      options: enOptions,
    },
  },
  ns: [...ns],
  defaultNS: "common",
  // English only. With a single locale there is nothing to detect and nothing to
  // switch: no stored preference, no navigator sniffing, no changeLanguage. It
  // also drops the second locale's JSON from the bundle, which was most of what
  // this chunk weighed.
  lng: "en",
  fallbackLng: "en",
  interpolation: { escapeValue: false },
  missingKeyHandler: import.meta.env.DEV
    ? (_lngs, _ns, key) => console.warn(`[i18n] missing: ${key}`)
    : undefined,
});

export default i18n;
