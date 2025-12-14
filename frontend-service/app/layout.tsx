import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/hooks/use-auth";
import { Header } from "@/components/common/Header";
import { Footer } from "@/components/common/Footer";
import { AuthModal } from "@/components/auth/AuthModal";
import { Toaster } from "@/components/ui/sonner";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "V-Asset - Video Asset Platform",
  description: "Download videos from various platforms with ease.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased min-h-screen flex flex-col relative overflow-x-hidden`}
        suppressHydrationWarning
      >
        <div className="fixed inset-0 -z-10 bg-white">
          <div className="absolute top-[-20%] left-[-10%] w-[50%] h-[50%] rounded-full bg-purple-200/40 blur-[120px] animate-pulse" />
          <div className="absolute top-[10%] right-[-10%] w-[40%] h-[40%] rounded-full bg-blue-200/40 blur-[100px] animate-pulse delay-1000" />
          <div className="absolute bottom-[-10%] left-[20%] w-[60%] h-[60%] rounded-full bg-pink-200/30 blur-[140px] animate-pulse delay-2000" />
        </div>

        <AuthProvider>
          <Header />
          <main className="flex-1 flex flex-col pt-16">
            {children}
          </main>
          <Footer />
          <AuthModal />
          <Toaster />
        </AuthProvider>
      </body>
    </html>
  );
}
