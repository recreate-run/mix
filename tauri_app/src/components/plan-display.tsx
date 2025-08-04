import { AIResponse } from '@/components/ui/kibo-ui/ai/response';
import { useState, useEffect } from 'react';

interface PlanOptionProps {
  text: string;
  onClick: () => void;
  focused: boolean;
  number: number;
}

function PlanOption({ text, onClick, focused, number }: PlanOptionProps) {
  return (
    <button
      onClick={onClick}
      className={`block w-full text-left px-3 py-2 rounded transition-colors font-mono text-sm ${
        focused 
          ? 'bg-blue-100 dark:bg-blue-900/30 border border-blue-300' 
          : 'hover:bg-gray-100 dark:hover:bg-gray-700'
      }`}
    >
      <span className="ml-1">{number}. {text}</span>
    </button>
  );
}

type PlanDisplayProps = {
  planContent: string;
  showOptions?: boolean;
  onProceed?: () => void;
  onKeepPlanning?: () => void;
};

export function PlanDisplay({ planContent, showOptions = false, onProceed, onKeepPlanning }: PlanDisplayProps) {
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
    <div className="border-2 rounded-xl p-4">
      <AIResponse>
        {planContent}
      </AIResponse>
      
      {showOptions && onProceed && onKeepPlanning && (
        <div className="mt-6 border-t border-gray-200 dark:border-gray-600 pt-4">
          <div className="font-mono text-sm">
            <div className="text-gray-700 dark:text-gray-300 mb-3">
              Would you like to proceed?
            </div>
            <div className="space-y-2">
              <PlanOption
                text="Yes, and auto-accept edits"
                onClick={onProceed}
                focused={focusedIndex === 0}
                number={1}
              />
              <PlanOption
                text="No, keep planning"
                onClick={onKeepPlanning}
                focused={focusedIndex === 1}
                number={2}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}