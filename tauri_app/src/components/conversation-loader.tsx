import { DotFlow, type DotFlowProps } from '../../components/gsap/dot-flow';

const thinking = [
  [],
  [24],
  [17, 24, 31],
  [10, 17, 24, 31, 38],
  [3, 10, 17, 24, 31, 38, 45],
  [3, 10, 17, 24, 31, 38, 45],
  [10, 17, 24, 31, 38],
  [17, 24, 31],
  [24],
  [],
];

const processing = [
  [21, 22, 23, 25, 26, 27],
  [14, 15, 16, 32, 33, 34],
  [7, 8, 9, 39, 40, 41],
  [0, 1, 2, 46, 47, 48],
  [0, 1, 2, 46, 47, 48],
  [7, 8, 9, 39, 40, 41],
  [14, 15, 16, 32, 33, 34],
  [21, 22, 23, 25, 26, 27],
];

const analyzing = [
  [24],
  [17, 31],
  [10, 24, 38],
  [3, 17, 31, 45],
  [10, 24, 38],
  [17, 31],
  [24],
  [],
  [1, 7, 41, 47],
  [8, 16, 32, 40],
  [1, 7, 41, 47],
  [],
];

const items: DotFlowProps['items'] = [
  {
    title: 'Thinking',
    frames: thinking,
    duration: 200,
    repeatCount: 2,
  },
  {
    title: 'Processing',
    frames: processing,
    duration: 150,
    repeatCount: 2,
  },
  {
    title: 'Analyzing',
    frames: analyzing,
    duration: 180,
    repeatCount: 2,
  },
];

export const ConversationLoader = () => {
  return <DotFlow items={items} />;
};
