import { Suspense } from "react";
import SearchPage from "@/components/pages/SearchPage";

// useSearchParams forces a Suspense boundary at build time; explicit
// dynamic prevents Next from trying to statically render this route.
export const dynamic = "force-dynamic";

export default function Page() {
  return (
    <Suspense>
      <SearchPage />
    </Suspense>
  );
}
