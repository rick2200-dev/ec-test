import { getRequestConfig } from "next-intl/server";
import { cookies } from "next/headers";
import jaCommon from "../../../../packages/i18n/messages/ja/common.json";
import jaBuyer from "../../../../packages/i18n/messages/ja/buyer.json";
import enCommon from "../../../../packages/i18n/messages/en/common.json";
import enBuyer from "../../../../packages/i18n/messages/en/buyer.json";

const locales = ["ja", "en"] as const;
const defaultLocale = "ja";

const messages = {
  ja: { ...jaCommon, ...jaBuyer },
  en: { ...enCommon, ...enBuyer },
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
