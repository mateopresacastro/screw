"use client";

import useAuth from "@/app/auth";
import useWebSocket from "@/app/use-ws";
import { ProtectedRoute } from "@/components/protected-route";
import NumberFlow from "@number-flow/react";
import { useState, type ChangeEvent } from "react";

export default function Home() {
  const [files, setFiles] = useState<File[] | null>(null);
  const { logout, user } = useAuth();

  function handleFileSelect(e: ChangeEvent<HTMLInputElement>) {
    const { files } = e.target;
    if (!files) return;
    const arrOfFiles = Array.from(files).filter((file) => file instanceof File);
    setFiles(arrOfFiles);
  }

  return (
    <ProtectedRoute>
      <div className="h-full flex flex-col items-start justify-center gap-4 max-w-md mx-auto px-4 md:px-0">
        <h1>SCREW</h1>
        {user.data && (
          <div>
            <img
              src={user.data.picture}
              className="size-7 rounded-full"
              alt={user.data.name}
            />
            <p>{user.data.email}</p>
            <p>{user.data.name}</p>
          </div>
        )}
        <button onClick={() => logout()}>log out</button>
        <input
          type="file"
          onChange={handleFileSelect}
          multiple
          accept="audio/*"
          className="w-full"
          max={5}
        />
        {files?.map((file) => (
          <AudioFile file={file} key={file.name.concat(String(file.size))} />
        ))}
      </div>
    </ProtectedRoute>
  );
}

function AudioFile({ file }: { file: File }) {
  const { isStreaming, processProgress, audioUrl } = useWebSocket(file);
  return (
    <div className="w-full pt-2">
      <div className="flex justify-between items-center mb-2 text-xs">
        <span className="truncate max-w-[70%]">{file.name}</span>
        {isStreaming ? (
          <NumberFlow
            value={processProgress}
            format={{
              minimumFractionDigits: 0,
              maximumFractionDigits: 0,
            }}
            willChange
            suffix=" %"
            trend={+1}
          />
        ) : null}
      </div>
      {audioUrl && (
        <div className="w-full">
          <audio controls className="w-full" src={audioUrl}>
            Your browser does not support the audio element.
          </audio>
        </div>
      )}
    </div>
  );
}
