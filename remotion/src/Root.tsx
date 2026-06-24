import "./index.css";
import { Composition } from "remotion";
import { HeroConsole } from "./HeroConsole";

export const RemotionRoot: React.FC = () => {
  return (
    <Composition
      id="HeroConsole"
      component={HeroConsole}
      durationInFrames={270}
      fps={30}
      width={1600}
      height={900}
    />
  );
};
