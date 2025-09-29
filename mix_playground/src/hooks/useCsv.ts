import { useQuery } from '@tanstack/react-query';
import { CACHE_KEYS } from '@/lib/cache-keys';

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

async function fetchCsvData(url: string): Promise<ParsedCsv> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const csvText = await response.text();
  return parseCSV(csvText);
}

export function useCsvData(url: string) {
  return useQuery({
    queryKey: CACHE_KEYS.csvData(url),
    queryFn: () => fetchCsvData(url),
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    enabled: !!url,
  });
}