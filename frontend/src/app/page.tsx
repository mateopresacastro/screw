"use client";

import { useEffect, useState, useRef } from "react";

interface ProgressMessage {
  type: string;
  progress: number;
  receivedSize: number;
  totalSize: number;
}

export default function Home() {
  const [socketStatus, setSocketStatus] = useState<string>("Disconnected");
  const [processProgress, setProcessProgress] = useState<number>(0);
  const socketRef = useRef<WebSocket | null>(null);
  const totalUploadedRef = useRef<number>(0);
  const audioChunks = useRef<Blob[]>([]);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);

  useEffect(() => {
    const socket = new WebSocket("ws://localhost:3000/ws");
    socketRef.current = socket;

    function handleOpen() {
      setSocketStatus("Connected");
      audioChunks.current = [];
      if (audioUrl) {
        URL.revokeObjectURL(audioUrl);
        setAudioUrl(null);
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
        if (message.type === "progress") {
          const progressData = message as ProgressMessage;
          setProcessProgress(progressData.progress);

          if (progressData.progress === 100) {
            const blob = new Blob(audioChunks.current, {
              type: "audio/aac",
            });
            const url = URL.createObjectURL(blob);
            console.log("Creating audio URL:", url);
            setAudioUrl(url);
            socket.close();
          }
        }
      } catch (error) {
        console.error("Error parsing message:", error);
      }
    }

    function handleDisconnect() {
      setSocketStatus("Disconnected");
      totalUploadedRef.current = 0;
      setProcessProgress(0);
      if (audioUrl) {
        URL.revokeObjectURL(audioUrl);
        setAudioUrl(null);
      }
      audioChunks.current = [];
    }

    function handleError(error: unknown) {
      console.error("WebSocket error:", error);
      setSocketStatus("Error");
    }

    socket.addEventListener("open", handleOpen);
    socket.addEventListener("message", handleMessage);
    socket.addEventListener("close", handleDisconnect);
    socket.addEventListener("error", handleError);

    return () => {
      if (socket.readyState === WebSocket.OPEN) socket.close();
      if (audioUrl) URL.revokeObjectURL(audioUrl);
      socket.removeEventListener("open", handleOpen);
      socket.removeEventListener("message", handleMessage);
      socket.removeEventListener("close", handleDisconnect);
      socket.removeEventListener("error", handleError);
    };
  }, []);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !socketRef.current) return;

    setProcessProgress(0);
    totalUploadedRef.current = 0;

    const message = {
      fileSize: file.size,
      fileName: file.name,
      mimeType: file.type,
    };

    console.log("sendign metadata", message);
    socketRef.current.send(JSON.stringify(message));

    const chunkSize = 64 * 1024;
    const totalChunks = Math.ceil(file.size / chunkSize);

    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunk = file.slice(start, end);

      await new Promise<void>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = async (e) => {
          if (
            socketRef.current?.readyState === WebSocket.OPEN &&
            e.target?.result
          ) {
            try {
              if (typeof e.target.result === "string") reject(new TypeError());
              socketRef.current.send(e.target.result);
              totalUploadedRef.current += chunk.size;
              resolve();
            } catch (error) {
              reject(error);
            }
          }
        };
        reader.onerror = reject;
        reader.readAsArrayBuffer(chunk);
      });
    }
  };

  return (
    <div className="h-full flex flex-col items-center justify-center gap-4">
      <h1>WebSocket File Transfer Test</h1>
      <p>Status: {socketStatus}</p>
      <input
        type="file"
        onChange={handleFileSelect}
        disabled={socketStatus !== "Connected"}
      />
      {processProgress > 0 && (
        <div className="w-full max-w-md">
          <div className="mb-2">
            Processing Progress: {processProgress.toFixed(2)}%
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2.5">
            <div
              className="bg-green-600 h-2.5 rounded-full transition-all duration-300"
              style={{ width: `${processProgress}%` }}
            ></div>
          </div>
        </div>
      )}
      {audioUrl && (
        <audio controls src={audioUrl} className="mt-4">
          Your browser does not support the audio element.
        </audio>
      )}
    </div>
  );
}
