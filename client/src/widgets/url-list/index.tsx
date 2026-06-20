import { useUrls } from "@/entities/url"
import { UrlList } from "@/features/view-urls"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card"

export function UrlListWidget() {
  const { data: urls = [], isLoading } = useUrls()
 console.log(urls)
  return (
    <Card>
      <CardHeader>
        <CardTitle>Your URLs</CardTitle>
        <CardDescription>All your shortened URLs</CardDescription>
      </CardHeader>
      <CardContent>
        <UrlList urls={urls} isLoading={isLoading} />
      </CardContent>
    </Card>
  )
}
