import { apiGet, apiPost } from "@/shared/api/base"
import type { ShortenedUrl } from "./types"

export function getUrls(): Promise<ShortenedUrl[]> {
  return apiGet("/api/urls")
}

export function shortenUrl(url: string): Promise<ShortenedUrl> {
  return apiPost("/api/shorten", { url })
}
