import type { GetServerSideProps } from "next"

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export const getServerSideProps: GetServerSideProps = async ({ params }) => {
  const short = params?.short as string
  if (!short) return { notFound: true }

  try {
    const res = await fetch(`${API_BASE}/api/urls/${short}`)

    if (res.ok) {
      const data = await res.json() as { original: string }
      return { redirect: { destination: data.original, permanent: false } }
    }
  } catch {
    // fetch failed, fall through to notFound
  }

  return { notFound: true }
}

export default function RedirectPage() {
  return null
}
