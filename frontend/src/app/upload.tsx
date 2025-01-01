import { useMutation } from "@tanstack/react-query";

type UploadResponse = {
  ref: string;
};

export default function useUpload() {
  const mutation = useMutation({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append("file", file!);
      const res = await fetch("http://localhost:3000/upload", {
        method: "POST",
        body: formData,
        credentials: "include",
      });
      if (!res.ok) throw new Error("Res not ok on upload");
      const json = await res.json();

      const isGood = (value: unknown): value is UploadResponse => {
        return (
          typeof value === "object" &&
          value !== null &&
          "ref" in value &&
          typeof value.ref === "string"
        );
      };

      if (!isGood(json)) {
        throw new TypeError("Bad json data");
      }
      console.log("upload response", json);
      return json;
    },
  });

  return mutation;
}
