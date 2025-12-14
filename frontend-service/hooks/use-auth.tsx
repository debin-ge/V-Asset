"use client"

import * as React from "react"
import { authApi, User } from "@/lib/api/auth"
import { historyApi } from "@/lib/api/history"
import { tokenManager } from "@/lib/api-client"

interface AuthContextType {
    user: User | null
    setUser: (user: User | null) => void
    quota: number
    isLoading: boolean
    isAuthenticated: boolean
    login: (email: string, password: string) => Promise<void>
    register: (email: string, password: string, nickname: string) => Promise<void>
    logout: () => Promise<void>
    refreshUser: () => Promise<void>
    isAuthModalOpen: boolean
    openAuthModal: () => void
    closeAuthModal: () => void
}

const AuthContext = React.createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = React.useState<User | null>(null)
    const [quota, setQuota] = React.useState(0)
    const [isLoading, setIsLoading] = React.useState(false)
    const [isAuthModalOpen, setIsAuthModalOpen] = React.useState(false)

    const isAuthenticated = !!user && tokenManager.isAuthenticated()

    // 初始化时检查登录状态
    React.useEffect(() => {
        const initAuth = async () => {
            if (tokenManager.isAuthenticated()) {
                try {
                    const profile = await authApi.getProfile()
                    setUser(profile)
                    const quotaData = await historyApi.getQuota()
                    setQuota(quotaData.remaining)
                } catch (error) {
                    console.error("Failed to restore session:", error)
                    tokenManager.clearTokens()
                }
            }
        }
        initAuth()

        // 监听登出事件
        const handleLogout = () => {
            setUser(null)
            setQuota(0)
        }
        window.addEventListener('auth:logout', handleLogout)
        return () => window.removeEventListener('auth:logout', handleLogout)
    }, [])

    const login = async (email: string, password: string) => {
        setIsLoading(true)
        try {
            const response = await authApi.login(email, password)
            setUser(response.user)
            // 获取配额
            const quotaData = await historyApi.getQuota()
            setQuota(quotaData.remaining)
            closeAuthModal()
        } finally {
            setIsLoading(false)
        }
    }

    const register = async (email: string, password: string, nickname: string) => {
        setIsLoading(true)
        try {
            await authApi.register(email, password, nickname)
            // 注册成功后自动登录
            await login(email, password)
        } finally {
            setIsLoading(false)
        }
    }

    const logout = async () => {
        setIsLoading(true)
        try {
            await authApi.logout()
        } finally {
            setUser(null)
            setQuota(0)
            setIsLoading(false)
        }
    }

    const refreshUser = async () => {
        if (tokenManager.isAuthenticated()) {
            const profile = await authApi.getProfile()
            setUser(profile)
            const quotaData = await historyApi.getQuota()
            setQuota(quotaData.remaining)
        }
    }

    const openAuthModal = () => setIsAuthModalOpen(true)
    const closeAuthModal = () => setIsAuthModalOpen(false)

    return (
        <AuthContext.Provider
            value={{
                user,
                setUser,
                quota,
                isLoading,
                isAuthenticated,
                login,
                register,
                logout,
                refreshUser,
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
