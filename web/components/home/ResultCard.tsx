"use client"

import { motion } from "framer-motion"
import { Download, Music, Video } from "lucide-react"
import { Button } from "@/components/ui/button"
import { VideoInfo } from "@/lib/mock-api"
import { useAuth } from "@/hooks/use-auth"

interface ResultCardProps {
    info: VideoInfo
    onDownload: (type: 'video' | 'audio') => void
}

export function ResultCard({ info, onDownload }: ResultCardProps) {
    const { user, openAuthModal } = useAuth()

    const handleDownload = (type: 'video' | 'audio') => {
        if (!user) {
            openAuthModal()
            return
        }
        onDownload(type)
    }

    return (
        <motion.div
            initial={{ y: 20, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            className="w-full max-w-2xl mx-auto mt-8 bg-white rounded-2xl shadow-xl overflow-hidden border border-gray-100"
        >
            <div className="flex flex-col md:flex-row">
                <div className="w-full md:w-48 h-32 md:h-auto relative">
                    <img
                        src={info.thumbnail}
                        alt={info.title}
                        className="w-full h-full object-cover"
                    />
                    <div className="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-1 rounded">
                        {info.duration}
                    </div>
                </div>
                <div className="p-6 flex-1 flex flex-col justify-between">
                    <div>
                        <div className="flex items-center gap-2 mb-2">
                            <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">
                                {info.platform}
                            </span>
                        </div>
                        <h3 className="font-semibold text-lg leading-tight line-clamp-2 mb-4">
                            {info.title}
                        </h3>
                    </div>
                    <div className="flex gap-3">
                        <Button
                            onClick={() => handleDownload('video')}
                            className="flex-1 bg-gradient-to-r from-blue-600 to-blue-500 hover:from-blue-700 hover:to-blue-600 text-white shadow-lg shadow-blue-500/20"
                        >
                            <Video className="w-4 h-4 mr-2" />
                            Download Video
                        </Button>
                        <Button
                            variant="outline"
                            onClick={() => handleDownload('audio')}
                            className="flex-1 border-gray-200 hover:bg-gray-50 hover:text-gray-900"
                        >
                            <Music className="w-4 h-4 mr-2" />
                            Audio Only
                        </Button>
                    </div>
                </div>
            </div>
        </motion.div>
    )
}
