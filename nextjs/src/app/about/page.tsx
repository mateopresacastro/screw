"use client";

import Header from "@/components/header";
import Link from "../../../$node_modules/next/link.js";

export default function Home() {
  return (
    <div className="h-full flex flex-col items-start justify-start w-full">
      <Header />
      <Link
        href="https://en.wikipedia.org/wiki/Chopped_and_screwed"
        className="underline decoration-gray-700 hover:decoration-gray-1000 leading-6"
      >
        Chopped + screwed
      </Link>
      <Link
        href="https://en.wikipedia.org/wiki/Chopped_and_screwed#Slowed_and_reverb"
        className="underline decoration-gray-700 hover:decoration-gray-1000 leading-6 mb-36"
      >
        Slowed + reverb
      </Link>
      <p className="pb-36">A little project to learn Go</p>

      <Link href="https://mateo.id" className="mb-36">
        -{" "}
        <span className="underline decoration-gray-700 hover:decoration-gray-1000">
          Mateo
        </span>
      </Link>
    </div>
  );
}
