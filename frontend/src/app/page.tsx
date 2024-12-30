"use client";

import useWebSocket from "@/app/use-ws";
import NumberFlow from "@number-flow/react";
import { useState, type ChangeEvent } from "react";

export default function Home() {
  const [files, setFiles] = useState<File[] | null>(null);

  function handleFileSelect(e: ChangeEvent<HTMLInputElement>) {
    const { files } = e.target;
    console.log(files);
    if (!files) return;
    const arrOfFiles = Array.from(files).filter((file) => file instanceof File);
    console.log(arrOfFiles);
    setFiles(arrOfFiles);
  }

  return (
    <div className="h-full flex flex-col items-start justify-center gap-4 max-w-md mx-auto px-4 md:px-0">
      <h1>SCREW</h1>
      <input
        type="file"
        onChange={handleFileSelect}
        multiple
        accept="audio/*"
        className="w-full"
        max={5}
      />
      {files
        ? files.map((file) => (
            <AudioFile file={file} key={file.name.concat(String(file.size))} />
          ))
        : null}
    </div>
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
