export function LoadingDots() {
  return (
    <div className="flex items-center space-x-1">
      <div
        className="h-1 w-1 animate-bounce rounded-full bg-gray-400"
        style={{ animationDelay: '0ms' }}
      />
      <div
        className="h-1 w-1 animate-bounce rounded-full bg-gray-400"
        style={{ animationDelay: '150ms' }}
      />
      <div
        className="h-1 w-1 animate-bounce rounded-full bg-gray-400"
        style={{ animationDelay: '300ms' }}
      />
    </div>
  );
}
