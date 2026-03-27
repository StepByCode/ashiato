import type { Metadata } from "next";
import type { ReactNode } from "react";
import { AuthProvider } from "@/lib/auth-context";
import { PeriodProvider } from "@/lib/period-context";
import "./globals.css";

export const metadata: Metadata = {
  title: "Ashiato Frontend",
  description: "Ashiato task creation and confirmation pages",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ja">
      <body>
        <AuthProvider>
          <PeriodProvider>{children}</PeriodProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
