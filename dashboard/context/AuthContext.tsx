"use client"

import { useRouter } from "next/navigation";
import { createContext, ReactNode, useContext, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

type AuthState = {
  accessToken: string | null;
  refreshToken: string | null;
};

type AuthContextValue = AuthState & {
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  refreshAccessToken: () => Promise<string>;
  logout: () => Promise<void>;
};

function getInitialAuthState(): AuthState {
  if (typeof window === "undefined") {
    return { accessToken: null, refreshToken: null };
  }

  return {
    accessToken: localStorage.getItem("access_token"),
    refreshToken: localStorage.getItem("refresh_token"),
  };
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children } : {children: ReactNode}) {
  const [auth, setAuth] = useState<AuthState>(getInitialAuthState);
  const [loading] = useState(false);
  const router = useRouter();
  
  async function login(email: string, password: string) {
    const res = await fetch(`${API}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      throw new Error("Login Failed");
    }
    
    const data = await res.json();
    localStorage.setItem("access_token", data.access_token);
    localStorage.setItem("refresh_token", data.refresh_token);
    
    setAuth({ accessToken: data.access_token, refreshToken: data.refresh_token });
  }
  
  async function register(name: string, email: string, password: string) {
    const res = await fetch(`${API}/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, email, password }),
    });
    
    if (!res.ok) {
      throw new Error("Register Failed");
    }
    
    await login(email, password);
  }
  
  
  async function refreshAccessToken() {
    if (!auth.refreshToken) throw new Error("No refresh Token");
    const res = await fetch(`${API}/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: auth.refreshToken }),
    });
    
    if (!res.ok) {
      throw new Error("Refresh Failed");
    }
    
    const data = await res.json();
    localStorage.setItem("access_token", data.access_token);
    setAuth((prev) => ({ ...prev, accessToken: data.access_token }));
    return data.access_token;
  }
  
  async function logout() {
    if (auth.refreshToken) {
      await fetch(`${API}/v1/auth/logout`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: auth.refreshToken }),
      });
    }

    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    setAuth({ accessToken: null, refreshToken: null });
    router.push("/");
  }
  
  return (
    <AuthContext.Provider value={{ ...auth, loading, login, register, logout, refreshAccessToken }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }

  return context;
}
