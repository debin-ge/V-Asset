"use client"

import * as React from "react"
import { useAuth } from "@/hooks/use-auth"
import { Button } from "@/components/ui/button"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Github } from "lucide-react"
import { toast } from "sonner"

export function AuthModal() {
    const { isAuthModalOpen, closeAuthModal, login, register, isLoading } = useAuth()
    const [email, setEmail] = React.useState("")
    const [password, setPassword] = React.useState("")
    const [nickname, setNickname] = React.useState("")
    const [confirmPassword, setConfirmPassword] = React.useState("")
    const [error, setError] = React.useState("")

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault()
        setError("")
        if (!email || !password) {
            setError("请填写邮箱和密码")
            return
        }
        try {
            await login(email, password)
            toast.success("登录成功！")
        } catch (err) {
            const message = err instanceof Error ? err.message : "登录失败"
            setError(message)
            toast.error(message)
        }
    }

    const handleRegister = async (e: React.FormEvent) => {
        e.preventDefault()
        setError("")
        if (!email || !password || !nickname) {
            setError("请填写所有必填字段")
            return
        }
        if (password !== confirmPassword) {
            setError("两次输入的密码不一致")
            return
        }
        if (password.length < 6) {
            setError("密码长度至少6位")
            return
        }
        try {
            await register(email, password, nickname)
            toast.success("注册成功！")
        } catch (err) {
            const message = err instanceof Error ? err.message : "注册失败"
            setError(message)
            toast.error(message)
        }
    }

    const resetForm = () => {
        setEmail("")
        setPassword("")
        setNickname("")
        setConfirmPassword("")
        setError("")
    }

    return (
        <Dialog open={isAuthModalOpen} onOpenChange={(open) => { if (!open) { resetForm(); closeAuthModal(); } }}>
            <DialogContent className="sm:max-w-[425px] bg-[#1A1A1A] text-white border-gray-800">
                <DialogHeader>
                    <DialogTitle className="text-center text-2xl font-bold">V-Asset</DialogTitle>
                    <DialogDescription className="text-center text-gray-400">
                        登录以访问您的下载和历史记录
                    </DialogDescription>
                </DialogHeader>
                <Tabs defaultValue="login" className="w-full" onValueChange={() => setError("")}>
                    <TabsList className="grid w-full grid-cols-2 bg-gray-800">
                        <TabsTrigger value="login">登录</TabsTrigger>
                        <TabsTrigger value="register">注册</TabsTrigger>
                    </TabsList>
                    <TabsContent value="login">
                        <form onSubmit={handleLogin} className="space-y-4 py-4">
                            {error && (
                                <div className="text-red-500 text-sm bg-red-500/10 p-2 rounded">
                                    {error}
                                </div>
                            )}
                            <div className="space-y-2">
                                <Label htmlFor="email">邮箱</Label>
                                <Input
                                    id="email"
                                    type="email"
                                    placeholder="m@example.com"
                                    className="bg-gray-900 border-gray-700 text-white"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    required
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor="password">密码</Label>
                                <Input
                                    id="password"
                                    type="password"
                                    className="bg-gray-900 border-gray-700 text-white"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    required
                                />
                            </div>
                            <Button type="submit" className="w-full bg-blue-600 hover:bg-blue-700" disabled={isLoading}>
                                {isLoading ? "登录中..." : "登录"}
                            </Button>
                        </form>
                    </TabsContent>
                    <TabsContent value="register">
                        <form onSubmit={handleRegister} className="space-y-4 py-4">
                            {error && (
                                <div className="text-red-500 text-sm bg-red-500/10 p-2 rounded">
                                    {error}
                                </div>
                            )}
                            <div className="space-y-2">
                                <Label htmlFor="reg-nickname">昵称</Label>
                                <Input
                                    id="reg-nickname"
                                    type="text"
                                    placeholder="您的昵称"
                                    className="bg-gray-900 border-gray-700 text-white"
                                    value={nickname}
                                    onChange={(e) => setNickname(e.target.value)}
                                    required
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor="reg-email">邮箱</Label>
                                <Input
                                    id="reg-email"
                                    type="email"
                                    placeholder="m@example.com"
                                    className="bg-gray-900 border-gray-700 text-white"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    required
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor="reg-password">密码</Label>
                                <Input
                                    id="reg-password"
                                    type="password"
                                    placeholder="至少6位"
                                    className="bg-gray-900 border-gray-700 text-white"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    required
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor="reg-confirm-password">确认密码</Label>
                                <Input
                                    id="reg-confirm-password"
                                    type="password"
                                    className="bg-gray-900 border-gray-700 text-white"
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    required
                                />
                            </div>
                            <Button type="submit" className="w-full bg-blue-600 hover:bg-blue-700" disabled={isLoading}>
                                {isLoading ? "注册中..." : "创建账号"}
                            </Button>
                        </form>
                    </TabsContent>
                </Tabs>
                <div className="relative">
                    <div className="absolute inset-0 flex items-center">
                        <span className="w-full border-t border-gray-700" />
                    </div>
                    <div className="relative flex justify-center text-xs uppercase">
                        <span className="bg-[#1A1A1A] px-2 text-gray-400">或通过以下方式</span>
                    </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                    <Button variant="outline" className="bg-gray-900 border-gray-700 hover:bg-gray-800 hover:text-white text-gray-300">
                        <svg className="mr-2 h-4 w-4" viewBox="0 0 24 24">
                            <path
                                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                                fill="#4285F4"
                            />
                            <path
                                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                                fill="#34A853"
                            />
                            <path
                                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                                fill="#FBBC05"
                            />
                            <path
                                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                                fill="#EA4335"
                            />
                        </svg>
                        Google
                    </Button>
                    <Button variant="outline" className="bg-gray-900 border-gray-700 hover:bg-gray-800 hover:text-white text-gray-300">
                        <Github className="mr-2 h-4 w-4" />
                        GitHub
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    )
}
