"use client";

import React, { useEffect, useState } from "react";
import DashboardLayout from "@/components/DashboardLayout";
import { MetricCard } from "@/components/MetricCard";
import { AnalyticsChart } from "@/components/AnalyticsChart";
import { TopChannels } from "@/components/TopChannels";
import { useParams } from "next/navigation";
import { fetchWithAuth } from "@/lib/api";

type OverviewTotals = {
  pageviews: number;
  visitors: number;
  sessions: number;
  bounce_rate: number;
  avg_time_seconds: number;
};

type TimeRange = {
  from: string;
  to: string;
  interval: string;
};

type OverviewTimeseries = {
  bucket_start: string;
  pageviews: number;
  visitors: number;
  sessions: number;
};

type OverviewType = {
  site_id: string;
  range: TimeRange;
  totals: OverviewTotals;
  timeseries: OverviewTimeseries[];
};

type TopPageType = {
  page_url: string;
  page_path: string;
  pageviews: number;
  visitors: number;
  avg_time_seconds: number;
  entry_count: number;
};

type PageResType = {
  site_id: string;
  range: TimeRange;
  pages: TopPageType[];
  next_cursor: string | null;
};

export type TopSourcesType = {
  source: string;
  medium: string;
  campaign: string;
  referrer: string;
  visitors: number;
  sessions: number;
};

type SourceResType = {
  site_id: string;
  range: TimeRange;
  sources: TopSourcesType[];
  next_cursor: string | null;
};

export default function SiteOverviewPage() {
  const params = useParams();
  const siteId = params.site_id as string;

  const [from, setFrom] = useState("2026-04-01T00:00:00Z");
  const [to, setTo] = useState("2026-04-13T23:59:59Z");
  const [interval, setInterval] = useState<"day" | "hour">("day");

  const [overview, setOverview] = useState<OverviewType | null>(null);
  const [pages, setPages] = useState<TopPageType[]>([]);
  const [sources, setSources] = useState<TopSourcesType[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadOverviewDate() {
      try {
        setLoading(true);
        setError(null);

        const [overviewRes, pagesRes, sourcesRes] = await Promise.all([
          fetchWithAuth(
            `/v1/sites/${siteId}/overview?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&interval=${interval}`,
          ),

          fetchWithAuth(
            `/v1/sites/${siteId}/pages?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&interval=${interval}`,
          ),

          fetchWithAuth(
            `/v1/sites/${siteId}/sources?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&interval=${interval}`,
          ),
        ]);

        if (!overviewRes.ok || !pagesRes.ok || !sourcesRes.ok) {
          throw new Error("Failed to load analytics data");
        }

        const overviewData = (await overviewRes.json()) as OverviewType;
        const pagesData = (await pagesRes.json()) as PageResType;
        const sourcesData = (await sourcesRes.json()) as SourceResType;
        

        setOverview(overviewData);
        console.log("overview: ", overview)
        setPages(pagesData.pages ?? []);
        setSources(sourcesData.sources ?? []);
      } catch (error) {
        setError(
          error instanceof Error ? error.message : "Something went wrong",
        );
      } finally {
        setLoading(false);
      }
    }

    if (siteId) {
      loadOverviewDate();
    }
  }, [siteId, from, to, interval]);

  console.log("overview: ", overview)

  return (
    <DashboardLayout>
      <main className="max-w-[1400px] mx-auto pb-12 w-full bg-background font-sans overflow-y-auto">
        {/* Page Header */}
        <div className="mb-8 border-b border-gray-100 pb-4">
          <h1 className="text-5xl font-serif tracking-wider text-gray-900 mb-2">
            Overview
          </h1>
          <p className="text-gray-500 text-[15px] font-medium">
            Monitor campaign activity, audience growth, and key metrics at a
            glance.
          </p>
        </div>
        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 md:grid-cols-5 gap-6 mb-6">
          <MetricCard
            title="Visitors"
            value={overview?.totals.visitors}
            trend={5.2}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="m15 15 6-6m-6 6v-4m0 4h4M4 4h16m-16 6h16M4 16h8" />
                <path d="M13 13.5 13.5 13" />
                <path d="M11 21A8 8 0 0 1 3 13V5" />
                <circle cx="15" cy="5" r="2" />
              </svg>
            }
          />
          <MetricCard
            title="Page Views"
            value={overview?.totals.pageviews}
            trend={2.4}
            trendDirection="down"
            trendText="From last month"
            icon={
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M8 3 4 7l4 4" />
                <path d="M4 7h16" />
                <path d="m16 21 4-4-4-4" />
                <path d="M20 17H4" />
              </svg>
            }
          />
          <MetricCard
            title="Bounce Rate"
            value={overview?.totals.bounce_rate}
            trend={3.6}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
                <line x1="3" x2="21" y1="9" y2="9" />
                <line x1="9" x2="9" y1="21" y2="9" />
              </svg>
            }
          />

          <MetricCard
            title="Sessions"
            value={overview?.totals.sessions}
            trend={3.6}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
                <line x1="3" x2="21" y1="9" y2="9" />
                <line x1="9" x2="9" y1="21" y2="9" />
              </svg>
            }
          />

          <MetricCard
            title="Avg Session Time"
            value={overview?.totals.avg_time_seconds}
            trend={3.6}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
                <line x1="3" x2="21" y1="9" y2="9" />
                <line x1="9" x2="9" y1="21" y2="9" />
              </svg>
            }
          />
        </div>

        {/* Main Content Area: Chart + Sidebar */}
        <div className="grid grid-cols-1 lg:grid-cols-1 mb-6">
          {/* Chart Section */}
          <div className="lg:col-span-2 bg-[#F8FAFB] rounded-md border border-gray-200 shadow-sm p-3 flex flex-col">
            <div className="flex items-center justify-between mb-4 px-2 pt-2">
              <h2 className="text-3xl font-serif text-gray-900 tracking-wide">
                Analytics
              </h2>
              <div className="relative">
                <select className="appearance-none bg-white border border-gray-200 text-gray-700 text-sm font-medium rounded-lg px-4 py-2 pr-8 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 cursor-pointer shadow-sm">
                  <option>6 months</option>
                  <option>3 months</option>
                  <option>1 month</option>
                </select>
                <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-gray-500">
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="m6 9 6 6 6-6" />
                  </svg>
                </div>
              </div>
            </div>

            <div className="flex flex-col bg-card px-6 py-4 rounded-md">
              <div className="flex gap-16 mb-8">
                <div>
                  <div className="text-gray-500 font-medium text-[15px] mb-2">
                    Page views
                  </div>
                  <div className="flex items-baseline gap-3">
                    <span className="text-3xl font-semibold font-serif text-gray-900 tracking-wide">
                      24,834
                    </span>
                    <div className="flex items-center gap-1 text-[13px] font-semibold text-green-600 bg-green-50 px-1.5 py-0.5 rounded-md border border-green-100">
                      <svg
                        width="12"
                        height="12"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <path d="m5 12 7-7 7 7" />
                        <path d="M12 19V5" />
                      </svg>
                      2.1%
                    </div>
                  </div>
                </div>
                <div>
                  <div className="text-gray-500 font-medium text-[15px] mb-2">
                    Visitors
                  </div>
                  <div className="flex items-baseline gap-3">
                    <span className="text-3xl font-semibold font-serif text-gray-900 tracking-wide">
                      5m <span className="ml-[2px]">31s</span>
                    </span>
                    <div className="flex items-center gap-1 text-[13px] font-semibold text-red-600 bg-red-50 px-1.5 py-0.5 rounded-md border border-red-100">
                      <svg
                        width="12"
                        height="12"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <path d="m19 12-7 7-7-7" />
                        <path d="M12 5v14" />
                      </svg>
                      2.4%
                    </div>
                  </div>
                </div>

                <div className="ml-auto flex gap-4 mt-2">
                  <div className="flex items-center gap-2 text-gray-500 text-sm font-medium">
                    <div className="w-2.5 h-2.5 rounded-sm bg-indigo-600"></div>{" "}
                    Page Views
                  </div>
                  <div className="flex items-center gap-2 text-gray-500 text-sm font-medium">
                    <div className="w-2.5 h-2.5 rounded-sm bg-teal-400/80"></div>{" "}
                    Visitors
                  </div>
                </div>
              </div>

              <div className="flex-1 min-h-0 relative -ml-4 -mr-2">
                <AnalyticsChart />
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          <div className="lg:col-span-1 h-full">
            <TopChannels sources={sources} />
          </div>

          <div className="flex flex-col h-full bg-[#F8FAFB] rounded-md border border-gray-200 shadow-sm p-4">
            <h3 className="text-2xl font-serif text-gray-900 mb-4 px-1">Top Pages</h3>
          
            <div className="bg-card rounded-md divide-y divide-gray-100">
              {pages.length === 0 ? (
                <p className="p-4 text-sm text-zinc-500">No page data for this range.</p>
              ) : (
                pages.map((page) => (
                  <div key={page.page_path} className="flex items-center justify-between p-4">
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-zinc-900 truncate">{page.page_path}</p>
                      <p className="text-xs text-zinc-500 truncate">{page.page_url}</p>
                    </div>
          
                    <div className="text-right">
                      <p className="text-sm font-semibold text-zinc-900">{page.pageviews}</p>
                      <p className="text-xs text-zinc-500">pageviews</p>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>  

        <div className="flex flex-col bg-[#F8FAFB] rounded-md border border-gray-200 shadow-sm p-4 h-[350px]">
          <h3 className="text-2xl font-serif text-gray-900 mb-2 px-1">
            Countries
          </h3>
        </div>
      </main>
    </DashboardLayout>
  );
}
