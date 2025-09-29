import { useEffect, useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableCaption,
} from '@/components/ui/table';

interface CsvViewerProps {
  url: string;
  title: string;
}

interface ParsedCsv {
  headers: string[];
  rows: string[][];
  totalRows: number;
  totalColumns: number;
}

const parseCSV = (csvText: string): ParsedCsv => {
  const lines = csvText.trim().split('\n');
  const headers: string[] = [];
  const rows: string[][] = [];

  if (lines.length === 0) {
    return { headers, rows, totalRows: 0, totalColumns: 0 };
  }

  // Simple CSV parser - handles basic cases
  const parseLine = (line: string): string[] => {
    const result: string[] = [];
    let current = '';
    let inQuotes = false;

    for (let i = 0; i < line.length; i++) {
      const char = line[i];
      const nextChar = line[i + 1];

      if (char === '"') {
        if (inQuotes && nextChar === '"') {
          // Escaped quote
          current += '"';
          i++; // Skip next quote
        } else {
          // Toggle quote state
          inQuotes = !inQuotes;
        }
      } else if (char === ',' && !inQuotes) {
        // End of field
        result.push(current.trim());
        current = '';
      } else {
        current += char;
      }
    }

    // Add the last field
    result.push(current.trim());
    return result;
  };

  // Parse headers
  headers.push(...parseLine(lines[0]));

  // Parse data rows
  for (let i = 1; i < lines.length; i++) {
    const row = parseLine(lines[i]);
    if (row.length > 0 && row.some(cell => cell.length > 0)) {
      rows.push(row);
    }
  }

  return {
    headers,
    rows,
    totalRows: rows.length,
    totalColumns: headers.length,
  };
};

export const CsvViewer = ({ url }: CsvViewerProps) => {
  const [csvData, setCsvData] = useState<ParsedCsv | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchAndParseCsv = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await fetch(url);
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        const csvText = await response.text();
        const parsed = parseCSV(csvText);
        setCsvData(parsed);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load CSV file');
      } finally {
        setLoading(false);
      }
    };

    fetchAndParseCsv();
  }, [url]);

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
        <div className="text-stone-400">Failed to load CSV: {error}</div>
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
                {' '}(showing first {maxDisplayRows} rows)
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