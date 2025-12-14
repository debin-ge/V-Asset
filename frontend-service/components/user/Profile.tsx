"use client"

import * as React from "react"
import { useAuth } from "@/hooks/use-auth"
import { authApi } from "@/lib/api/auth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Loader2 } from "lucide-react"
import { toast } from "sonner"

export function Profile() {
    const { user, setUser } = useAuth()
    const [nickname, setNickname] = React.useState("")
    const [oldPassword, setOldPassword] = React.useState("")
    const [newPassword, setNewPassword] = React.useState("")
    const [isSavingProfile, setIsSavingProfile] = React.useState(false)
    const [isSavingPassword, setIsSavingPassword] = React.useState(false)

    React.useEffect(() => {
        if (user) {
            setNickname(user.nickname)
        }
    }, [user])

    const handleSaveProfile = async () => {
        if (!nickname.trim()) {
            toast.error("昵称不能为空")
            return
        }
        setIsSavingProfile(true)
        try {
            const updatedUser = await authApi.updateProfile(nickname)
            setUser({ ...user!, nickname: updatedUser.nickname })
            toast.success("个人信息更新成功")
        } catch (error) {
            toast.error("更新失败，请重试")
        } finally {
            setIsSavingProfile(false)
        }
    }

    const handleChangePassword = async () => {
        if (!oldPassword || !newPassword) {
            toast.error("请填写完整的密码信息")
            return
        }
        if (newPassword.length < 6) {
            toast.error("新密码长度至少6位")
            return
        }
        setIsSavingPassword(true)
        try {
            await authApi.changePassword(oldPassword, newPassword)
            setOldPassword("")
            setNewPassword("")
            toast.success("密码修改成功")
        } catch (error) {
            const message = error instanceof Error ? error.message : "密码修改失败"
            toast.error(message)
        } finally {
            setIsSavingPassword(false)
        }
    }

    if (!user) return null

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>个人信息</CardTitle>
                    <CardDescription>更新您的账户资料和设置。</CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                    <div className="flex items-center gap-6">
                        <Avatar className="h-20 w-20">
                            <AvatarImage src={user.avatar_url} />
                            <AvatarFallback className="text-lg">{user.nickname[0].toUpperCase()}</AvatarFallback>
                        </Avatar>
                        <Button variant="outline" disabled>更换头像</Button>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-2">
                            <Label htmlFor="nickname">昵称</Label>
                            <Input
                                id="nickname"
                                value={nickname}
                                onChange={(e) => setNickname(e.target.value)}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="email">邮箱</Label>
                            <Input id="email" defaultValue={user.email} disabled className="bg-gray-50" />
                        </div>
                    </div>

                    <Button onClick={handleSaveProfile} disabled={isSavingProfile}>
                        {isSavingProfile && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                        保存更改
                    </Button>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>安全设置</CardTitle>
                    <CardDescription>管理您的密码和账户安全。</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="current-password">当前密码</Label>
                        <Input
                            id="current-password"
                            type="password"
                            value={oldPassword}
                            onChange={(e) => setOldPassword(e.target.value)}
                        />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="new-password">新密码</Label>
                        <Input
                            id="new-password"
                            type="password"
                            value={newPassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                            placeholder="至少6位字符"
                        />
                    </div>
                    <Button variant="outline" onClick={handleChangePassword} disabled={isSavingPassword}>
                        {isSavingPassword && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                        修改密码
                    </Button>
                </CardContent>
            </Card>
        </div>
    )
}

