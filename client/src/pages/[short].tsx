import type { GetServerSideProps } from "next"

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export const getServerSideProps: GetServerSideProps = async ({ params }) => {
  const short = params?.short as string
  if (!short) return { notFound: true }

  try {
    const res = await fetch(`${API_BASE}/${short}`, { redirect: "manual" })

    if (res.status === 302) {
      const location = res.headers.get("location")
      if (location) {
        return { redirect: { destination: location, permanent: false } }
      }
    }
  } catch {
    // fetch failed, fall through to notFound
  }

  return { notFound: true }
}

export default function RedirectPage() {
  return null
}
