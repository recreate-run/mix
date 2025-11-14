// Component to render JSON as flat key-value list
export const JSONDisplay = ({
	data,
	title,
}: {
	data: string;
	title: string;
}) => {
	try {
		const parsed = JSON.parse(data);
		const flattenObject = (
			obj: unknown,
			prefix = "",
		): Record<string, unknown> => {
			const flattened: Record<string, unknown> = {};
			if (!obj || typeof obj !== "object") return flattened;
			const record = obj as Record<string, unknown>;
			for (const key in record) {
				const fullKey = prefix ? `${prefix}.${key}` : key;
				const value = record[key];
				if (
					value !== null &&
					typeof value === "object" &&
					!Array.isArray(value)
				) {
					Object.assign(flattened, flattenObject(value, fullKey));
				} else {
					flattened[fullKey] = value;
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
	} catch {
		return (
			<div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm">
				<div className="font-semibold text-destructive">Invalid JSON</div>
				<div className="mt-2 text-muted-foreground">
					Failed to parse JSON data
				</div>
			</div>
		);
	}
};
