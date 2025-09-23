export function EmptyStateDisplay() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] space-y-12 animate-in fade-in duration-1000">
      {/* Mix Logo */}
      <div className="text-center space-y-4">
        <img
          src="/mix_logo.png"
          alt="Mix Logo"
          className="size-48 object-contain mx-auto mb-6 animate-in slide-in-from-top duration-1000"
        />
        {/* <div className="text-2xl text-muted-foreground animate-in slide-in-from-bottom duration-1000">
          The multimodal agents SDK
        </div> */}
      </div>
    </div>
  );
}