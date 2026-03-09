import "./globals.css";

import { Toaster } from "@/components/ui/sonner";
import { AuthProvider } from "@/hooks/use-auth";

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN" className="font-sans">
      <body>
        <AuthProvider>
          {children}
          <Toaster richColors position="top-right" />
        </AuthProvider>
      </body>
    </html>
  );
}
