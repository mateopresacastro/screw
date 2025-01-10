"use client";

import useWebSocket from "@/hooks/use-ws";
import WaveForm from "@/components/waveform";
import NumberFlow from "@number-flow/react";
import { Input } from "@/components/input";
import { AnimatePresence, motion } from "motion/react";
import { useState, type ChangeEvent } from "react";
import type { Session } from "@/auth";

export default function Main({ session }: { session: Session | null }) {
  const [files, setFiles] = useState<File[] | null>(null);

  function handleFileSelect(e: ChangeEvent<HTMLInputElement>) {
    const { files } = e.target;
    if (!files) return;
    const arrOfFiles = Array.from(files).filter((file) => file instanceof File);
    setFiles(arrOfFiles);
  }

  return (
    <div className="h-full flex flex-col items-start justify-start w-full">
      {session ? (
        <span className="block">
          Hello {session.name.split(" ").at(0) ?? "unknown"}
        </span>
      ) : null}
      <span className="block pb-2">Select your audio files to screw them:</span>
      <div className="w-full">
        <Input
          type="file"
          onChange={handleFileSelect}
          multiple
          accept="audio/*"
          max={5}
        />
      </div>
      {files?.slice(0, 5).map((file, i) => (
        <AudioFile
          file={file}
          key={file.name.concat(String(file.size))}
          index={i}
        />
      ))}
    </div>
  );
}

function AudioFile({ file, index }: { file: File; index: number }) {
  const { isStreaming, processProgress, audioBlob } = useWebSocket(file);
  return (
    <div className="w-full pt-2 flex flex-col mt-24">
      <AnimatePresence mode="popLayout">
        <motion.div
          className="flex justify-between items-start h-16"
          style={{ flexDirection: audioBlob ? "column" : "row" }}
        >
          <motion.span
            className="truncate max-w-[70%]"
            layoutId={file.name.concat(String(index))}
            key={file.name.concat(String(index))}
          >
            {file.name}
          </motion.span>
          {isStreaming ? (
            <motion.div exit={{ opacity: 0 }}>
              <NumberFlow
                value={processProgress}
                format={{
                  minimumFractionDigits: 0,
                  maximumFractionDigits: 0,
                }}
                suffix=" %"
                trend={+1}
              />
            </motion.div>
          ) : null}

          {audioBlob ? (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ delay: 1 }}
              key={`wave-${file.name.concat(String(index))}`}
              className="w-full"
            >
              <WaveForm blob={audioBlob} fileName={file.name} />
            </motion.div>
          ) : null}
        </motion.div>
      </AnimatePresence>
    </div>
  );
}
