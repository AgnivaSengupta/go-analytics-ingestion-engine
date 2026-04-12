import React from "react";
import DashboardLayout from "@/components/DashboardLayout";
import { MetricCard } from "@/components/MetricCard";
import { AnalyticsChart } from "@/components/AnalyticsChart";
import { TopChannels } from "@/components/TopChannels";

export default function SiteOverviewPage() {
  return (
    <DashboardLayout>
      <main className="max-w-[1400px] mx-auto pb-12 w-full bg-background font-sans">
        
        {/* Page Header */}
        <div className="mb-8 border-b border-gray-100 pb-4">
          <h1 className="text-5xl font-serif tracking-wider text-gray-900 mb-2">Overview</h1>
          <p className="text-gray-500 text-[15px] font-medium">Monitor campaign activity, audience growth, and key metrics at a glance.</p>
        </div>

        {/* Top Metric Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
          <MetricCard
            title="Visitors"
            value="518"
            trend={5.2}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="m15 15 6-6m-6 6v-4m0 4h4M4 4h16m-16 6h16M4 16h8" />
                <path d="M13 13.5 13.5 13" />
                <path d="M11 21A8 8 0 0 1 3 13V5" />
                <circle cx="15" cy="5" r="2" />
              </svg>
            }
          />
          <MetricCard
            title="Page Views"
            value="1584"
            trend={2.4}
            trendDirection="down"
            trendText="From last month"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M8 3 4 7l4 4" />
                <path d="M4 7h16" />
                <path d="m16 21 4-4-4-4" />
                <path d="M20 17H4" />
              </svg>
            }
          />
          <MetricCard
            title="Bounce Rate"
            value="10,573"
            trend={3.6}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
                <line x1="3" x2="21" y1="9" y2="9" />
                <line x1="9" x2="9" y1="21" y2="9" />
              </svg>
            }
          />
          
          <MetricCard
            title="Session Time"
            value="10,573"
            trend={3.6}
            trendDirection="up"
            trendText="From last month"
            icon={
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
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
              <h2 className="text-3xl font-serif text-gray-900 tracking-wide">Analytics</h2>
              <div className="relative">
                <select className="appearance-none bg-white border border-gray-200 text-gray-700 text-sm font-medium rounded-lg px-4 py-2 pr-8 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 cursor-pointer shadow-sm">
                  <option>6 months</option>
                  <option>3 months</option>
                  <option>1 month</option>
                </select>
                <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-gray-500">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="m6 9 6 6 6-6" />
                  </svg>
                </div>
              </div>
            </div>
            
            <div className="flex flex-col bg-card px-6 py-4 rounded-md">
              <div className="flex gap-16 mb-8">
                <div>
                  <div className="text-gray-500 font-medium text-[15px] mb-2">Page views</div>
                  <div className="flex items-baseline gap-3">
                    <span className="text-3xl font-semibold font-serif text-gray-900 tracking-wide">24,834</span>
                    <div className="flex items-center gap-1 text-[13px] font-semibold text-green-600 bg-green-50 px-1.5 py-0.5 rounded-md border border-green-100">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="m5 12 7-7 7 7"/>
                        <path d="M12 19V5"/>
                      </svg>
                      2.1%
                    </div>
                  </div>
                </div>
                <div>
                  <div className="text-gray-500 font-medium text-[15px] mb-2">Visit durations</div>
                  <div className="flex items-baseline gap-3">
                    <span className="text-3xl font-semibold font-serif text-gray-900 tracking-wide">
                      5m <span className="ml-[2px]">31s</span>
                    </span>
                    <div className="flex items-center gap-1 text-[13px] font-semibold text-red-600 bg-red-50 px-1.5 py-0.5 rounded-md border border-red-100">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="m19 12-7 7-7-7"/>
                        <path d="M12 5v14"/>
                      </svg>
                      2.4%
                    </div>
                  </div>
                </div>
  
                 <div className="ml-auto flex gap-4 mt-2">
                   <div className="flex items-center gap-2 text-gray-500 text-sm font-medium">
                     <div className="w-2.5 h-2.5 rounded-sm bg-indigo-600"></div> Organic
                   </div>
                   <div className="flex items-center gap-2 text-gray-500 text-sm font-medium">
                     <div className="w-2.5 h-2.5 rounded-sm bg-teal-400/80"></div> Referral
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
            <TopChannels />
          </div>
          
          <div>
            <div className="flex flex-col h-full bg-[#F8FAFB] rounded-md border border-gray-200 shadow-sm p-4">
              <h3 className="text-2xl font-serif text-gray-900 mb-2 px-1">Top Pages</h3>
        
              <div className="bg-card rounded-md p-1">
                
              </div>
            </div>
          </div>
        </div>
        
        <div className="flex flex-col bg-[#F8FAFB] rounded-md border border-gray-200 shadow-sm p-4 h-[350px]">
          <h3 className="text-2xl font-serif text-gray-900 mb-2 px-1">Countries</h3>
        </div>
        
      </main>
    </DashboardLayout>
  );
}
