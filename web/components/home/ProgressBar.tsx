"use client"

import { motion } from "framer-motion"
import { Progress } from "@/components/ui/progress"

interface ProgressBarProps {
    progress: number
    speed: string
    timeLeft: string
}

export function ProgressBar({ progress, speed, timeLeft }: ProgressBarProps) {
    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="w-full max-w-2xl mx-auto mt-8 bg-white rounded-2xl shadow-lg p-6 border border-gray-100"
        >
            <div className="flex justify-between items-center mb-4">
                <span className="font-medium text-gray-700">Downloading...</span>
                <span className="font-bold text-blue-600">{Math.round(progress)}%</span>
            </div>

            <div className="relative h-3 bg-gray-100 rounded-full overflow-hidden mb-4">
                <motion.div
                    className="absolute top-0 left-0 h-full bg-gradient-to-r from-blue-500 to-purple-500"
                    initial={{ width: 0 }}
                    animate={{ width: `${progress}%` }}
                    transition={{ type: "spring", stiffness: 50, damping: 20 }}
                />
                {/* Shimmer effect */}
                <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/30 to-transparent w-full -translate-x-full animate-[shimmer_2s_infinite]" />
            </div>

            <div className="flex justify-between text-sm text-gray-500">
                <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                    Speed: {speed}
                </div>
                <div>Time left: {timeLeft}</div>
            </div>
        </motion.div>
    )
}
