import "./index.css";
import { Composition } from "remotion";
import { DynamicVideoComposition } from "./DynamicVideoComposition";
import { CreativeShowcase } from "./CreativeShowcase";

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="DynamicComposition"
        component={DynamicVideoComposition}
        durationInFrames={150}
        fps={30}
        width={1920}
        height={1080}
      />
      <Composition
        id="CreativeShowcase"
        component={CreativeShowcase}
        durationInFrames={180}
        fps={30}
        width={1280}
        height={720}
      />
    </>
  );
};
