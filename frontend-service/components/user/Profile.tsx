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
            toast.error("Nickname cannot be empty")
            return
        }
        setIsSavingProfile(true)
        try {
            const updatedUser = await authApi.updateProfile(nickname)
            setUser({ ...user!, nickname: updatedUser.nickname })
            toast.success("Profile updated successfully")
        } catch (error) {
            toast.error("Update failed, please try again")
        } finally {
            setIsSavingProfile(false)
        }
    }

    const handleChangePassword = async () => {
        if (!oldPassword || !newPassword) {
            toast.error("Please fill in all password fields")
            return
        }
        if (newPassword.length < 6) {
            toast.error("New password must be at least 6 characters")
            return
        }
        setIsSavingPassword(true)
        try {
            await authApi.changePassword(oldPassword, newPassword)
            setOldPassword("")
            setNewPassword("")
            toast.success("Password changed successfully")
        } catch (error) {
            const message = error instanceof Error ? error.message : "Password change failed"
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
                    <CardTitle>Profile Information</CardTitle>
                    <CardDescription>Update your account details and settings.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                    <div className="flex items-center gap-6">
                        <Avatar className="h-20 w-20">
                            <AvatarImage src={user.avatar_url} />
                            <AvatarFallback className="text-lg">{user.nickname[0].toUpperCase()}</AvatarFallback>
                        </Avatar>
                        <Button variant="outline" disabled>Change Avatar</Button>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-2">
                            <Label htmlFor="nickname">Nickname</Label>
                            <Input
                                id="nickname"
                                value={nickname}
                                onChange={(e) => setNickname(e.target.value)}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="email">Email</Label>
                            <Input id="email" defaultValue={user.email} disabled className="bg-gray-50" />
                        </div>
                    </div>

                    <Button onClick={handleSaveProfile} disabled={isSavingProfile}>
                        {isSavingProfile && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                        Save Changes
                    </Button>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Security Settings</CardTitle>
                    <CardDescription>Manage your password and account security.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="current-password">Current Password</Label>
                        <Input
                            id="current-password"
                            type="password"
                            value={oldPassword}
                            onChange={(e) => setOldPassword(e.target.value)}
                        />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="new-password">New Password</Label>
                        <Input
                            id="new-password"
                            type="password"
                            value={newPassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                            placeholder="At least 6 characters"
                        />
                    </div>
                    <Button variant="outline" onClick={handleChangePassword} disabled={isSavingPassword}>
                        {isSavingPassword && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                        Change Password
                    </Button>
                </CardContent>
            </Card>
        </div>
    )
}

