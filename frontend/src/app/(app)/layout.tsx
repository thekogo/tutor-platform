import { AppHeader } from "@/components/AppHeader";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:rounded focus:bg-white focus:p-2">
        ข้ามไปยังเนื้อหาหลัก
      </a>
      <AppHeader />
      <main id="main" className="mx-auto w-full max-w-7xl flex-1 px-4 py-6">
        {children}
      </main>
    </>
  );
}
