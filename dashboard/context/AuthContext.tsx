"use client"

import { useRouter } from "next/navigation";
import { createContext, ReactNode, useContext, useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

type AuthState = {
  accessToken: string | null;
  refreshToken: string | null;
};

type UserProfile = {
  name: string | null;
  // profilePic: string | null;
}

type AuthContextValue = AuthState & {
  loading: boolean;
  user: UserProfile;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
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
  const [auth, setAuth] = useState<AuthState>({ accessToken: null, refreshToken: null });
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const [user, setUser] = useState<UserProfile>({ name: null});

  
  const fetchUserStatus = async (token: string): Promise<UserProfile> => {
      try {
        const res = await fetch(`${API}/v1/auth/me`, { 
          method: "GET",
          headers: {
            "Authorization": `Bearer ${token}`
          }
        });
        
        if (!res.ok) {
          return { name: null};
        }
        return await res.json();
      } catch (error) {
        console.error("Failed to fetch user data", error);
        return { name: null };
      }
    };
  
  useEffect(() => {
    // setAuth(getInitialAuthState());
    // setLoading(false);
    
    async function initializeAuth() {
      const initialState = getInitialAuthState();
      setAuth(initialState);
      
      if (initialState.accessToken) {
        const userData = await fetchUserStatus(initialState.accessToken);
        setUser(userData);
      }
      
      setLoading(false);
    }
    
    initializeAuth();
  }, []);
  
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
    setLoading(false);
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
    setUser({ name: null});
    setLoading(false);
    router.push("/");
  }
  
  return (
    <AuthContext.Provider value={{ ...auth, loading, login, register, logout, user }}>
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
