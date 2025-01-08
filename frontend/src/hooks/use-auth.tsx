import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { z } from "zod";

const SessionSchema = z.object({
  name: z.string(),
  picture: z.string().url(),
  email: z.string().email(),
});

export default function useAuth() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const user = useQuery({
    queryKey: ["session"],
    queryFn: async () => {
      const res = await fetch("http://localhost:3000/login/session", {
        credentials: "include",
      });
      if (!res.ok) throw new Error("Unauthorized");
      const json = await res.json();
      return SessionSchema.parse(json);
    },
    retry: false,
  });

  const { mutate: logout } = useMutation({
    mutationKey: ["logout"],
    mutationFn: async () => {
      const res = await fetch("http://localhost:3000/logout", {
        credentials: "include",
        method: "POST",
      });
      if (!res.ok) throw new Error("Logout failed");
      queryClient.removeQueries({ queryKey: ["session"] });
      router.push("/login");
    },
  });

  return { user, logout };
}
