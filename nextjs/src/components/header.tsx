import useAuth from "@/hooks/use-auth";
import Link from "next/link";

export default function Header() {
  const { logout } = useAuth();
  return (
    <div className="w-full flex pb-36 items-baseline justify-between">
      <Link href="/">
        <h1 className="underline decoration-gray-700 hover:decoration-gray-1000">
          Screw
        </h1>
      </Link>
      <Link href="/about" className="hover:text-gray-1100 px-3">
        ?
      </Link>
      <button
        onClick={() => logout()}
        className="underline decoration-gray-700 hover:decoration-gray-1000"
      >
        Log out
      </button>
    </div>
  );
}
