interface AudioWaveformProps {
  className?: string;
  small?: boolean;
}

export const AudioWaveform = ({
  className = '',
  small = false,
}: AudioWaveformProps) => {
  const bars = small ? 8 : 12;
  const heights = small
    ? [0.3, 0.7, 0.5, 0.9, 0.4, 0.8, 0.6, 0.2]
    : [0.3, 0.7, 0.5, 0.9, 0.4, 0.8, 0.6, 0.2, 0.7, 0.5, 0.8, 0.3];

  return (
    <div className={`flex items-end justify-center gap-0.5 ${className}`}>
      {Array.from({ length: bars }).map((_, i) => (
        <div
          className="rounded-full bg-stone-300"
          key={i}
          style={{
            width: small ? '2px' : '3px',
            height: `${heights[i] * 100}%`,
            minHeight: small ? '2px' : '3px',
          }}
        />
      ))}
    </div>
  );
};
