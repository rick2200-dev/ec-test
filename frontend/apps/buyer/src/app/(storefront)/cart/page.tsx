import { getTranslations } from "next-intl/server";
import type { Metadata } from "next";
import CartPage from "@/components/pages/CartPage";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("cart");
  return { title: t("title") };
}

export default function Page() {
  return <CartPage />;
}
