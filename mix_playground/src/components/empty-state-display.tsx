export function EmptyStateDisplay() {
  return (
    <div className="fade-in flex min-h-[60vh] animate-in flex-col items-center justify-center space-y-12 duration-1000">
      {/* Mix Logo */}
      <div className="space-y-4 text-center">
        <img
          alt="Mix Logo"
          className="slide-in-from-top mx-auto mb-6 size-48 animate-in object-contain duration-1000"
          src="/mix_logo.png"
        />
        {/* <div className="text-2xl text-muted-foreground animate-in slide-in-from-bottom duration-1000">
          The multimodal agents SDK
        </div> */}
      </div>
    </div>
  );
}
