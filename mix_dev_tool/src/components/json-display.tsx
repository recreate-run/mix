// Component to render JSON as flat key-value list
export const JSONDisplay = ({ data, title }: { data: string; title: string }) => {
	try {
		const parsed = JSON.parse(data);
		const flattenObject = (obj: any, prefix = ""): Record<string, any> => {
			const flattened: Record<string, any> = {};
			for (const key in obj) {
				const fullKey = prefix ? `${prefix}.${key}` : key;
				if (obj[key] !== null && typeof obj[key] === "object" && !Array.isArray(obj[key])) {
					Object.assign(flattened, flattenObject(obj[key], fullKey));
				} else {
					flattened[fullKey] = obj[key];
				}
			}
			return flattened;
		};

		const flattened = flattenObject(parsed);

		return (
			<div className="mb-4 rounded-lg border bg-muted/50 p-4">
				<div className="mb-3 font-semibold">{title}</div>
				<div className="space-y-2">
					{Object.entries(flattened).map(([key, value]) => (
						<div key={key} className="flex gap-2 text-sm">
							<span className="font-mono text-muted-foreground">{key}:</span>
							<span className="font-mono">{JSON.stringify(value)}</span>
						</div>
					))}
				</div>
			</div>
		);
	} catch (error) {
		return (
			<div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm">
				<div className="font-semibold text-destructive">Invalid JSON</div>
				<div className="mt-2 text-muted-foreground">Failed to parse JSON data</div>
			</div>
		);
	}
};
