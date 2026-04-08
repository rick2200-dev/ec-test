import { getRequestConfig } from "next-intl/server";
import { cookies } from "next/headers";
import jaCommon from "../../../../packages/i18n/messages/ja/common.json";
import jaSeller from "../../../../packages/i18n/messages/ja/seller.json";
import enCommon from "../../../../packages/i18n/messages/en/common.json";
import enSeller from "../../../../packages/i18n/messages/en/seller.json";

const locales = ["ja", "en"] as const;
const defaultLocale = "ja";

const messages = {
  ja: { ...jaCommon, ...jaSeller },
  en: { ...enCommon, ...enSeller },
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
