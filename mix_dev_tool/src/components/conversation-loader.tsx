export const ConversationLoader = () => {
	return (
		<div className="flex items-center gap-2 text-muted-foreground">
			<div className="flex gap-1">
				<div className="h-2 w-2 animate-bounce rounded-full bg-current [animation-delay:-0.3s]" />
				<div className="h-2 w-2 animate-bounce rounded-full bg-current [animation-delay:-0.15s]" />
				<div className="h-2 w-2 animate-bounce rounded-full bg-current" />
			</div>
			<span className="text-sm">Processing...</span>
		</div>
	);
};
