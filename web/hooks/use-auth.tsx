"use client"

import * as React from "react"
import { User, mockApi } from "@/lib/mock-api"

interface AuthContextType {
    user: User | null
    isLoading: boolean
    login: (email: string) => Promise<void>
    logout: () => void
    isAuthModalOpen: boolean
    openAuthModal: () => void
    closeAuthModal: () => void
}

const AuthContext = React.createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = React.useState<User | null>(null)
    const [isLoading, setIsLoading] = React.useState(false)
    const [isAuthModalOpen, setIsAuthModalOpen] = React.useState(false)

    // Load user from local storage on mount
    React.useEffect(() => {
        const storedUser = localStorage.getItem("v-asset-user")
        if (storedUser) {
            setUser(JSON.parse(storedUser))
        }
    }, [])

    const login = async (email: string) => {
        setIsLoading(true)
        try {
            const user = await mockApi.login(email)
            setUser(user)
            localStorage.setItem("v-asset-user", JSON.stringify(user))
            closeAuthModal()
        } catch (error) {
            console.error("Login failed", error)
        } finally {
            setIsLoading(false)
        }
    }

    const logout = () => {
        setUser(null)
        localStorage.removeItem("v-asset-user")
    }

    const openAuthModal = () => setIsAuthModalOpen(true)
    const closeAuthModal = () => setIsAuthModalOpen(false)

    return (
        <AuthContext.Provider
            value={{
                user,
                isLoading,
                login,
                logout,
                isAuthModalOpen,
                openAuthModal,
                closeAuthModal,
            }}
        >
            {children}
        </AuthContext.Provider>
    )
}

export function useAuth() {
    const context = React.useContext(AuthContext)
    if (context === undefined) {
        throw new Error("useAuth must be used within an AuthProvider")
    }
    return context
}
