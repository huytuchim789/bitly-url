import type { ShortenedUrl } from "@/entities/url"
import { Card, CardContent } from "@/shared/ui/card"
import { Skeleton } from "@/shared/ui/skeleton"

interface UrlListProps {
  urls: ShortenedUrl[]
  isLoading: boolean
}

export function UrlList({ urls, isLoading }: UrlListProps) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full rounded-lg" />
        ))}
      </div>
    )
  }

  if (urls.length === 0) {
    return <p className="text-muted-foreground text-sm">No URLs yet.</p>
  }

  return (
    <div className="space-y-2">
      {urls.map((url) => (
        <Card key={url.id}>
          <CardContent className="flex items-center gap-2 p-3">
            <span className="flex-1 truncate text-sm">{url.original}</span>
            <span className="text-muted-foreground">→</span>
            <a
              href={`/${url.short}`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:underline font-mono text-sm shrink-0"
            >
              /{url.short}
            </a>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
