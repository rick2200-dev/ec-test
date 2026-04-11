import OrderDetailPage from "@/components/pages/OrderDetailPage";

interface PageProps {
  params: Promise<{ id: string }>;
}

export default async function Page({ params }: PageProps) {
  const { id } = await params;
  return <OrderDetailPage orderId={id} />;
}
