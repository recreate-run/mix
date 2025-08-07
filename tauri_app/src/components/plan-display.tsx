import { useEffect, useState } from 'react';
import { AIResponse } from '@/components/ui/kibo-ui/ai/response';

interface PlanOptionProps {
  text: string;
  onClick: () => void;
  focused: boolean;
  number: number;
}

function PlanOption({ text, onClick, focused, number }: PlanOptionProps) {
  return (
    <button
      className={`block w-full rounded px-3 py-2 text-left font-mono text-sm transition-colors ${
        focused
          ? 'border border-blue-300 bg-blue-100 dark:bg-blue-900/30'
          : 'hover:bg-gray-100 dark:hover:bg-gray-700'
      }`}
      onClick={onClick}
    >
      <span className="ml-1">
        {number}. {text}
      </span>
    </button>
  );
}

type PlanDisplayProps = {
  planContent: string;
  showOptions?: boolean;
  onProceed?: () => void;
  onKeepPlanning?: () => void;
};

export function PlanDisplay({
  planContent,
  showOptions = false,
  onProceed,
  onKeepPlanning,
}: PlanDisplayProps) {
  const [focusedIndex, setFocusedIndex] = useState(0);

  useEffect(() => {
    if (!showOptions) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setFocusedIndex(1);
          break;
        case 'ArrowUp':
          e.preventDefault();
          setFocusedIndex(0);
          break;
        case 'Enter':
          e.preventDefault();
          if (focusedIndex === 0 && onProceed) {
            onProceed();
          } else if (focusedIndex === 1 && onKeepPlanning) {
            onKeepPlanning();
          }
          break;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [showOptions, focusedIndex, onProceed, onKeepPlanning]);

  if (!planContent) {
    return null;
  }

  return (
    <div className="rounded-xl border-2 p-4">
      <AIResponse>{planContent}</AIResponse>

      {showOptions && onProceed && onKeepPlanning && (
        <div className="mt-6 border-gray-200 border-t pt-4 dark:border-gray-600">
          <div className="font-mono text-sm">
            <div className="mb-3 text-gray-700 dark:text-gray-300">
              Would you like to proceed?
            </div>
            <div className="space-y-2">
              <PlanOption
                focused={focusedIndex === 0}
                number={1}
                onClick={onProceed}
                text="Yes, and auto-accept edits"
              />
              <PlanOption
                focused={focusedIndex === 1}
                number={2}
                onClick={onKeepPlanning}
                text="No, keep planning"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
