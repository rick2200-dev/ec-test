"use client";

// API tokens settings page. Client component: owns all local state
// (token list, modal visibility, create-form values, loading flags,
// error messages) and delegates rendering to APITokensPagePresenter.
//
// This page is intentionally NOT split as a server-component Container
// + pure Presenter like the Dashboard — every interaction here mutates
// client state (open a modal, type into a form, click to revoke), so a
// server Container would just forward its job to a child client
// component anyway. The Container/Presenter boundary we care about
// here is pure-presenter vs state-owning-client-component, and the
// presenter file does that job.

import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslations } from "next-intl";

import {
  APIError,
  createAPIToken,
  listAPITokens,
  revokeAPIToken,
  type APIToken,
  type CreateAPITokenRequest,
} from "@/lib/api-client";
import {
  APITokensPagePresenter,
  type APITokenRow,
  type APITokenStatus,
  type CreateFormState,
  type ScopeOption,
} from "@/components/pages/APITokensPage/APITokensPage.presenter";

// Closed set of scopes. Must match
// backend/services/auth/internal/domain/api_token.go AllAPITokenScopes.
// Hard-coded on the client too so the checkbox list renders without a
// network round-trip — the value set never churns.
const ALL_SCOPES = [
  "products:read",
  "products:write",
  "orders:read",
  "orders:write",
  "inventory:read",
  "inventory:write",
] as const;

// formatDateTime renders timestamps in the same locale the rest of the
// seller app uses. Returns the fallback string on null/undefined.
function formatDateTime(
  iso: string | null | undefined,
  fallback: string,
): string {
  if (!iso) return fallback;
  try {
    return new Date(iso).toLocaleString("ja-JP", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return fallback;
  }
}

// computeStatus derives the display status from the backend-persisted
// timestamps. The server also reports this via
// SellerAPIToken.Status(time.Now) but we do it client-side so the list
// updates without a re-fetch after a successful revoke.
function computeStatus(token: APIToken): APITokenStatus {
  if (token.revoked_at) return "revoked";
  if (token.expires_at && new Date(token.expires_at).getTime() <= Date.now()) {
    return "expired";
  }
  return "active";
}

const STATUS_STYLE: Record<APITokenStatus, string> = {
  active: "bg-green-100 text-green-800",
  revoked: "bg-red-100 text-red-800",
  expired: "bg-gray-100 text-gray-700",
};

export default function APITokensPage() {
  const t = useTranslations();

  // Build scope option labels once per render so the Presenter receives
  // a stable, i18n-aware list of checkbox options.
  const scopeOptions: ScopeOption[] = useMemo(
    () =>
      ALL_SCOPES.map((value) => ({
        value,
        label: t(`apiTokens.scopeLabels.${value}`),
      })),
    [t],
  );

  // Token list + top-level loading/error state.
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [listError, setListError] = useState<string | undefined>(undefined);

  // Create dialog state.
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState<CreateFormState>({
    name: "",
    selectedScopes: [],
    expiresAt: "",
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | undefined>(undefined);

  // Plaintext reveal dialog state. The plaintext is stored here in memory
  // ONLY until the user dismisses the dialog — we intentionally drop it
  // on dismiss so a stale React state cannot leak the token in future
  // renders.
  const [plaintext, setPlaintext] = useState<string | null>(null);
  const [isCopied, setIsCopied] = useState(false);

  // Revoke confirmation dialog state.
  const [revokeTarget, setRevokeTarget] = useState<APIToken | null>(null);
  const [confirmInput, setConfirmInput] = useState("");
  const [isRevoking, setIsRevoking] = useState(false);

  const loadTokens = useCallback(async () => {
    setIsLoading(true);
    setListError(undefined);
    try {
      const items = await listAPITokens();
      setTokens(items);
    } catch (err) {
      setListError(
        err instanceof APIError
          ? err.message
          : t("apiTokens.errors.loadFailed"),
      );
    } finally {
      setIsLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadTokens();
  }, [loadTokens]);

  const openCreate = useCallback(() => {
    setCreateForm({ name: "", selectedScopes: [], expiresAt: "" });
    setCreateError(undefined);
    setCreateOpen(true);
  }, []);

  const closeCreate = useCallback(() => {
    setCreateOpen(false);
  }, []);

  const handleNameChange = useCallback((value: string) => {
    setCreateForm((prev) => ({ ...prev, name: value }));
  }, []);

  const handleScopeToggle = useCallback((scope: string) => {
    setCreateForm((prev) => {
      const has = prev.selectedScopes.includes(scope);
      return {
        ...prev,
        selectedScopes: has
          ? prev.selectedScopes.filter((s) => s !== scope)
          : [...prev.selectedScopes, scope],
      };
    });
  }, []);

  const handleExpiresAtChange = useCallback((value: string) => {
    setCreateForm((prev) => ({ ...prev, expiresAt: value }));
  }, []);

  const handleCreateSubmit = useCallback(async () => {
    // Minimal client-side validation. The server re-validates every
    // field — we only check here to avoid the round-trip on obvious
    // errors so the user sees feedback immediately.
    if (!createForm.name.trim()) {
      setCreateError(t("apiTokens.errors.nameRequired"));
      return;
    }
    if (createForm.selectedScopes.length === 0) {
      setCreateError(t("apiTokens.errors.scopeRequired"));
      return;
    }

    setIsSubmitting(true);
    setCreateError(undefined);
    try {
      const req: CreateAPITokenRequest = {
        name: createForm.name.trim(),
        scopes: createForm.selectedScopes,
        expires_at: createForm.expiresAt
          ? new Date(createForm.expiresAt).toISOString()
          : null,
      };
      const resp = await createAPIToken(req);
      setCreateOpen(false);
      setPlaintext(resp.token);
      setIsCopied(false);
      // Optimistically prepend the new token to the list so the user
      // can see it without waiting for a refresh. The next loadTokens
      // call will replace this with the server-authoritative copy.
      const { token: _plain, ...persisted } = resp;
      setTokens((prev) => [persisted, ...prev]);
    } catch (err) {
      setCreateError(
        err instanceof APIError
          ? err.message
          : t("apiTokens.errors.createFailed"),
      );
    } finally {
      setIsSubmitting(false);
    }
  }, [createForm, t]);

  const handleCopy = useCallback(async () => {
    if (!plaintext) return;
    try {
      await navigator.clipboard.writeText(plaintext);
      setIsCopied(true);
    } catch {
      // Clipboard API can fail in insecure contexts / Storybook. Swallow
      // and leave the button un-flipped; user can select+copy manually.
    }
  }, [plaintext]);

  const handleDismissReveal = useCallback(() => {
    setPlaintext(null);
    setIsCopied(false);
  }, []);

  const handleAskRevoke = useCallback(
    (id: string) => {
      const target = tokens.find((t) => t.id === id) ?? null;
      setRevokeTarget(target);
      setConfirmInput("");
    },
    [tokens],
  );

  const handleConfirmRevoke = useCallback(async () => {
    if (!revokeTarget) return;
    setIsRevoking(true);
    try {
      await revokeAPIToken(revokeTarget.id);
      setRevokeTarget(null);
      setConfirmInput("");
      // Refresh from the server — the response carries the fresh
      // revoked_at timestamp and keeps the client in lock-step.
      await loadTokens();
    } catch (err) {
      setListError(
        err instanceof APIError
          ? err.message
          : t("apiTokens.errors.revokeFailed"),
      );
      setRevokeTarget(null);
    } finally {
      setIsRevoking(false);
    }
  }, [revokeTarget, loadTokens, t]);

  const handleCancelRevoke = useCallback(() => {
    setRevokeTarget(null);
    setConfirmInput("");
  }, []);

  // Build the table rows. Done in the container so the Presenter stays
  // purely presentational.
  const rows: APITokenRow[] = useMemo(() => {
    const neverLabel = t("apiTokens.lastUsedNever");
    const noExpiryLabel = t("apiTokens.noExpiry");
    return tokens.map((token) => {
      const status = computeStatus(token);
      return {
        id: token.id,
        name: token.name,
        scopesLabel: token.scopes.join(", "),
        status,
        statusLabel: t(`apiTokens.status.${status}`),
        statusClassName: STATUS_STYLE[status],
        lastUsedLabel: formatDateTime(token.last_used_at, neverLabel),
        createdAtLabel: formatDateTime(token.created_at, ""),
        expiresAtLabel: formatDateTime(token.expires_at, noExpiryLabel),
      };
    });
  }, [tokens, t]);

  return (
    <APITokensPagePresenter
      heading={{
        title: t("apiTokens.title"),
        description: t("apiTokens.description"),
        newTokenLabel: t("apiTokens.newToken"),
      }}
      isLoading={isLoading}
      loadingLabel={t("apiTokens.loading")}
      errorMessage={listError}
      onCreateClick={openCreate}
      table={{
        columnLabels: {
          name: t("apiTokens.table.name"),
          scopes: t("apiTokens.table.scopes"),
          lastUsed: t("apiTokens.table.lastUsed"),
          created: t("apiTokens.table.created"),
          expires: t("apiTokens.table.expires"),
          status: t("apiTokens.table.status"),
          actions: t("apiTokens.table.actions"),
        },
        emptyLabel: t("apiTokens.empty"),
        revokeLabel: t("apiTokens.revoke"),
        rows,
        onRevoke: handleAskRevoke,
      }}
      createDialog={{
        open: createOpen,
        title: t("apiTokens.create.title"),
        description: t("apiTokens.create.description"),
        nameLabel: t("apiTokens.create.name"),
        namePlaceholder: t("apiTokens.create.namePlaceholder"),
        scopesLabel: t("apiTokens.create.scopes"),
        scopeOptions,
        expiresAtLabel: t("apiTokens.create.expiresAt"),
        expiresAtHint: t("apiTokens.create.expiresAtHint"),
        cancelLabel: t("apiTokens.create.cancel"),
        submitLabel: t("apiTokens.create.submit"),
        submittingLabel: t("apiTokens.create.submitting"),
        isSubmitting,
        errorMessage: createError,
        form: createForm,
        onNameChange: handleNameChange,
        onScopeToggle: handleScopeToggle,
        onExpiresAtChange: handleExpiresAtChange,
        onCancel: closeCreate,
        onSubmit: handleCreateSubmit,
      }}
      revealDialog={{
        open: plaintext !== null,
        title: t("apiTokens.reveal.title"),
        description: t("apiTokens.reveal.description"),
        plaintextLabel: t("apiTokens.reveal.plaintextLabel"),
        plaintextValue: plaintext ?? "",
        copyLabel: t("apiTokens.reveal.copy"),
        copiedLabel: t("apiTokens.reveal.copied"),
        isCopied,
        dismissLabel: t("apiTokens.reveal.dismiss"),
        warningTitle: t("apiTokens.reveal.warningTitle"),
        warningBody: t("apiTokens.reveal.warningBody"),
        onCopy: handleCopy,
        onDismiss: handleDismissReveal,
      }}
      confirmRevokeDialog={{
        open: revokeTarget !== null,
        title: t("apiTokens.confirmRevoke.title"),
        description: t("apiTokens.confirmRevoke.description"),
        confirmPromptPrefix: t("apiTokens.confirmRevoke.promptPrefix"),
        confirmPromptSuffix: t("apiTokens.confirmRevoke.promptSuffix"),
        confirmNameTarget: revokeTarget?.name ?? "",
        confirmInputValue: confirmInput,
        cancelLabel: t("apiTokens.confirmRevoke.cancel"),
        revokeLabel: t("apiTokens.confirmRevoke.revoke"),
        revokingLabel: t("apiTokens.confirmRevoke.revoking"),
        isRevoking,
        canConfirm: confirmInput === (revokeTarget?.name ?? ""),
        onConfirmInputChange: setConfirmInput,
        onCancel: handleCancelRevoke,
        onConfirm: handleConfirmRevoke,
      }}
    />
  );
}
