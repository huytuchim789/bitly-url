import { useState } from "react"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"

interface UrlFormProps {
  onSubmit: (url: string) => void
  isLoading: boolean
}

export function UrlForm({ onSubmit, isLoading }: UrlFormProps) {
  const [value, setValue] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!value) return
    onSubmit(value)
    setValue("")
  }

  return (
    <form onSubmit={handleSubmit} className="flex gap-2">
      <Input
        type="url"
        placeholder="https://example.com/very-long-url"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        required
        disabled={isLoading}
      />
      <Button type="submit" disabled={isLoading}>
        {isLoading ? "Shortening..." : "Shorten"}
      </Button>
    </form>
  )
}
