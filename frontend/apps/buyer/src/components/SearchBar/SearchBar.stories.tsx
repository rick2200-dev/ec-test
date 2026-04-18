import type { Meta, StoryObj } from "@storybook/react";
import { SearchBarPresenter } from "./SearchBar.presenter";

const meta: Meta<typeof SearchBarPresenter> = {
  component: SearchBarPresenter,
  title: "Buyer/SearchBar",
};
export default meta;

type Story = StoryObj<typeof SearchBarPresenter>;

const labels = {
  placeholder: "商品を検索...",
  searchAriaLabel: "検索",
};

export const Empty: Story = {
  args: {
    ...labels,
    value: "",
    suggestions: [],
    loading: false,
    onChange: () => {},
    onSubmit: () => {},
    onPickSuggestion: () => {},
  },
};

export const TypingNoSuggestions: Story = {
  args: {
    ...labels,
    value: "head",
    suggestions: [],
    loading: false,
    onChange: () => {},
    onSubmit: () => {},
    onPickSuggestion: () => {},
  },
};

export const Loading: Story = {
  args: {
    ...labels,
    value: "head",
    suggestions: [],
    loading: true,
    onChange: () => {},
    onSubmit: () => {},
    onPickSuggestion: () => {},
  },
};

export const WithSuggestions: Story = {
  args: {
    ...labels,
    value: "head",
    suggestions: ["headphones", "headset", "headlamp"],
    loading: false,
    onChange: () => {},
    onSubmit: () => {},
    onPickSuggestion: () => {},
  },
};
