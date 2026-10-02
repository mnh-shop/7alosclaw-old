/** The messengers this console offers. The engine supports more (Bitrix24,
 *  Feishu/Lark, Pancake, Zalo) but we ship these five. */
export const CHANNEL_TYPES = [
  { value: "discord", label: "Discord" },
  { value: "facebook", label: "Facebook" },
  { value: "slack", label: "Slack" },
  { value: "telegram", label: "Telegram" },
  { value: "whatsapp", label: "WhatsApp" },
] as const;
