import WaveSurfer from "wavesurfer.js";
import { useTheme } from "next-themes";
import { useEffect, useRef, useState } from "react";
import { sand, sandDark } from "@radix-ui/colors";
import { IoPlaySharp, IoPauseSharp } from "react-icons/io5";
import { ArrowDownToLine } from "lucide-react";

export default function WaveForm({
  blob,
  fileName,
}: {
  blob: Blob;
  fileName: string;
}) {
  const waveformRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WaveSurfer | null>(null);
  const { resolvedTheme } = useTheme();
  const [isPlaying, setIsplaying] = useState(false);

  const colorPallete = resolvedTheme === "dark" ? sandDark : sand;

  useEffect(() => {
    if (!blob || !waveformRef.current) return;
    const OPTIONS = {
      container: waveformRef.current,
      barHeight: 5,
      barWidth: 0.2,
      height: 27,
      normalize: true,
      waveColor: colorPallete.sand9,
      progressColor: colorPallete.sand10,
      cursorColor: colorPallete.sand11,
      hideScrollbar: true,
    };
    const ws = WaveSurfer.create(OPTIONS);
    wsRef.current = ws;
    ws.loadBlob(blob);
    return () => ws.destroy();
  }, [blob, colorPallete.sand9, colorPallete.sand11, colorPallete.sand10]);

  function handlePlayPause() {
    if (!wsRef.current) return;
    wsRef.current.playPause()
    setIsplaying((s) => !s);
  }

  return (
    <div className="w-full flex items-center justify-center -ml-[0.155rem]">
      <div
        onClick={handlePlayPause}
        className="mr-6 text-gray-1000 cursor-pointer hover:text-gray-1200"
      >
        {isPlaying ? (
          <IoPauseSharp className="size-[1.12rem]" />
        ) : (
          <IoPlaySharp className="size-[1.12rem]" />
        )}
      </div>
      <div ref={waveformRef} className="w-full cursor-pointer" />
      <div
        onClick={() => download(blob, fileName)}
        className="ml-6 text-gray-800 cursor-pointer hover:text-gray-1200"
      >
        <ArrowDownToLine className="size-[1.12rem]" />
      </div>
    </div>
  );
}

export function download(blob: Blob, fileName: string) {
  const newFileURL = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = newFileURL;
  link.download = `screw-${fileName}`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(newFileURL);
}
