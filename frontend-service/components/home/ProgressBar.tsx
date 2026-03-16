"use client"

import { motion } from "framer-motion"

interface ProgressBarProps {
    progress: number
    speed: string
    timeLeft: string
    phase?: string
    phaseLabel?: string
}

export function ProgressBar({ progress, speed, timeLeft, phase, phaseLabel }: ProgressBarProps) {
    const isDownloading = !phase || phase.startsWith("downloading")
    const isIndeterminate = phase === "merging" || phase === "processing" || phase === "transferring"

    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="w-full max-w-2xl mx-auto mt-8 bg-white rounded-2xl shadow-lg p-6 border border-gray-100"
        >
            <div className="flex justify-between items-center mb-4">
                <span className="px-3 py-1 bg-blue-50 text-blue-700 rounded-full text-sm font-medium border border-blue-200 shadow-sm flex items-center gap-2">
                    {isDownloading ? (
                        <span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
                    ) : (
                        <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                    )}
                    {phaseLabel || "Downloading..."}
                </span>
                <span className="font-bold text-blue-600 text-lg">{Math.round(progress)}%</span>
            </div>

            <div className="relative h-3 bg-gray-100 rounded-full overflow-hidden mb-4">
                <motion.div
                    className={`absolute top-0 left-0 h-full ${isIndeterminate
                        ? "bg-gradient-to-r from-purple-400 via-blue-500 to-purple-400"
                        : "bg-gradient-to-r from-blue-500 to-purple-500"
                        }`}
                    initial={{ width: 0 }}
                    animate={{ width: `${progress}%` }}
                    transition={{ type: "spring", stiffness: 50, damping: 20 }}
                />
                {/* Shimmer / pulse effect */}
                {isIndeterminate ? (
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/40 to-transparent animate-[shimmer_1.2s_infinite]" />
                ) : (
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent w-full -translate-x-full animate-[shimmer_2s_infinite]" />
                )}
            </div>

            <div className="flex justify-between text-sm text-gray-500">
                {isDownloading ? (
                    <>
                        <div className="flex items-center gap-2">
                            <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                            Speed: {speed}
                        </div>
                        <div>Remaining time: {timeLeft || "--"}</div>
                    </>
                ) : (
                    <div className="flex items-center gap-2 text-blue-600 font-medium">
                        {phase === "merging" && "Merging video and audio, please wait..."}
                        {phase === "processing" && "Processing the final details, almost there..."}
                        {phase === "transferring" && "Browser download is being handed off. You should see it in your browser's download manager."}
                    </div>
                )}
            </div>
        </motion.div>
    )
}
