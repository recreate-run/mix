import { AlertTriangle, Info } from 'lucide-react';
import { Progress } from '@/components/ui/progress';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface ComponentBreakdown {
  name: string;
  tokens: number;
  percentage: number;
  isTotal?: boolean;
}

interface ContextData {
  model: string;
  maxTokens: number;
  totalTokens: number;
  usagePercent: number;
  components: ComponentBreakdown[];
  warningLevel?: string;
  warningMessage?: string;
}

interface ContextDisplayProps {
  data: ContextData;
}

export function ContextDisplay({ data }: ContextDisplayProps) {
  const formatTokens = (tokens: number) => {
    if (tokens >= 1000) {
      return `${Math.round(tokens / 1000)}K`;
    }
    return tokens.toString();
  };

  const getWarningIcon = () => {
    if (data.warningLevel === 'high') {
      return <AlertTriangle className="h-4 w-4 text-red-500" />;
    }
    if (data.warningLevel === 'medium') {
      return <Info className="h-4 w-4 text-yellow-500" />;
    }
    return null;
  };

  const getProgressColor = () => {
    if (data.usagePercent > 80) return 'bg-red-500';
    if (data.usagePercent > 60) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="space-y-2">
        <h3 className="font-semibold text-lg">Context Usage Breakdown</h3>
        <p className="text-muted-foreground text-sm">
          {formatTokens(data.totalTokens)} / {formatTokens(data.maxTokens)} (
          {Math.round(data.usagePercent)}%) • {data.model}
        </p>
      </div>

      {/* Progress Bar */}
      <div className="space-y-2">
        <div className="flex items-center justify-between text-sm">
          <span>Usage</span>
          <span>{data.usagePercent.toFixed(1)}%</span>
        </div>
        <div className="relative">
          <Progress className="h-3" value={data.usagePercent} />
          <div
            className={`absolute top-0 left-0 h-3 rounded-l-full transition-all ${getProgressColor()}`}
            style={{ width: `${Math.min(data.usagePercent, 100)}%` }}
          />
        </div>
      </div>

      {/* Component Breakdown Table */}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Component</TableHead>
            <TableHead className="text-right">Tokens</TableHead>
            <TableHead className="text-right">Percentage</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.components.map((component, index) => (
            <TableRow
              className={component.isTotal ? 'border-t-2 font-semibold' : ''}
              key={index}
            >
              <TableCell>{component.name}</TableCell>
              <TableCell className="text-right">
                {component.name === 'System Prompt' ||
                component.name === 'Tool Descriptions'
                  ? '~'
                  : ''}
                {formatTokens(component.tokens)}
              </TableCell>
              <TableCell className="text-right">
                {component.percentage.toFixed(1)}%
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {/* Warning Message */}
      {data.warningMessage && (
        <div className="flex items-center gap-2 rounded-md bg-muted/50 p-3">
          {getWarningIcon()}
          <span className="text-sm">{data.warningMessage}</span>
        </div>
      )}
    </div>
  );
}
