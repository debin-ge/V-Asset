"use client"

import { useDownload } from "@/hooks/use-download"
import { InputSection } from "@/components/home/Input"
import { ResultCard } from "@/components/home/ResultCard"
import { ProgressBar } from "@/components/home/ProgressBar"

export default function Home() {
  const {
    url,
    setUrl,
    status,
    videoInfo,
    progress,
    speed,
    timeLeft,
    handleParse,
    startDownload,
    downloadFile,
    reset,
    autoDownloadAttempted,
    phase,
    phaseLabel,
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

      {status === "parsed" && videoInfo && (
        <ResultCard info={videoInfo} onDownload={startDownload} />
      )}

      {(status === "downloading" || status === "completed") && (
        <ProgressBar progress={progress} speed={speed} timeLeft={timeLeft} phase={phase} phaseLabel={phaseLabel} />
      )}

      {status === "completed" && (
        <div className="mt-8 text-center animate-fade-in">
          <p className="text-green-600 font-medium mb-4">Ready and downloading to your local device!</p>
          <p className="text-sm text-gray-500 mb-4">
            Please check your browser&apos;s download manager. If it doesn&apos;t start automatically, please click the button below to download manually.
          </p>
          <div className="flex items-center justify-center gap-4">
            <button
              onClick={downloadFile}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
            >
              Download manually
            </button>
            <button
              onClick={reset}
              className="text-blue-600 hover:underline"
            >
              Download another video
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
