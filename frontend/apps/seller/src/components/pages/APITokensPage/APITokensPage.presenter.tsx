// APITokensPage presenter. Pure view: accepts a fully-formed view
// model and a set of event callbacks, renders the page. No hooks, no
// fetches — the Container (../../app/(dashboard)/settings/api-tokens/page.tsx)
// owns all state and effects. This is what Storybook exercises.
//
// The presenter intentionally renders ALL dialogs unconditionally but
// hides them via `open=false`; the parent flips `open` to true when a
// user intent triggers it. This keeps the component tree stable and
// avoids focus-management bugs from conditional mounts.

export type APITokenStatus = "active" | "revoked" | "expired";

export interface APITokenRow {
  id: string;
  name: string;
  scopesLabel: string;
  status: APITokenStatus;
  statusLabel: string;
  statusClassName: string;
  lastUsedLabel: string;
  createdAtLabel: string;
  expiresAtLabel: string;
}

export interface ScopeOption {
  value: string;
  label: string;
}

export interface CreateFormState {
  name: string;
  // Scope values selected, in the order they appear in scopeOptions.
  selectedScopes: string[];
  // Optional ISO-8601 string; empty = never expires.
  expiresAt: string;
}

export interface APITokensPagePresenterProps {
  heading: {
    title: string;
    description: string;
    newTokenLabel: string;
  };
  table: {
    columnLabels: {
      name: string;
      scopes: string;
      lastUsed: string;
      created: string;
      expires: string;
      status: string;
      actions: string;
    };
    emptyLabel: string;
    revokeLabel: string;
    rows: APITokenRow[];
    onRevoke: (id: string) => void;
  };
  loadingLabel?: string;
  errorMessage?: string;
  isLoading: boolean;

  onCreateClick: () => void;

  createDialog: {
    open: boolean;
    title: string;
    description: string;
    nameLabel: string;
    namePlaceholder: string;
    scopesLabel: string;
    scopeOptions: ScopeOption[];
    expiresAtLabel: string;
    expiresAtHint: string;
    cancelLabel: string;
    submitLabel: string;
    submittingLabel: string;
    isSubmitting: boolean;
    errorMessage?: string;
    form: CreateFormState;
    onNameChange: (value: string) => void;
    onScopeToggle: (scope: string) => void;
    onExpiresAtChange: (value: string) => void;
    onCancel: () => void;
    onSubmit: () => void;
  };

  revealDialog: {
    open: boolean;
    title: string;
    description: string;
    plaintextLabel: string;
    plaintextValue: string;
    copyLabel: string;
    copiedLabel: string;
    isCopied: boolean;
    dismissLabel: string;
    warningTitle: string;
    warningBody: string;
    onCopy: () => void;
    onDismiss: () => void;
  };

  confirmRevokeDialog: {
    open: boolean;
    title: string;
    description: string;
    confirmPromptPrefix: string;
    confirmPromptSuffix: string;
    confirmNameTarget: string;
    confirmInputValue: string;
    cancelLabel: string;
    revokeLabel: string;
    revokingLabel: string;
    isRevoking: boolean;
    canConfirm: boolean;
    onConfirmInputChange: (value: string) => void;
    onCancel: () => void;
    onConfirm: () => void;
  };
}

export function APITokensPagePresenter(props: APITokensPagePresenterProps) {
  const { heading, table, createDialog, revealDialog, confirmRevokeDialog } = props;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{heading.title}</h2>
          <p className="text-text-secondary mt-1">{heading.description}</p>
        </div>
        <button
          type="button"
          onClick={props.onCreateClick}
          className="px-4 py-2 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors"
        >
          {heading.newTokenLabel}
        </button>
      </div>

      {props.errorMessage && (
        <div className="bg-red-50 border border-red-200 rounded-lg px-4 py-3 text-sm text-danger">
          {props.errorMessage}
        </div>
      )}

      <div className="bg-white rounded-lg border border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.name}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.scopes}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.lastUsed}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.created}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.expires}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.status}
                </th>
                <th className="text-right px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {table.columnLabels.actions}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {table.rows.length === 0 && !props.isLoading && (
                <tr>
                  <td
                    colSpan={7}
                    className="px-6 py-12 text-center text-sm text-text-secondary"
                  >
                    {table.emptyLabel}
                  </td>
                </tr>
              )}
              {props.isLoading && (
                <tr>
                  <td
                    colSpan={7}
                    className="px-6 py-12 text-center text-sm text-text-secondary"
                  >
                    {props.loadingLabel}
                  </td>
                </tr>
              )}
              {table.rows.map((row) => (
                <tr key={row.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {row.name}
                  </td>
                  <td className="px-6 py-4 text-xs font-mono text-text-secondary">
                    {row.scopesLabel}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {row.lastUsedLabel}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {row.createdAtLabel}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {row.expiresAtLabel}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${row.statusClassName}`}
                    >
                      {row.statusLabel}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right">
                    {row.status === "active" && (
                      <button
                        type="button"
                        onClick={() => table.onRevoke(row.id)}
                        className="text-sm text-danger hover:underline font-medium"
                      >
                        {table.revokeLabel}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <CreateAPITokenDialog dialog={createDialog} />
      <PlaintextRevealDialog dialog={revealDialog} />
      <ConfirmRevokeDialog dialog={confirmRevokeDialog} />
    </div>
  );
}

// ----------------------------------------------------------------------------
// Sub-components. These are declared in the same file because they are
// only ever used together, and splitting them would quadruple the number
// of i18n prop bundles the container has to assemble.
// ----------------------------------------------------------------------------

function CreateAPITokenDialog({
  dialog,
}: {
  dialog: APITokensPagePresenterProps["createDialog"];
}) {
  if (!dialog.open) return null;
  return (
    <DialogShell title={dialog.title} onClose={dialog.onCancel}>
      <p className="text-sm text-text-secondary">{dialog.description}</p>

      <div className="space-y-2">
        <label
          htmlFor="create-token-name"
          className="block text-sm font-medium text-text-primary"
        >
          {dialog.nameLabel}
        </label>
        <input
          id="create-token-name"
          type="text"
          value={dialog.form.name}
          onChange={(e) => dialog.onNameChange(e.target.value)}
          placeholder={dialog.namePlaceholder}
          className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:ring-2 focus:ring-accent focus:outline-none"
        />
      </div>

      <div className="space-y-2">
        <p className="block text-sm font-medium text-text-primary">
          {dialog.scopesLabel}
        </p>
        <div className="grid grid-cols-2 gap-2">
          {dialog.scopeOptions.map((scope) => {
            const checked = dialog.form.selectedScopes.includes(scope.value);
            return (
              <label
                key={scope.value}
                className="flex items-center gap-2 px-3 py-2 border border-border rounded-lg cursor-pointer hover:bg-surface-hover"
              >
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => dialog.onScopeToggle(scope.value)}
                  className="rounded border-border"
                />
                <span className="text-sm text-text-primary">{scope.label}</span>
              </label>
            );
          })}
        </div>
      </div>

      <div className="space-y-2">
        <label
          htmlFor="create-token-expires"
          className="block text-sm font-medium text-text-primary"
        >
          {dialog.expiresAtLabel}
        </label>
        <input
          id="create-token-expires"
          type="datetime-local"
          value={dialog.form.expiresAt}
          onChange={(e) => dialog.onExpiresAtChange(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:ring-2 focus:ring-accent focus:outline-none"
        />
        <p className="text-xs text-text-secondary">{dialog.expiresAtHint}</p>
      </div>

      {dialog.errorMessage && (
        <div className="bg-red-50 border border-red-200 rounded-lg px-4 py-3 text-sm text-danger">
          {dialog.errorMessage}
        </div>
      )}

      <div className="flex items-center justify-end gap-3 pt-2">
        <button
          type="button"
          onClick={dialog.onCancel}
          className="px-4 py-2 border border-border rounded-lg text-sm font-medium text-text-primary hover:bg-surface-hover transition-colors"
        >
          {dialog.cancelLabel}
        </button>
        <button
          type="button"
          onClick={dialog.onSubmit}
          disabled={dialog.isSubmitting}
          className="px-4 py-2 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {dialog.isSubmitting ? dialog.submittingLabel : dialog.submitLabel}
        </button>
      </div>
    </DialogShell>
  );
}

function PlaintextRevealDialog({
  dialog,
}: {
  dialog: APITokensPagePresenterProps["revealDialog"];
}) {
  if (!dialog.open) return null;
  return (
    <DialogShell title={dialog.title} onClose={dialog.onDismiss}>
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg px-4 py-3">
        <p className="text-sm font-semibold text-yellow-900">{dialog.warningTitle}</p>
        <p className="text-sm text-yellow-800 mt-1">{dialog.warningBody}</p>
      </div>

      <div className="space-y-2">
        <p className="block text-sm font-medium text-text-primary">
          {dialog.plaintextLabel}
        </p>
        <div className="flex gap-2">
          <code className="flex-1 block px-3 py-2 bg-surface border border-border rounded-lg text-xs font-mono text-text-primary break-all">
            {dialog.plaintextValue}
          </code>
          <button
            type="button"
            onClick={dialog.onCopy}
            className="px-4 py-2 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors whitespace-nowrap"
          >
            {dialog.isCopied ? dialog.copiedLabel : dialog.copyLabel}
          </button>
        </div>
      </div>

      <p className="text-sm text-text-secondary">{dialog.description}</p>

      <div className="flex items-center justify-end pt-2">
        <button
          type="button"
          onClick={dialog.onDismiss}
          className="px-4 py-2 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors"
        >
          {dialog.dismissLabel}
        </button>
      </div>
    </DialogShell>
  );
}

function ConfirmRevokeDialog({
  dialog,
}: {
  dialog: APITokensPagePresenterProps["confirmRevokeDialog"];
}) {
  if (!dialog.open) return null;
  return (
    <DialogShell title={dialog.title} onClose={dialog.onCancel}>
      <p className="text-sm text-text-secondary">{dialog.description}</p>

      <div className="space-y-2">
        <p className="text-sm text-text-primary">
          {dialog.confirmPromptPrefix}{" "}
          <code className="px-1 py-0.5 bg-surface border border-border rounded text-xs font-mono">
            {dialog.confirmNameTarget}
          </code>{" "}
          {dialog.confirmPromptSuffix}
        </p>
        <input
          type="text"
          value={dialog.confirmInputValue}
          onChange={(e) => dialog.onConfirmInputChange(e.target.value)}
          className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:ring-2 focus:ring-danger focus:outline-none"
        />
      </div>

      <div className="flex items-center justify-end gap-3 pt-2">
        <button
          type="button"
          onClick={dialog.onCancel}
          className="px-4 py-2 border border-border rounded-lg text-sm font-medium text-text-primary hover:bg-surface-hover transition-colors"
        >
          {dialog.cancelLabel}
        </button>
        <button
          type="button"
          onClick={dialog.onConfirm}
          disabled={!dialog.canConfirm || dialog.isRevoking}
          className="px-4 py-2 bg-danger hover:opacity-90 text-white rounded-lg text-sm font-medium transition-opacity disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {dialog.isRevoking ? dialog.revokingLabel : dialog.revokeLabel}
        </button>
      </div>
    </DialogShell>
  );
}

// DialogShell renders the backdrop + centered card used by all three
// dialogs. Kept local to this file because no other seller-app page
// needs a dialog yet — the minute a second page wants one, lift this
// into src/components/ui/Dialog.
//
// The backdrop is a real <button> sibling (not a wrapping div with
// onClick) so the a11y linter is satisfied and keyboard users can
// reach the close control via the normal tab order. The modal card
// sits above it via `relative` + DOM order, so clicks inside the
// card never reach the backdrop button.
function DialogShell({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
      <button
        type="button"
        aria-label="Close dialog"
        tabIndex={-1}
        className="absolute inset-0 bg-black/40 cursor-default"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        className="relative bg-white rounded-lg border border-border shadow-lg w-full max-w-xl max-h-[90vh] overflow-y-auto"
      >
        <div className="px-6 py-4 border-b border-border">
          <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
        </div>
        <div className="px-6 py-4 space-y-4">{children}</div>
      </div>
    </div>
  );
}
