"use client"

import { useAuth } from "@/context/AuthContext";
import { useRouter } from "next/navigation";
import { ReactNode, useEffect } from "react";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { accessToken, loading } = useAuth();
  const router = useRouter();
  
  useEffect(() => {
    if (!loading && !accessToken) {
      router.replace("/");
    }
  }, [loading, accessToken, router]);
  
  if (loading) {
    return (
      <div>Checking Session....</div>
    );
  }
  
  if (!accessToken) {
    return null;
  }
  
  return <>{children}</>
}