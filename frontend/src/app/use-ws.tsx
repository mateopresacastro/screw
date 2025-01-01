import { useState, useRef, useEffect } from "react";

interface ProgressMessage {
  type: string;
  progress: number;
  receivedSize: number;
  totalSize: number;
}

type Status = "streaming" | "creatingUrl" | "init" | "error";

export default function useWebSocket(file: File) {
  const [processProgress, setProcessProgress] = useState<number>(0);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [status, setStatus] = useState<Status>("init");
  const audioChunks = useRef<Blob[]>([]);

  const isStreaming = status === "streaming";
  const isError = status === "error";
  const isCreatingUrl = status === "creatingUrl";

  useEffect(() => {
    const socket = new WebSocket("ws://localhost:3000/ws");
    socket.addEventListener("open", handleOpen);
    socket.addEventListener("message", handleMessage);
    socket.addEventListener("close", handleDisconnect);
    socket.addEventListener("error", handleError);

    async function handleOpen() {
      const message = {
        fileSize: file.size,
        fileName: file.name,
        mimeType: file.type,
      };
      socket.send(JSON.stringify(message));

      const chunkSize = 64 * 1024;
      const totalChunks = Math.ceil(file.size / chunkSize);

      setStatus("streaming");

      for (let i = 0; i < totalChunks; i++) {
        const start = i * chunkSize;
        const end = Math.min(start + chunkSize, file.size);
        const chunk = file.slice(start, end);
        await new Promise<void>((resolve, reject) => {
          const reader = new FileReader();
          reader.onload = async (e) => {
            if (socket.readyState !== WebSocket.OPEN || !e.target?.result) {
              return;
            }

            try {
              if (typeof e.target.result === "string") {
                reject(new TypeError());
              }
              socket.send(e.target.result);
              resolve();
            } catch (error) {
              reject(error);
            }
          };
          reader.onerror = reject;
          reader.readAsArrayBuffer(chunk);
        });
      }
    }

    function handleMessage(event: MessageEvent) {
      if (event.data instanceof Blob) {
        const audioBlob = new Blob([event.data], { type: "audio/aac" });
        audioChunks.current.push(audioBlob);
        return;
      }

      try {
        const message = JSON.parse(event.data);
        if (message.type !== "progress") return;
        const { progress } = message as ProgressMessage;
        setProcessProgress(progress);
        if (progress !== 100) return;
        const blob = new Blob(audioChunks.current, {
          type: "audio/aac",
        });
        const url = URL.createObjectURL(blob);
        console.log("Creating audio URL:", url);
        setAudioUrl(url);
        socket.close();
        setStatus("init");
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    }

    function handleDisconnect() {
      setProcessProgress(0);
      if (audioUrl) {
        URL.revokeObjectURL(audioUrl);
        setAudioUrl(null);
      }
      audioChunks.current = [];
    }

    function handleError(error: unknown) {
      console.error("WebSocket error:", error);
      setError(
        error instanceof Error
          ? error
          : new Error("Error on websocket", { cause: error })
      );
    }

    return () => {
      if (socket.readyState === WebSocket.OPEN) socket.close();
      if (audioUrl) URL.revokeObjectURL(audioUrl);
      socket.removeEventListener("open", handleOpen);
      socket.removeEventListener("message", handleMessage);
      socket.removeEventListener("close", handleDisconnect);
      socket.removeEventListener("error", handleError);
    };
  }, []);

  return {
    isStreaming,
    processProgress,
    audioUrl,
    error,
    isError,
    isCreatingUrl,
  };
}
