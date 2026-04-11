import SellerInquiryDetailPage from "@/components/pages/SellerInquiryDetailPage";

interface PageProps {
  params: Promise<{ id: string }>;
}

export default async function Page({ params }: PageProps) {
  const { id } = await params;
  return <SellerInquiryDetailPage id={id} />;
}
