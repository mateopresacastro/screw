// components/auth/protected-route.tsx
"use client";

import { ReactNode, useEffect } from "react";
import { useRouter } from "next/navigation";
import useAuth from "@/app/auth";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!user.isPending && !user.data) {
      router.replace("/login");
    }
  }, [user.isPending, user.data, router]);

  if (user.isPending) {
    return (
      <div className="h-full flex items-center justify-center">Loading</div>
    );
  }

  if (user.data) {
    return <>{children}</>;
  }

  return null;
}
