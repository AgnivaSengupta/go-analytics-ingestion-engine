import { MoreVerticalIcon } from "@hugeicons/core-free-icons";
import React from "react";
// import { MoreVerticalIcon } from "@hugeicons/react";

interface MetricCardProps {
  title: string;
  value: string;
  trend: number;
  trendDirection: "up" | "down";
  trendText: string;
  icon: React.ReactNode;
}

export function MetricCard({
  title,
  value,
  trend,
  trendDirection,
  trendText,
  icon,
}: MetricCardProps) {
  const isUp = trendDirection === "up";

  return (
    <div className="bg-[#F8FAFB] rounded-md border border-gray-200 p-1.5 flex flex-col gap-5 shadow-sm">
      <div className="flex flex-col gap-2 bg-card rounded-sm py-2 px-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 text-gray-600 font-medium text-[15px]">
            <div className="p-1.5 border border-gray-200 rounded-lg text-gray-500 bg-white shadow-sm flex items-center justify-center">
              {icon}
            </div>
            <span>{title}</span>
          </div>
          <button className="text-gray-400 hover:text-gray-600 cursor-pointer">
            {/* <MoreVerticalIcon size={20} /> */}
            ...
          </button>
        </div>
  
        <div className="text-4xl font-serif font-semibold text-gray-900 tracking-wider leading-none mt-1 mb-4 px-1">
          {value}
        </div>
      </div>


      <div className="flex items-center justify-between mt-auto px-2 pb-1">
        <div
          className={`flex items-center gap-1.5 text-[15px] font-medium ${isUp ? "text-green-600" : "text-red-600"
            }`}
        >
          {isUp ? (
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="opacity-80">
              <circle cx="12" cy="12" r="10" />
              <path d="m16 12-4-4-4 4" />
              <path d="M12 8v8" />
            </svg>
          ) : (
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="opacity-80">
              <circle cx="12" cy="12" r="10" />
              <path d="m8 12 4 4 4-4" />
              <path d="M12 16V8" />
            </svg>
          )}
          <span>{trend}%</span>
        </div>
        <span className="text-gray-400 text-[15px] font-medium">{trendText}</span>
      </div>
    </div>
  );
}
