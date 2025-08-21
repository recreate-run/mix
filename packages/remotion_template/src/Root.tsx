import "./index.css";
import { Composition, getInputProps } from "remotion";
import { DynamicVideoComposition, VideoConfig } from "./DynamicVideoComposition";
import { getDimensionsForFormat, VIDEO_DIMENSIONS } from "./constants/videoDimensions";

export const RemotionRoot: React.FC = () => {
  // Use default values for composition registration
  // Actual dimensions will be overridden by CLI parameters during export
  let duration = 150;
  let fps = 30;
  
  // Try to get actual values from input props if available (for preview)
  try {
    const inputProps = getInputProps() as { config?: VideoConfig };
    if (inputProps.config?.composition) {
      const { composition } = inputProps.config;
      duration = composition.durationInFrames;
      fps = composition.fps;
    }
  } catch (error) {
    // getInputProps not available during registration - use defaults
  }

  return (
    <Composition
      id="DynamicComposition"
      component={DynamicVideoComposition}
      durationInFrames={duration}
      fps={fps}
      width={VIDEO_DIMENSIONS.horizontal.width}
      height={VIDEO_DIMENSIONS.horizontal.height}
    />
  );
};
