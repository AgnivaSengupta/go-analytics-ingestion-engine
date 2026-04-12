"use client";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

const ACCESS_TOKEN_KEY = "access_token";
const REFRESH_TOKEN_KEY = "refresh_token";

function isBrowser() {
  return typeof window !== "undefined";
}

function getAccessToken() {
  if (!isBrowser()) {
    return null;
  }

  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

function getRefreshToken() {
  if (!isBrowser()) {
    return null;
  }

  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

function setAccessToken(token: string) {
  if (!isBrowser()) {
    return;
  }

  localStorage.setItem(ACCESS_TOKEN_KEY, token);
}

function clearStoredTokens() {
  if (!isBrowser()) {
    return;
  }

  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

async function refreshAccessToken() {
  const refreshToken = getRefreshToken();
  if (!refreshToken) {
    clearStoredTokens();
    throw new Error("Missing refresh token");
  }

  const response = await fetch(`${API_BASE_URL}/v1/auth/refresh`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!response.ok) {
    clearStoredTokens();
    throw new Error("Unable to refresh access token");
  }

  const data = (await response.json()) as { access_token?: string };
  if (!data.access_token) {
    clearStoredTokens();
    throw new Error("Refresh response did not include an access token");
  }

  setAccessToken(data.access_token);
  return data.access_token;
}

function buildHeaders(headers?: HeadersInit, accessToken?: string | null) {
  const nextHeaders = new Headers(headers);

  if (accessToken && !nextHeaders.has("Authorization")) {
    nextHeaders.set("Authorization", `Bearer ${accessToken}`);
  }

  return nextHeaders;
}

type FetchWithAuthOptions = RequestInit & {
  skipAuth?: boolean;
  retryOnUnauthorized?: boolean;
};

export async function fetchWithAuth(
  input: string,
  options: FetchWithAuthOptions = {},
): Promise<Response> {
  const {
    headers,
    skipAuth = false,
    retryOnUnauthorized = true,
    ...rest
  } = options;

  const accessToken = skipAuth ? null : getAccessToken();
  const response = await fetch(`${API_BASE_URL}${input}`, {
    ...rest,
    headers: buildHeaders(headers, accessToken),
  });

  if (response.status !== 401 || skipAuth || !retryOnUnauthorized) {
    return response;
  }

  const refreshedToken = await refreshAccessToken();

  return fetch(`${API_BASE_URL}${input}`, {
    ...rest,
    headers: buildHeaders(headers, refreshedToken),
  });
}

export { API_BASE_URL };
