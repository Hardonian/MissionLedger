import "./globals.css";

export const metadata = {
  title: "MissionLedger Operator Console",
  description: "Read-only operator view for governed missions, approvals, proof events, and degraded states.",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
