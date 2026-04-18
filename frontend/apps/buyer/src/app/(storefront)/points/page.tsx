import { getTranslations } from "next-intl/server";
import type { Metadata } from "next";
import PointsPage from "@/components/pages/PointsPage";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("points");
  return { title: t("title") };
}

export default function Page() {
  return <PointsPage />;
}
