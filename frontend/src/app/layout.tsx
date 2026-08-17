import type { Metadata } from "next";
import { Fraunces, Source_Sans_3 } from "next/font/google";
import { CountryProvider } from "@/lib/country";
import { AuthProvider } from "@/lib/auth";
import { SiteHeader } from "@/components/site-header";
import "./globals.css";

const display = Fraunces({
  variable: "--font-display",
  subsets: ["latin"],
  weight: ["500", "600", "700"],
});

const body = Source_Sans_3({
  variable: "--font-body",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "HomeGauge — Mortgage clarity",
  description:
    "Understand mortgage options in your market, check salary-account eligibility, estimate affordability, and get guided help. HomeGauge is not a bank.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${display.variable} ${body.variable} antialiased`}>
        <AuthProvider>
          <CountryProvider>
            <SiteHeader />
            {children}
          </CountryProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
