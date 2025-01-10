"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

export default function RouteSync({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (window.location.pathname !== pathname) {
      router.replace("/", { scroll: false });
    }
  }, [pathname, router]);

  return children;
}
