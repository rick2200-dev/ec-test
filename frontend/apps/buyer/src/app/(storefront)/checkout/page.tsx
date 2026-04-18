import { getTranslations } from "next-intl/server";
import type { Metadata } from "next";
import CheckoutPage from "@/components/pages/CheckoutPage";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("checkout");
  return { title: t("title") };
}

export default function Page() {
  return <CheckoutPage />;
}
