import { useMutation } from "@tanstack/react-query";

type UploadResponse = {
  tagId: string;
};

export default function useUpload() {
  const mutation = useMutation<UploadResponse, Error, File>({
    mutationFn: async (file) => {
      const formData = new FormData();
      formData.append("file", file);
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
          "tagId" in value &&
          typeof value.tagId === "string"
        );
      };
      if (!isGood(json)) {
        throw new TypeError("Bad json data");
      }
      return json;
    },
  });
  return mutation;
}
