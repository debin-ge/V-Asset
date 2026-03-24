"use client"

import Link from "next/link"
import { useAuth } from "@/hooks/use-auth"
import { Button } from "@/components/ui/button"
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { History, LogOut, Shield, User, Wallet } from "lucide-react"
import { formatCurrencyYuan } from "@/lib/format"

export function Header() {
    const { user, billingAccount, openAuthModal, logout } = useAuth()

    return (
        <header className="fixed top-0 left-0 right-0 z-50 flex h-16 items-center justify-between px-6 backdrop-blur-md bg-white/10 border-b border-white/10">
            <Link href="/" className="flex items-center gap-2">
                <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white font-bold">
                    Y
                </div>
                <span className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-purple-600">
                    YouDLP
                </span>
            </Link>

            <div className="flex items-center gap-4">
                {user ? (
                    <div className="flex items-center gap-4">
                        <div className="hidden md:flex items-center gap-2 px-3 py-1 rounded-full bg-black/5 border border-black/5">
                            <Wallet className="h-4 w-4 text-blue-600" />
                            <span className="text-sm font-medium">
                                {formatCurrencyYuan(billingAccount?.available_balance_yuan)}
                            </span>
                        </div>
                        <div className="hidden lg:flex items-center gap-2 px-3 py-1 rounded-full bg-black/5 border border-black/5">
                            <Shield className="h-4 w-4 text-slate-500" />
                            <span className="text-sm font-medium text-slate-700">
                                Reserved {formatCurrencyYuan(billingAccount?.reserved_balance_yuan)}
                            </span>
                        </div>
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <Button variant="ghost" className="relative h-10 w-10 rounded-full">
                                    <Avatar className="h-10 w-10 border-2 border-white shadow-sm">
                                        <AvatarImage src={user.avatar_url} alt={user.nickname} />
                                        <AvatarFallback>{user.nickname[0].toUpperCase()}</AvatarFallback>
                                    </Avatar>
                                </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent className="w-56" align="end" forceMount>
                                <DropdownMenuLabel className="font-normal">
                                    <div className="flex flex-col space-y-1">
                                        <p className="text-sm font-medium leading-none">{user.nickname}</p>
                                        <p className="text-xs leading-none text-muted-foreground">
                                            {user.email}
                                        </p>
                                    </div>
                                </DropdownMenuLabel>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem asChild>
                                    <Link href="/user?tab=profile" className="cursor-pointer">
                                        <User className="mr-2 h-4 w-4" />
                                        <span>Profile</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                    <Link href="/user?tab=history" className="cursor-pointer">
                                        <History className="mr-2 h-4 w-4" />
                                        <span>History</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                    <Link href="/user?tab=stats" className="cursor-pointer">
                                        <Wallet className="mr-2 h-4 w-4" />
                                        <span>Account</span>
                                    </Link>
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem onClick={logout} className="text-red-600 cursor-pointer">
                                    <LogOut className="mr-2 h-4 w-4" />
                                    <span>Log out</span>
                                </DropdownMenuItem>
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </div>
                ) : (
                    <Button onClick={openAuthModal} className="bg-blue-600 hover:bg-blue-700 text-white rounded-full px-6">
                        Login
                    </Button>
                )}
            </div>
        </header>
    )
}
