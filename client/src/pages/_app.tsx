import type { AppProps } from "next/app"
import { Providers } from "@/app/providers"
import "@/app/styles/globals.css"

export default function App({ Component, pageProps }: AppProps) {
  return (
    <Providers>
      <Component {...pageProps} />
    </Providers>
  )
}
