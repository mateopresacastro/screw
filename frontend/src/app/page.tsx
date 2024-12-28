"use client";

import { useMutation } from "@tanstack/react-query";
import {
  GrpcWebImpl,
  TaggerClientImpl,
  type HelloRequest,
} from "@/proto/tagger";

const grpcClient = new GrpcWebImpl("http://localhost:3000", {
  debug: true,
});

const client = new TaggerClientImpl(grpcClient);

export default function Home() {
  const mutation = useSayHelloMutation();

  const handleSubmit = () => {
    if (mutation.error) {
      mutation.reset();
    } else {
      mutation.mutate("Mateo");
    }
  };

  return (
    <div className="h-full flex items-center justify-center">
      <div>
        <button onClick={handleSubmit} disabled={mutation.isPending}>
          Say Hello
        </button>

        {mutation.isPending && <p>Sending greeting...</p>}
        {mutation.error && <p>{mutation.error.toString()}</p>}
        {mutation.data && <p>Response: {mutation.data.message}</p>}
      </div>
    </div>
  );
}

function useSayHelloMutation() {
  return useMutation({
    mutationFn: async (name: string) => {
      const request: HelloRequest = {
        name,
      };
      return await client.SayHello(request);
    },
  });
}
