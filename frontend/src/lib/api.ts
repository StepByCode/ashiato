function resolveApiBaseUrl(): string {
  const configured = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  if (configured) return configured;

  if (typeof window !== "undefined") {
    const { hostname } = window.location;
    if (hostname === "backatage.stepbycode.work") {
      return "https://api.stepbycode.work";
    }
  }

  return "";
}

export async function apiFetch(
  path: string,
  token: string | null,
  init?: RequestInit
): Promise<Response> {
  const apiBaseUrl = resolveApiBaseUrl();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  return fetch(`${apiBaseUrl}${path}`, { ...init, headers });
}
