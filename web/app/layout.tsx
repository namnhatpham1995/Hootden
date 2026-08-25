import type { Metadata } from "next";
import { Fredoka, Nunito_Sans, Lora } from "next/font/google";
import "./globals.css";

const fredoka = Fredoka({
  variable: "--font-fredoka",
  subsets: ["latin"],
});

const nunitoSans = Nunito_Sans({
  variable: "--font-nunito-sans",
  subsets: ["latin"],
});

const lora = Lora({
  variable: "--font-lora",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Hootden",
  description: "A small den for your notes.",
};

// Applies a stored theme choice before first paint -- inline and
// synchronous so there's no flash of the wrong theme while React hydrates.
// Static string, no user input: safe as dangerouslySetInnerHTML.
const themeInitScript = `(function () {
  try {
    var stored = localStorage.getItem("theme");
    if (stored === "light" || stored === "dark") {
      document.documentElement.dataset.theme = stored;
    }
  } catch (e) {}
})();`;

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${fredoka.variable} ${nunitoSans.variable} ${lora.variable}`}
      // themeInitScript sets data-theme on this element before hydration,
      // on purpose -- that's what prevents a flash of the wrong theme, and
      // it's the one attribute React should not try to reconcile away.
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body>{children}</body>
    </html>
  );
}
