import { getTranslations } from "next-intl/server";

// Platform-level settings do not have a backend surface yet. The page
// exists so the sidebar link resolves instead of 404-ing; concrete
// controls land here as the ops needs crystallize (branding, default
// commission rate, maintenance flags, etc.).
export default async function AdminSettingsPage() {
  const t = await getTranslations();
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("settings.title")}</h2>
        <p className="text-text-secondary mt-1">{t("settings.description")}</p>
      </div>

      <div className="bg-white rounded-lg border border-border shadow-sm p-6">
        <p className="text-sm text-text-secondary">{t("settings.empty")}</p>
      </div>
    </div>
  );
}
