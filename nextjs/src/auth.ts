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
  try {
    // Fetch from the "proxy" docker container (nginx).
    const res = await fetch("http://proxy/api/login/session", {
      headers: {
        Cookie: `session=${session?.value ?? ""}`,
      },
      cache: "no-store",
    });
    if (!res.ok) throw new Error("Unauthorized");
    const json = await res.json();
    return SessionSchema.parse(json);
  } catch (error) {
    return null;
  }
}
