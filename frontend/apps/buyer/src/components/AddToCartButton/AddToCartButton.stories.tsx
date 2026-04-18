import type { Meta, StoryObj } from "@storybook/react";
import { AddToCartButtonPresenter } from "./AddToCartButton.presenter";

const meta: Meta<typeof AddToCartButtonPresenter> = {
  component: AddToCartButtonPresenter,
  title: "Buyer/AddToCartButton",
};
export default meta;

type Story = StoryObj<typeof AddToCartButtonPresenter>;

const labels = {
  label: "カートに追加",
  loadingLabel: "追加中...",
  successLabel: "カートに追加しました",
  loginCta: "ログインして続ける",
  loginHref: "/auth/login?returnTo=%2Fproducts%2Fdemo",
} as const;

export const Idle: Story = {
  args: {
    ...labels,
    disabled: false,
    loading: false,
    succeeded: false,
    needsLogin: false,
    onClick: () => {},
  },
};

export const Loading: Story = {
  args: {
    ...labels,
    disabled: false,
    loading: true,
    succeeded: false,
    needsLogin: false,
    onClick: () => {},
  },
};

export const Succeeded: Story = {
  args: {
    ...labels,
    disabled: false,
    loading: false,
    succeeded: true,
    needsLogin: false,
    onClick: () => {},
  },
};

export const Disabled: Story = {
  args: {
    ...labels,
    disabled: true,
    loading: false,
    succeeded: false,
    needsLogin: false,
    onClick: () => {},
  },
};

export const NeedsLogin: Story = {
  args: {
    ...labels,
    disabled: false,
    loading: false,
    succeeded: false,
    needsLogin: true,
    onClick: () => {},
  },
};

export const WithError: Story = {
  args: {
    ...labels,
    disabled: false,
    loading: false,
    succeeded: false,
    needsLogin: false,
    onClick: () => {},
    errorMessage: "在庫が不足しています",
  },
};
