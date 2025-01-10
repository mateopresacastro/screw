import auth from "@/auth";
import Link from "next/link";

export default async function About() {
  const user = await auth();
  if (!user) return "This a protected route. Log in to enter.";
  return (
    <div className="h-full flex flex-col items-start justify-start w-full leading-6">
      <p className="pb-36 ">
        A little project with Go, Next.js, 0Auth2.0, DB Sessions, NGINX, SQLite,
        WebSockets, FFmpeg, Docker, and Docker-Compose.
      </p>
      <Link
        href="https://en.wikipedia.org/wiki/Chopped_and_screwed"
        className="underline decoration-gray-700 hover:decoration-gray-1000"
      >
        Chopped + screwed
      </Link>
      <Link
        href="https://en.wikipedia.org/wiki/Chopped_and_screwed#Slowed_and_reverb"
        className="underline decoration-gray-700 hover:decoration-gray-1000 mb-36"
      >
        Slowed + reverb
      </Link>

      <Link href="https://mateo.id" className="mb-36">
        -{" "}
        <span className="underline decoration-gray-700 hover:decoration-gray-1000">
          Mateo
        </span>
      </Link>
    </div>
  );
}
