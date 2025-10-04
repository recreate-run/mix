import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableCaption,
} from '@/components/ui/table';
import { useCsvData } from '@/hooks/useCsv';

interface CsvViewerProps {
  url: string;
  title: string;
}

export const CsvViewer = ({ url }: CsvViewerProps) => {
  const { data: csvData, isLoading: loading, error } = useCsvData(url);

  if (loading) {
    return (
      <div className="flex h-48 items-center justify-center rounded-md bg-stone-700/30">
        <div className="text-stone-400">Loading CSV data...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-48 items-center justify-center rounded-md bg-stone-700/30">
        <div className="text-stone-400">
          Failed to load CSV:{' '}
          {error instanceof Error ? error.message : 'Unknown error'}
        </div>
      </div>
    );
  }

  if (!csvData || csvData.headers.length === 0) {
    return (
      <div className="flex h-48 items-center justify-center rounded-md bg-stone-700/30">
        <div className="text-stone-400">No CSV data found</div>
      </div>
    );
  }

  const maxDisplayRows = 100; // Limit rows for performance
  const displayRows = csvData.rows.slice(0, maxDisplayRows);
  const hasMoreRows = csvData.rows.length > maxDisplayRows;

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="max-h-96 overflow-auto">
        <Table>
          <TableCaption>
            {csvData.totalRows} rows × {csvData.totalColumns} columns
            {hasMoreRows && (
              <span className="text-muted-foreground">
                {' '}
                (showing first {maxDisplayRows} rows)
              </span>
            )}
          </TableCaption>
          <TableHeader>
            <TableRow>
              {csvData.headers.map((header, index) => (
                <TableHead key={index} className="bg-muted/50 font-medium">
                  {header || `Column ${index + 1}`}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {displayRows.map((row, rowIndex) => (
              <TableRow key={rowIndex}>
                {csvData.headers.map((_, colIndex) => (
                  <TableCell key={colIndex} className="max-w-xs truncate">
                    {row[colIndex] || ''}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
};
