"use server";

import "server-only";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export async function logout() {
  const cookieStore = await cookies();
  const session = cookieStore.get("session");
  const res = await fetch(`${process.env.NEXT_PUBLIC_HOST}/api/logout`, {
    headers: {
      Cookie: `session=${session?.value ?? ""}`,
    },
    cache: "no-store",
    method: "POST",
  });

  if (!res.ok) throw new Error("Logout failed");
  redirect("/");
}
