import { AlertTriangle, Clock } from 'lucide-react';
import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';

interface RateLimitDisplayProps {
  retryAfter?: number;
  attempt?: number;
  maxAttempts?: number;
  error?: string;
}

export function RateLimitDisplay({ retryAfter = 60, attempt = 1, maxAttempts = 8, error }: RateLimitDisplayProps) {
  const [timeLeft, setTimeLeft] = useState<number>(retryAfter);
  const [progress, setProgress] = useState<number>(0);
  const maxTime = retryAfter;

  useEffect(() => {
    // Only start countdown if we have a valid retryAfter value
    if (!retryAfter || retryAfter <= 0) return;

    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [retryAfter]);

  useEffect(() => {
    // Calculate progress percentage inversely (100% when timeLeft is 0)
    const calculatedProgress = ((maxTime - timeLeft) / maxTime) * 100;
    setProgress(calculatedProgress);
  }, [timeLeft, maxTime]);

  return (
    <Card className="border-yellow-200 bg-yellow-50 dark:border-yellow-900 dark:bg-yellow-950">
      <CardHeader className="pb-2">
        <div className="flex items-center gap-2">
          <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
          <CardTitle className="text-yellow-800 dark:text-yellow-200">
            Rate Limit Reached
          </CardTitle>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <p className="text-yellow-700 dark:text-yellow-300">{error || "This request would exceed your account's rate limit. The application will automatically retry."}</p>
          
          <div className="flex flex-col gap-2">
            <div className="flex justify-between text-sm">
              <span className="text-yellow-700 dark:text-yellow-300">
                Attempt {attempt} of {maxAttempts}
              </span>
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-yellow-600 dark:text-yellow-400" />
                <span className="text-yellow-700 dark:text-yellow-300">
                  {timeLeft > 0 ? `Retrying in ${timeLeft}s` : "Retrying now..."}
                </span>
              </div>
            </div>
            <Progress value={progress} className="h-2 bg-yellow-200 dark:bg-yellow-800">
              <div 
                className="h-full bg-yellow-500 dark:bg-yellow-500 transition-all" 
                style={{ width: `${progress}%` }} 
              />
            </Progress>
          </div>

          <div className="text-sm text-yellow-700 dark:text-yellow-300 bg-yellow-100 dark:bg-yellow-900 p-2 rounded">
            <p className="font-medium">Why this happens:</p>
            <p>Anthropic's API enforces rate limits to ensure fair usage across all users.</p>
            <p>The application will automatically retry your request when possible.</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}