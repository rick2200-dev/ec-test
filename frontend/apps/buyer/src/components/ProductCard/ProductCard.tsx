import { Product } from "@/lib/types";
import { formatPrice, getLowestPrice, getSellerById } from "@/lib/mock-data";
import { ProductCardPresenter } from "./ProductCard.presenter";

interface ProductCardProps {
  product: Product;
}

export default function ProductCard({ product }: ProductCardProps) {
  const price = getLowestPrice(product.id);
  const seller = getSellerById(product.seller_id);

  return (
    <ProductCardPresenter
      href={`/products/${product.slug}`}
      name={product.name}
      sellerName={seller?.name}
      priceLabel={formatPrice(price)}
    />
  );
}
