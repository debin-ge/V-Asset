const YOUTUBE_THUMBNAIL_NAMES = [
  "maxresdefault.jpg",
  "sddefault.jpg",
  "hqdefault.jpg",
  "mqdefault.jpg",
  "default.jpg",
] as const

export function getThumbnailCandidates(src?: string | null): string[] {
  if (!src) return []

  const candidates = new Set<string>([src])

  try {
    const url = new URL(src)
    const hostname = url.hostname.toLowerCase()

    if (hostname.endsWith("ytimg.com")) {
      const pathname = url.pathname
      const matchedName = YOUTUBE_THUMBNAIL_NAMES.find((name) =>
        pathname.endsWith(`/${name}`)
      )

      if (matchedName) {
        YOUTUBE_THUMBNAIL_NAMES.forEach((name) => {
          candidates.add(src.replace(matchedName, name))
        })
      }
    }
  } catch {
    return [src]
  }

  return Array.from(candidates)
}
