import Link from "next/link";
import Header from "./Header";
import { ReactNode } from "react";
export default function DashboardLayout({ children }: { children: ReactNode }) {
  return (
    <div className="h-screen flex flex-col items-center w-full">
      <Header />

      <div className="flex-1 w-full overflow-hidden flex justify-center">
        {children}
      </div>
    </div>
  );
}
