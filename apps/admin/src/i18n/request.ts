import { getRequestConfig } from "next-intl/server";
import { cookies } from "next/headers";
import jaCommon from "../../../../packages/i18n/messages/ja/common.json";
import jaAdmin from "../../../../packages/i18n/messages/ja/admin.json";
import enCommon from "../../../../packages/i18n/messages/en/common.json";
import enAdmin from "../../../../packages/i18n/messages/en/admin.json";

const locales = ["ja", "en"] as const;
const defaultLocale = "ja";

const messages = {
  ja: { ...jaCommon, ...jaAdmin },
  en: { ...enCommon, ...enAdmin },
};

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const localeCookie = cookieStore.get("locale")?.value;

  let locale: (typeof locales)[number] = defaultLocale;
  if (localeCookie && locales.includes(localeCookie as (typeof locales)[number])) {
    locale = localeCookie as (typeof locales)[number];
  }

  return {
    locale,
    messages: messages[locale],
  };
});
