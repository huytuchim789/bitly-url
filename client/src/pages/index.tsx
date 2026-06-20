import { UrlShortenerWidget } from "@/widgets/url-shortener"
import { UrlListWidget } from "@/widgets/url-list"

export default function Home() {
  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="container mx-auto px-4 py-6">
          <h1 className="text-3xl font-bold">Bitly URL Shortener</h1>
        </div>
      </header>

      <main className="container mx-auto px-4 py-8 space-y-8">
        <UrlShortenerWidget />
        <UrlListWidget />
      </main>

      <footer className="border-t mt-12">
        <div className="container mx-auto px-4 py-6 text-center text-muted-foreground text-sm">
          &copy; {new Date().getFullYear()} Bitly URL Shortener
        </div>
      </footer>
    </div>
  )
}
