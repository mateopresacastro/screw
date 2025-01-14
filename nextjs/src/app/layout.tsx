import { Geist, Geist_Mono } from "next/font/google";
import { Metadata } from "next";
import Providers from "@/components/providers";
import Header from "@/components/header";
import "./globals.css";

const geistSans = Geist({
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "screw",
  description: "slowed + reverb",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${geistSans.className} ${geistMono.className} bg-gray-200 text-gray-1200 mx-auto antialiased h-[100dvh] text-xs p-4 max-w-96 leading-6`}
        suppressHydrationWarning
      >
        <div>
          <Providers>
            <Header />
            {children}
          </Providers>
        </div>
      </body>
    </html>
  );
}
