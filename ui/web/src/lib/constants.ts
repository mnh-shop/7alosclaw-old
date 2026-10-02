// Barrel re-exports for backward compatibility.
// Import directly from sub-modules for new code.
export { ROUTES } from "./routes";
export {
  TIMEZONE_OPTIONS,
  getAllIanaTimezones,
  isValidIanaTimezone,
} from "./timezone-utils";

export const LOCAL_STORAGE_KEYS = {
  TOKEN: "zalosclaw:token",
  USER_ID: "zalosclaw:userId",
  SENDER_ID: "zalosclaw:senderID",
  TENANT_ID: "zalosclaw:tenant_id",
  TENANT_HINT: "zalosclaw:tenant_hint",
  SETUP_SKIPPED: "zalosclaw:setup_skipped",
  THEME: "zalosclaw:theme",
  SIDEBAR_COLLAPSED: "zalosclaw:sidebarCollapsed",
  LANGUAGE: "zalosclaw:language",
  TIMEZONE: "zalosclaw:timezone",
} as const;

/** English only. With one locale there is nothing to select or detect — keep
 *  this list honest rather than leaving a second entry with no translations
 *  behind it. */
export const SUPPORTED_LANGUAGES = ["en"] as const;
export type Language = (typeof SUPPORTED_LANGUAGES)[number];

export const LANGUAGE_LABELS: Record<Language, string> = {
  en: "English",
};
