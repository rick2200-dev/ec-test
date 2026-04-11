import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { ProductFormPresenter, type ProductFormPresenterProps } from "./ProductForm.presenter";

const meta: Meta<typeof ProductFormPresenter> = {
  title: "Seller/ProductForm",
  component: ProductFormPresenter,
  parameters: { layout: "padded" },
};

export default meta;
type Story = StoryObj<typeof ProductFormPresenter>;

const sampleCategories = [
  {
    id: "cat-1",
    tenant_id: "t1",
    name: "アパレル",
    slug: "apparel",
    sort_order: 1,
    created_at: "2025-01-01",
  },
  {
    id: "cat-2",
    tenant_id: "t1",
    name: "雑貨",
    slug: "goods",
    sort_order: 2,
    created_at: "2025-01-01",
  },
];

export const Empty: Story = {
  args: {
    name: "",
    slug: "",
    description: "",
    categoryId: "",
    status: "draft",
    categories: sampleCategories,
    onNameChange: () => {},
    onSlugChange: () => {},
    onDescriptionChange: () => {},
    onCategoryChange: () => {},
    onStatusChange: () => {},
  },
};

export const Filled: Story = {
  args: {
    name: "オーガニックコットンTシャツ",
    slug: "organic-cotton-tshirt",
    description: "肌に優しいオーガニックコットン100%のTシャツです。",
    categoryId: "cat-1",
    status: "active",
    categories: sampleCategories,
    onNameChange: () => {},
    onSlugChange: () => {},
    onDescriptionChange: () => {},
    onCategoryChange: () => {},
    onStatusChange: () => {},
  },
};

function InteractiveStory(args: ProductFormPresenterProps) {
  const [name, setName] = useState(args.name);
  const [slug, setSlug] = useState(args.slug);
  const [description, setDescription] = useState(args.description);
  const [categoryId, setCategoryId] = useState(args.categoryId);
  const [status, setStatus] = useState<"draft" | "active">(args.status);
  return (
    <ProductFormPresenter
      {...args}
      name={name}
      slug={slug}
      description={description}
      categoryId={categoryId}
      status={status}
      onNameChange={setName}
      onSlugChange={setSlug}
      onDescriptionChange={setDescription}
      onCategoryChange={setCategoryId}
      onStatusChange={setStatus}
    />
  );
}

export const Interactive: Story = {
  render: (args) => <InteractiveStory {...args} />,
  args: {
    name: "",
    slug: "",
    description: "",
    categoryId: "",
    status: "draft",
    categories: sampleCategories,
    onNameChange: () => {},
    onSlugChange: () => {},
    onDescriptionChange: () => {},
    onCategoryChange: () => {},
    onStatusChange: () => {},
  },
};
