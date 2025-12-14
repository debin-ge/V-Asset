"use client"

import * as React from "react"
import { motion } from "framer-motion"
import { ArrowRight, Link2, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface InputProps {
    value: string
    onChange: (value: string) => void
    onParse: (value: string) => void
    isLoading: boolean
    disabled?: boolean
}

export function InputSection({ value, onChange, onParse, isLoading, disabled }: InputProps) {
    const [isFocused, setIsFocused] = React.useState(false)

    const handlePaste = async () => {
        try {
            const text = await navigator.clipboard.readText()
            onChange(text)
            onParse(text)
        } catch (err) {
            console.error("Failed to read clipboard", err)
        }
    }

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === "Enter" && value) {
            onParse(value)
        }
    }

    return (
        <div className="w-full max-w-2xl mx-auto relative z-10">
            <motion.div
                initial={{ scale: 0.95, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                transition={{ duration: 0.5 }}
                className={cn(
                    "relative flex items-center bg-white rounded-2xl shadow-2xl transition-all duration-300",
                    isFocused ? "ring-4 ring-blue-500/20 scale-[1.02]" : "hover:shadow-xl",
                    disabled && "opacity-50 pointer-events-none"
                )}
            >
                <div className="pl-6 text-gray-400">
                    <Link2 className="w-6 h-6" />
                </div>
                <input
                    type="text"
                    value={value}
                    onChange={(e) => onChange(e.target.value)}
                    onFocus={() => setIsFocused(true)}
                    onBlur={() => setIsFocused(false)}
                    onKeyDown={handleKeyDown}
                    placeholder="Paste video link here..."
                    className="flex-1 h-16 px-4 bg-transparent border-none outline-none text-lg text-gray-800 placeholder:text-gray-400"
                    disabled={disabled}
                />
                <div className="pr-2">
                    <Button
                        size="lg"
                        className={cn(
                            "rounded-xl h-12 px-6 transition-all duration-300",
                            value ? "bg-blue-600 hover:bg-blue-700 text-white" : "bg-gray-100 text-gray-400 hover:bg-gray-200"
                        )}
                        onClick={() => value ? onParse(value) : handlePaste()}
                        disabled={isLoading || disabled}
                    >
                        {isLoading ? (
                            <Loader2 className="w-5 h-5 animate-spin" />
                        ) : value ? (
                            <ArrowRight className="w-5 h-5" />
                        ) : (
                            <span className="font-medium">Paste</span>
                        )}
                    </Button>
                </div>
            </motion.div>

            {/* Decorative glow */}
            <div className={cn(
                "absolute inset-0 -z-10 bg-gradient-to-r from-blue-500 to-purple-500 rounded-2xl blur-xl transition-opacity duration-500",
                isFocused || value ? "opacity-30" : "opacity-0"
            )} />
        </div>
    )
}
