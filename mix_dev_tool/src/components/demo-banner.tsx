export function DemoBanner() {
	return (
		<div className="border-b border-blue-500/20 bg-blue-500/10 p-3 text-center">
			<p className="text-blue-300 text-sm">
				🎮 Demo Mode - Exploring Mix in your browser.{" "}
				<a
					href="https://github.com/recreate-run/mix"
					className="underline hover:text-blue-200"
					target="_blank"
					rel="noopener noreferrer"
				>
					Get the full desktop app
				</a>
			</p>
		</div>
	);
}
