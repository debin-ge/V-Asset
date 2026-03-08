"use client"

import * as React from "react"
import Image from "next/image"
import { getThumbnailCandidates } from "@/lib/thumbnail"
import { cn } from "@/lib/utils"

interface RemoteThumbnailProps {
  src?: string | null
  alt: string
  className?: string
  sizes?: string
  fallbackClassName?: string
  fallbackText?: string
}

export function RemoteThumbnail({
  src,
  alt,
  className,
  sizes = "100vw",
  fallbackClassName,
  fallbackText = "No cover",
}: RemoteThumbnailProps) {
  const candidates = React.useMemo(() => getThumbnailCandidates(src), [src])
  const [candidateIndex, setCandidateIndex] = React.useState(0)

  React.useEffect(() => {
    setCandidateIndex(0)
  }, [src])

  const currentSrc = candidates[candidateIndex]

  if (!currentSrc) {
    return (
      <div
        className={cn(
          "flex h-full w-full items-center justify-center bg-gray-100 text-gray-400",
          fallbackClassName
        )}
      >
        {fallbackText}
      </div>
    )
  }

  return (
    <Image
      key={currentSrc}
      src={currentSrc}
      alt={alt}
      fill
      sizes={sizes}
      className={cn("object-cover", className)}
      onError={() => {
        setCandidateIndex((current) =>
          current < candidates.length - 1 ? current + 1 : current
        )
      }}
    />
  )
}
