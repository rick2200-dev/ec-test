"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { searchSuggest } from "@/lib/api";
import { SearchBarPresenter } from "./SearchBar.presenter";

const DEBOUNCE_MS = 200;
const MIN_CHARS = 2;

export default function SearchBar() {
  const t = useTranslations();
  const router = useRouter();
  const [value, setValue] = useState("");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  // Debounced suggest fetch. Cancels any in-flight call before issuing a
  // new one — without this you get racy reorderings where slow responses
  // overwrite fast ones.
  useEffect(() => {
    if (value.trim().length < MIN_CHARS) {
      setSuggestions([]);
      setLoading(false);
      abortRef.current?.abort();
      return;
    }
    const controller = new AbortController();
    abortRef.current?.abort();
    abortRef.current = controller;
    setLoading(true);
    const handle = window.setTimeout(async () => {
      try {
        const res = await searchSuggest(value.trim(), { signal: controller.signal });
        if (controller.signal.aborted) return;
        setSuggestions(res.suggestions.map((s) => s.text));
      } catch (err) {
        if ((err as { name?: string })?.name === "AbortError") return;
        // Silently swallow other errors — the search bar should not nag.
        setSuggestions([]);
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, DEBOUNCE_MS);
    return () => {
      window.clearTimeout(handle);
      controller.abort();
    };
  }, [value]);

  const submit = (term: string) => {
    const q = term.trim();
    if (!q) return;
    setSuggestions([]);
    router.push(`/search?q=${encodeURIComponent(q)}`);
  };

  return (
    <SearchBarPresenter
      value={value}
      onChange={setValue}
      onSubmit={() => submit(value)}
      onPickSuggestion={(s) => {
        setValue(s);
        submit(s);
      }}
      suggestions={suggestions}
      loading={loading}
      placeholder={t("header.searchPlaceholder")}
      searchAriaLabel={t("a11y.searchProducts")}
    />
  );
}
