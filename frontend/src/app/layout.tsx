import { Geist, Geist_Mono } from "next/font/google";
import { Metadata } from "next";
import "./globals.css";
import Providers from "@/app/providers";

const geistSans = Geist({
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Tagger",
  description: "Tag your audio files",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.className} ${geistMono.className} bg-gray-100 text-gray-1200 antialiased h-[100dvh] text-sm`}
      >
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
