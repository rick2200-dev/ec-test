import Link from "next/link";
import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("checkout");
  return { title: t("complete.title") };
}

interface CompletePageProps {
  searchParams: Promise<{ orderId?: string }>;
}

export default async function CheckoutCompletePage({ searchParams }: CompletePageProps) {
  const t = await getTranslations("checkout");
  const { orderId } = await searchParams;

  return (
    <div className="mx-auto max-w-3xl px-4 py-16 text-center">
      <div className="mx-auto h-16 w-16 rounded-full bg-green-100 flex items-center justify-center">
        <svg
          aria-hidden="true"
          className="h-8 w-8 text-green-600"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </svg>
      </div>
      <h1 className="mt-6 text-2xl font-bold text-gray-900">{t("complete.title")}</h1>
      <p className="mt-2 text-sm text-gray-500">{t("complete.description")}</p>
      {orderId && (
        <p className="mt-4 inline-block rounded-md bg-gray-100 px-3 py-1 text-sm font-mono text-gray-700">
          {t("complete.orderIdLabel")}: {orderId}
        </p>
      )}
      <div className="mt-8 flex items-center justify-center gap-3">
        {orderId && (
          <Link
            href={`/orders/${orderId}`}
            className="rounded-md bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-blue-700"
          >
            {t("complete.viewOrderCta")}
          </Link>
        )}
        <Link
          href="/"
          className="rounded-md border border-gray-300 bg-white px-5 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          {t("complete.continueCta")}
        </Link>
      </div>
    </div>
  );
}
