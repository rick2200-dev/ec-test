import type { Preview } from "@storybook/react";
import React from "react";
import { NextIntlClientProvider } from "next-intl";
import "./storybook.css";
import jaCommon from "../../../packages/i18n/messages/ja/common.json";
import jaBuyer from "../../../packages/i18n/messages/ja/buyer.json";
import jaSeller from "../../../packages/i18n/messages/ja/seller.json";
import jaAdmin from "../../../packages/i18n/messages/ja/admin.json";
import enCommon from "../../../packages/i18n/messages/en/common.json";
import enBuyer from "../../../packages/i18n/messages/en/buyer.json";
import enSeller from "../../../packages/i18n/messages/en/seller.json";
import enAdmin from "../../../packages/i18n/messages/en/admin.json";

const allMessages = {
  ja: { ...jaCommon, ...jaBuyer, ...jaSeller, ...jaAdmin },
  en: { ...enCommon, ...enBuyer, ...enSeller, ...enAdmin },
};

const preview: Preview = {
  globalTypes: {
    locale: {
      description: "Internationalization locale",
      toolbar: {
        icon: "globe",
        items: [
          { value: "ja", title: "日本語" },
          { value: "en", title: "English" },
        ],
        showName: true,
      },
    },
  },
  initialGlobals: {
    locale: "ja",
  },
  decorators: [
    (Story, context) => {
      const locale = (context.globals.locale as "ja" | "en") || "ja";
      return (
        <NextIntlClientProvider locale={locale} messages={allMessages[locale]}>
          <Story />
        </NextIntlClientProvider>
      );
    },
  ],
  parameters: {
    a11y: {},
  },
};

export default preview;
