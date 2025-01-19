"use server";

import "server-only";

import { cookies } from "next/headers";
import { z } from "zod";

const SessionSchema = z.object({
  name: z.string(),
  picture: z.string().url(),
  email: z.string().email(),
});

export type Session = z.infer<typeof SessionSchema>;

export default async function auth() {
  const cookieStore = await cookies();
  const session = cookieStore.get("session");
  const url = "http://proxy/api/login/session";
  try {
    const res = await fetch(url, {
      headers: {
        Cookie: `session=${session?.value ?? ""}`,
      },
      cache: "no-store",
    });
    if (!res.ok) throw new Error("Unauthorized");
    const json = await res.json();
    return SessionSchema.parse(json);
  } catch (e) {
    console.log(e);
    return null;
  }
}
