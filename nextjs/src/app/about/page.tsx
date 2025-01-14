import auth from "@/auth";
import Link from "next/link";

export default async function About() {
  const user = await auth();
  if (!user) return "This a protected route. Log in.";
  return (
    <div className="h-full flex flex-col items-start justify-start w-full leading-6">
      <p className="pb-36 ">
        A little project with Go, Next.js, 0Auth2.0, DB Sessions, NGINX, SQLite,
        WebSockets, FFmpeg, Docker, and Docker-Compose.
      </p>
      <p className="pb-36 ">
        The audio streams to a Go API via WebSocket connections, where FFmpeg
        processes each stream in real-time. The processed audio is returned
        through the same connections and buffered client-side, where it&apos;s
        rendered as a waveform.
      </p>
      <Link href="https://mateo.id" className="mb-36">
        -{" "}
        <span className="underline decoration-gray-700 hover:decoration-gray-1000">
          Mateo
        </span>
      </Link>
    </div>
  );
}
