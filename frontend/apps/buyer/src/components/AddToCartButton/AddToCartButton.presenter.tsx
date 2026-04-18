export interface AddToCartButtonPresenterProps {
  label: string;
  loadingLabel: string;
  successLabel: string;
  loginCta: string;
  /** Disabled when no SKU is selected (e.g. the variant matrix is incomplete). */
  disabled: boolean;
  loading: boolean;
  /** Briefly true after a successful add — flips the button label. */
  succeeded: boolean;
  /** When set the button is replaced with a "log in to add" link. */
  needsLogin: boolean;
  loginHref: string;
  onClick: () => void;
  /** Optional inline error message rendered below the button. */
  errorMessage?: string;
}

export function AddToCartButtonPresenter({
  label,
  loadingLabel,
  successLabel,
  loginCta,
  disabled,
  loading,
  succeeded,
  needsLogin,
  loginHref,
  onClick,
  errorMessage,
}: AddToCartButtonPresenterProps) {
  if (needsLogin) {
    return (
      <a
        href={loginHref}
        className="mt-8 inline-flex w-full items-center justify-center rounded-lg border border-gray-300 bg-white px-8 py-3 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
      >
        {loginCta}
      </a>
    );
  }

  const text = loading ? loadingLabel : succeeded ? successLabel : label;

  return (
    <div className="mt-8">
      <button
        type="button"
        disabled={disabled || loading}
        onClick={onClick}
        className="w-full rounded-lg bg-blue-600 px-8 py-3 text-sm font-semibold text-white shadow-sm hover:bg-blue-700 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {text}
      </button>
      {errorMessage && (
        <p role="alert" className="mt-2 text-sm text-red-600">
          {errorMessage}
        </p>
      )}
    </div>
  );
}
