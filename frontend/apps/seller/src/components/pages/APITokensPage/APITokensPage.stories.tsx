import type { Meta, StoryObj } from "@storybook/react";
import { APITokensPagePresenter } from "./APITokensPage.presenter";

// Storybook for the APITokensPage presenter. All stories pass
// hard-coded props — the presenter is pure, so no fetch mocks or
// provider wrappers are needed. Container-owned interactivity
// (opening dialogs, typing, copying) is represented via multiple
// stories that fix `open=true` on a specific dialog.

const meta: Meta<typeof APITokensPagePresenter> = {
  title: "Seller/Pages/APITokensPage",
  component: APITokensPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof APITokensPagePresenter>;

const noop = () => {};

const baseCreateDialog = {
  open: false,
  title: "Create API token",
  description: "Tokens grant programmatic access to your seller data.",
  nameLabel: "Name",
  namePlaceholder: "e.g. Inventory sync job",
  scopesLabel: "Scopes",
  scopeOptions: [
    { value: "products:read", label: "Read products" },
    { value: "products:write", label: "Write products" },
    { value: "orders:read", label: "Read orders" },
    { value: "orders:write", label: "Write orders" },
    { value: "inventory:read", label: "Read inventory" },
    { value: "inventory:write", label: "Write inventory" },
  ],
  expiresAtLabel: "Expires at (optional)",
  expiresAtHint: "Leave empty for no expiration.",
  cancelLabel: "Cancel",
  submitLabel: "Create",
  submittingLabel: "Creating...",
  isSubmitting: false,
  form: { name: "", selectedScopes: [], expiresAt: "" },
  onNameChange: noop,
  onScopeToggle: noop,
  onExpiresAtChange: noop,
  onCancel: noop,
  onSubmit: noop,
};

const baseRevealDialog = {
  open: false,
  title: "Your new token",
  description: "Copy the token and store it somewhere safe — it will not be shown again.",
  plaintextLabel: "Token",
  // Deliberately not using the real `sk_live_` prefix here: GitHub push
  // protection matches that pattern as a Highnote live key. This is a
  // Storybook fixture, so any clearly-fake value is fine.
  plaintextValue: "DEMO-TOKEN-PLACEHOLDER-xxxxxxxx-xxxxxxxx-for-storybook-only",
  copyLabel: "Copy",
  copiedLabel: "Copied",
  isCopied: false,
  dismissLabel: "I've saved it",
  warningTitle: "This is your only chance to copy this token.",
  warningBody: "Close this dialog and the plaintext is gone forever.",
  onCopy: noop,
  onDismiss: noop,
};

const baseConfirmRevoke = {
  open: false,
  title: "Revoke token",
  description: "This immediately stops the token from authenticating. This cannot be undone.",
  confirmPromptPrefix: "Type",
  confirmPromptSuffix: "to confirm.",
  confirmNameTarget: "Inventory sync",
  confirmInputValue: "",
  cancelLabel: "Cancel",
  revokeLabel: "Revoke",
  revokingLabel: "Revoking...",
  isRevoking: false,
  canConfirm: false,
  onConfirmInputChange: noop,
  onCancel: noop,
  onConfirm: noop,
};

const baseArgs = {
  heading: {
    title: "API Tokens",
    description: "Issue and manage access tokens for programmatic API access.",
    newTokenLabel: "New token",
  },
  isLoading: false,
  loadingLabel: "Loading tokens...",
  onCreateClick: noop,
  table: {
    columnLabels: {
      name: "Name",
      scopes: "Scopes",
      lastUsed: "Last used",
      created: "Created",
      expires: "Expires",
      status: "Status",
      actions: "Actions",
    },
    emptyLabel: "No API tokens yet.",
    revokeLabel: "Revoke",
    onRevoke: noop,
    rows: [
      {
        id: "1",
        name: "Inventory sync",
        scopesLabel: "inventory:read, inventory:write",
        status: "active" as const,
        statusLabel: "Active",
        statusClassName: "bg-green-100 text-green-800",
        lastUsedLabel: "Apr 10, 08:14",
        createdAtLabel: "Apr 1, 09:00",
        expiresAtLabel: "No expiry",
      },
      {
        id: "2",
        name: "Analytics reader",
        scopesLabel: "orders:read, products:read",
        status: "active" as const,
        statusLabel: "Active",
        statusClassName: "bg-green-100 text-green-800",
        lastUsedLabel: "Apr 9, 15:42",
        createdAtLabel: "Mar 15, 12:00",
        expiresAtLabel: "Dec 31, 23:59",
      },
      {
        id: "3",
        name: "Old ERP bridge",
        scopesLabel: "orders:read, orders:write",
        status: "revoked" as const,
        statusLabel: "Revoked",
        statusClassName: "bg-red-100 text-red-800",
        lastUsedLabel: "Mar 20, 11:05",
        createdAtLabel: "Feb 1, 10:00",
        expiresAtLabel: "No expiry",
      },
    ],
  },
  createDialog: baseCreateDialog,
  revealDialog: baseRevealDialog,
  confirmRevokeDialog: baseConfirmRevoke,
};

export const Default: Story = { args: baseArgs };

export const Empty: Story = {
  args: {
    ...baseArgs,
    table: { ...baseArgs.table, rows: [] },
  },
};

export const Loading: Story = {
  args: {
    ...baseArgs,
    isLoading: true,
    table: { ...baseArgs.table, rows: [] },
  },
};

export const WithErrorBanner: Story = {
  args: {
    ...baseArgs,
    errorMessage: "Failed to load API tokens. Please try again.",
  },
};

export const CreateDialogOpen: Story = {
  args: {
    ...baseArgs,
    createDialog: {
      ...baseCreateDialog,
      open: true,
      form: {
        name: "Inventory sync",
        selectedScopes: ["inventory:read", "inventory:write"],
        expiresAt: "",
      },
    },
  },
};

export const PlaintextRevealOpen: Story = {
  args: {
    ...baseArgs,
    revealDialog: {
      ...baseRevealDialog,
      open: true,
    },
  },
};

export const PlaintextRevealCopied: Story = {
  args: {
    ...baseArgs,
    revealDialog: {
      ...baseRevealDialog,
      open: true,
      isCopied: true,
    },
  },
};

export const ConfirmRevokeOpen: Story = {
  args: {
    ...baseArgs,
    confirmRevokeDialog: {
      ...baseConfirmRevoke,
      open: true,
      confirmInputValue: "Inventory sync",
      canConfirm: true,
    },
  },
};
