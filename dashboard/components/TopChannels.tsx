import React from "react";

const channels = [
  {
    name: "Instagram",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#e1306c" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <rect x="2" y="2" width="20" height="20" rx="5" ry="5"></rect>
        <path d="M16 11.37A4 4 0 1 1 12.63 8 4 4 0 0 1 16 11.37z"></path>
        <line x1="17.5" y1="6.5" x2="17.51" y2="6.5"></line>
      </svg>
    ),
    bgColor: "bg-pink-50"
  },
  {
    name: "Facebook",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#1877F2" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M18 2h-3a5 5 0 0 0-5 5v3H7v4h3v8h4v-8h3l1-4h-4V7a1 1 0 0 1 1-1h3z"></path>
      </svg>
    ),
    bgColor: "bg-blue-50"
  },
  {
    name: "Google",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#DB4437" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2a10 10 0 0 0-7.07 17.07l1.41-1.41A8 8 0 1 1 12 4z"></path>
        <path d="M12 12m-3 0a3 3 0 1 0 6 0a3 3 0 1 0-6 0"></path>
      </svg>
    ),
    bgColor: "bg-red-50"
  },
  {
    name: "X (Twitter)",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#000000" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M4 4l11.733 16h4.267l-11.733-16z"></path>
        <path d="M4 20l6.768-6.768m2.46-2.46L20 4"></path>
      </svg>
    ),
    bgColor: "bg-gray-100"
  },
];

export function TopChannels() {
  // 32 bars to represent the distribution visually
  const totalBars = 32;
  const organicBars = Math.floor(totalBars * 0.58);

  return (
    <div className="flex flex-col h-full bg-[#F8FAFB] rounded-md border border-gray-200 shadow-sm p-4">
      <h3 className="text-2xl font-serif text-gray-900 mb-2 px-1">Channels</h3>

      {/* Channels List */}
      <div className="bg-card rounded-md p-1">
        <div className="flex flex-col gap-1">
          {channels.map((channel) => (
            <div key={channel.name} className="flex items-center gap-3 p-2 rounded-md hover:bg-gray-100 transition-colors cursor-pointer">
              <div className="flex gap-3 items-center">
                <div className={`w-9 h-9 rounded-lg flex items-center justify-center border border-gray-100 shadow-sm ${channel.bgColor}`}>
                  {channel.icon}
                </div>
                <span className="font-medium text-gray-800 text-[15px]">{channel.name}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
