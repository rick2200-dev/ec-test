import { OrderDetailPage } from "@/components/pages/OrderDetailPage";

interface OrderDetailRouteProps {
  params: Promise<{ id: string }>;
}

export default async function OrderDetailRoute({ params }: OrderDetailRouteProps) {
  const { id } = await params;
  return <OrderDetailPage orderId={id} />;
}
