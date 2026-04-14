"use client";

import React from "react";
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  AreaChart,
} from "recharts";

const data = [
  { name: "Jan", organic: 55000, referral: -5000 },
  { name: "", organic: 52000, referral: 5000 },
  { name: "", organic: 62000, referral: 30000 },
  { name: "Feb", organic: 63000, referral: 25000 },
  { name: "", organic: 70000, referral: 12000 },
  { name: "", organic: 78000, referral: 22000 },
  { name: "Mar", organic: 95000, referral: 22000 },
  { name: "", organic: 88000, referral: 28000 },
  { name: "", organic: 68000, referral: 25000 },
  { name: "Apr", organic: 60000, referral: 30000 },
  { name: "", organic: 50000, referral: 32000 },
  { name: "", organic: 52000, referral: 42000 },
  { name: "May", organic: 45000, referral: 50000 },
  { name: "", organic: 35000, referral: 42000 },
  { name: "", organic: 20000, referral: 38000 },
  { name: "Jun", organic: 52000, referral: 25000 },
  { name: "", organic: 45000, referral: 22000 },
  { name: "", organic: 20000, referral: 10000 },
];

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const CustomTooltip = ({ active, payload, label }: any) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white px-4 py-3 border border-gray-100 rounded-xl shadow-lg text-sm flex flex-col gap-2 min-w-[250px]">
        <div className="flex items-center gap-2 text-gray-800 font-medium pb-1 border-b border-gray-50 mb-1">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-500">
            <rect width="18" height="18" x="3" y="4" rx="2" ry="2" />
            <line x1="16" x2="16" y1="2" y2="6" />
            <line x1="8" x2="8" y1="2" y2="6" />
            <line x1="3" x2="21" y1="10" y2="10" />
            <path d="M8 14h.01" />
            <path d="M12 14h.01" />
            <path d="M16 14h.01" />
            <path d="M8 18h.01" />
            <path d="M12 18h.01" />
            <path d="M16 18h.01" />
          </svg>
          {label || "Date"} 2025
        </div>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
             <span className="w-2.5 h-2.5 rounded-sm bg-indigo-600"></span>
             <span className="text-gray-500">Page views</span>
          </div>
          <span className="font-semibold text-gray-800">
            {payload[0].value.toLocaleString()}k
          </span>
        </div>
        <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
             <span className="w-2.5 h-2.5 rounded-sm bg-emerald-400"></span>
             <span className="text-gray-500">Visitors</span>
          </div>
          <span className="font-semibold text-gray-800">
            {payload[1].value.toLocaleString()}k
          </span>
        </div>
      </div>
    );
  }
  return null;
};

export function AnalyticsChart() {
  return (
    <div className="w-full h-full min-h-[300px]">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={data}
          margin={{ top: 20, right: 10, left: -20, bottom: 0 }}
        >
          <CartesianGrid strokeDasharray="0" vertical={false} stroke="#f3f4f6" />
          <XAxis
            dataKey="name"
            axisLine={false}
            tickLine={false}
            tick={{ fill: "#9ca3af", fontSize: 13 }}
            dy={10}
            interval="preserveStartEnd"
          />
          <YAxis
             axisLine={false}
             tickLine={false}
             tick={{ fill: "#9ca3af", fontSize: 13 }}
             ticks={[0, 25000, 50000, 100000]}
             tickFormatter={(val) => {
               if (val === 0) return "0";
               return `${val / 1000}k`;
             }}
          />
          <Tooltip
             content={<CustomTooltip />}
             cursor={{ stroke: '#f3f4f6', strokeWidth: 40, opacity: 0.5 }}
          />
          <Line
            type="monotone"
            dataKey="organic"
            stroke="#4f46e5"
            strokeWidth={2.5}
            dot={false}
            activeDot={{ r: 4, strokeWidth: 0, fill: "#4f46e5" }}
          />
          <Line
            type="monotone"
            dataKey="referral"
            stroke="#34d399"
            strokeWidth={2.5}
            dot={false}
            activeDot={{ r: 4, strokeWidth: 0, fill: "#34d399" }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
