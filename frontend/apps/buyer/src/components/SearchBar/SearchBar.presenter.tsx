export interface SearchBarPresenterProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  onPickSuggestion: (text: string) => void;
  /** When non-empty, rendered below the input as a dropdown. */
  suggestions: string[];
  loading: boolean;
  placeholder: string;
  searchAriaLabel: string;
}

export function SearchBarPresenter({
  value,
  onChange,
  onSubmit,
  onPickSuggestion,
  suggestions,
  loading,
  placeholder,
  searchAriaLabel,
}: SearchBarPresenterProps) {
  return (
    <form
      role="search"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
      className="relative"
    >
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete="off"
        className="w-full rounded-full border border-gray-300 bg-gray-50 py-2 pl-4 pr-10 text-sm focus:border-blue-500 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
      />
      <button
        type="submit"
        aria-label={searchAriaLabel}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
      >
        <svg
          aria-hidden="true"
          className="h-5 w-5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
          />
        </svg>
      </button>
      {(suggestions.length > 0 || loading) && (
        <ul
          role="listbox"
          className="absolute left-0 right-0 top-full z-50 mt-1 max-h-72 overflow-y-auto rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg"
        >
          {loading && suggestions.length === 0 && <li className="px-3 py-2 text-gray-400">…</li>}
          {suggestions.map((s) => (
            <li key={s}>
              <button
                type="button"
                onClick={() => onPickSuggestion(s)}
                className="block w-full px-3 py-2 text-left text-gray-700 hover:bg-gray-50"
              >
                {s}
              </button>
            </li>
          ))}
        </ul>
      )}
    </form>
  );
}
