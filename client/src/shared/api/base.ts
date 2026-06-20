// Backend API base URL
// - If NEXT_PUBLIC_API_URL is set (production behind Nginx), use it
// - Local dev (NODE_ENV=development): frontend :3000 calls backend :8080 directly
// - Default: same origin (Nginx proxies /api/* and /:short to backend)
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || (process.env.NODE_ENV === "development" ? "http://localhost:8080" : "")

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message)
    this.name = "ApiError"
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new ApiError(body.error || `Request failed (${res.status})`, res.status)
  }
  return res.json()
}

export function apiGet<T>(path: string): Promise<T> {
  return request<T>(path)
}

export function apiPost<T>(path: string, body: unknown): Promise<T> {
  return request<T>(path, {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export { ApiError }
