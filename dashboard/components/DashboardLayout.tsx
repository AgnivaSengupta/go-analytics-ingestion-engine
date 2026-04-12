import Link from 'next/link';
import Header from './Header';
import { ReactNode } from 'react';
export default function DashboardLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col items-center w-full">
      <Header />
      
      <main>
        {children}
      </main>
    </div>
  )
}
