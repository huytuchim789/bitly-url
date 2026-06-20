import { useShortenUrl } from "@/entities/url"
import { UrlForm } from "@/features/shorten-url"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card"
import { CardFooter } from "@/shared/ui/card"

export function UrlShortenerWidget() {
  const { mutate, isPending, isError, data } = useShortenUrl()

  return (
    <Card>
      <CardHeader>
        <CardTitle>Shorten a URL</CardTitle>
        <CardDescription>Enter a long URL to get a short link</CardDescription>
      </CardHeader>
      <CardContent>
        <UrlForm onSubmit={(url) => mutate(url)} isLoading={isPending} />
        {isError && (
          <p className="text-destructive text-sm mt-2">Failed to shorten URL</p>
        )}
      </CardContent>
      {data && (
        <CardFooter className="flex gap-2 text-sm border-t pt-4">
          <span className="text-muted-foreground">Short URL:</span>
          <a
            href={`/${data.short}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline font-mono"
          >
            /{data.short}
          </a>
        </CardFooter>
      )}
    </Card>
  )
}
