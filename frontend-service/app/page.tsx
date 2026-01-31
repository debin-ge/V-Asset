"use client"

import { useDownload } from "@/hooks/use-download"
import { InputSection } from "@/components/home/Input"
import { ResultCard } from "@/components/home/ResultCard"

export default function Home() {
  const {
    url,
    setUrl,
    status,
    videoInfo,
    handleParse,
    startDownload,
    reset,
  } = useDownload()

  return (
    <div className="flex flex-col items-center justify-center min-h-[calc(100vh-140px)] px-4 py-12">
      <div className="text-center mb-12 space-y-4">
        <h1 className="text-5xl md:text-7xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 pb-2">
          Unlock Video Power
        </h1>
        <p className="text-xl text-gray-600 max-w-2xl mx-auto">
          Download high-quality videos from YouTube, Bilibili, TikTok, and more.
          Simple, fast, and free.
        </p>
      </div>

      <InputSection
        value={url}
        onChange={setUrl}
        onParse={handleParse}
        isLoading={status === "parsing"}
        disabled={status === "downloading"}
      />

      {(status === "parsed" || status === "downloading" || status === "completed") && videoInfo && (
        <ResultCard info={videoInfo} onDownload={startDownload} />
      )}
    </div>
  )
}
