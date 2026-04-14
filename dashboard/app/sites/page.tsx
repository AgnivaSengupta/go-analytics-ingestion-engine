"use client";

import DashboardLayout from "@/components/DashboardLayout";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import AddSiteDialog from "@/components/SiteAddDialog";
import SiteCard from "@/components/SiteCard";
import { Button } from "@/components/ui/button";
import { fetchWithAuth } from "@/lib/api";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

type Site = {
  id: string;
  name: string;
  url: string;
  created_at: string;
};

type ListSitesResponse = {
  sites: Site[];
};

export default function Sites() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadSites() {
      try {
        setLoading(true);
        setError(null);

        const response = await fetchWithAuth("/v1/sites", {
          method: "GET",
        });

        if (response.status === 401) {
          router.replace("/");
          return;
        }

        if (!response.ok) {
          throw new Error("Failed to load sites");
        }

        const data = (await response.json()) as ListSitesResponse;
        setSites(data.sites ?? []);
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load sites";
        setError(message);
      } finally {
        setLoading(false);
      }
    }

    loadSites();
  }, [router]);

  const handleCloseAddSiteDialog = () => {
    setOpen(false);
  };

  const handleSiteCreated = (site: Site) => {
    setSites((current) => [site, ...current]);
    setError(null);
    setOpen(false);
  };

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <main className="w-6xl h-full flex flex-col">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="font-serif text-4xl tracking-wide">Sites</h1>
            </div>

            <Button
              size="lg"
              className="text-lg font-serif tracking-widest px-4 py-4 cursor-pointer"
              onClick={() => setOpen(true)}
            >
              + Add Site
            </Button>
          </div>

          <div className="flex flex-wrap gap-6 p-8 bg-zinc-100 rounded-lg justify-start flex-1 overflow-y-auto min-h-0 my-10">
            {loading ? <p className="text-zinc-600">Loading sites...</p> : null}

            {!loading && error ? <p className="text-red-600">{error}</p> : null}

            {!loading && !error && sites.length === 0 ? (
              <p className="text-zinc-600">
                No sites yet. Create your first site.
              </p>
            ) : null}

            {!loading && !error
              ? sites.map((site) => (
                  <div key={site.id} onClick={() => router.push(`/sites/${site.id}`)}>
                    <SiteCard
                      key={site.id}
                      name={site.name}
                      url={site.url}
                      visitorCount="--"
                    />
                  </div>
                ))
              : null}
          </div>

          <AddSiteDialog
            open={open}
            onOpenChange={handleCloseAddSiteDialog}
            onSiteCreated={handleSiteCreated}
          />
        </main>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
