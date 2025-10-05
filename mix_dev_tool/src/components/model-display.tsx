import { CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { handleModelSelection } from "@/handlers/model-command-handler";
import type { ModelDisplayProps } from "@/types/provider";

export function ModelDisplay({ data }: ModelDisplayProps) {
	const handleSelect = async (modelId: string) => {
		// Call the model selection handler
		await handleModelSelection(modelId);
	};

	return (
		<Card>
			<CardContent className="p-4">
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<h3 className="font-medium text-lg">Select Model</h3>
						<span className="text-muted-foreground text-sm">
							Provider: <strong>{data.provider.displayName}</strong>
						</span>
					</div>

					{data.models.length > 0 ? (
						<div className="grid grid-cols-1 gap-2">
							{data.models.map((model) => (
								<Button
									className={`w-full justify-between ${model.isSelected ? "bg-primary text-primary-foreground hover:bg-primary/90" : ""}`}
									key={model.id}
									onClick={() => handleSelect(model.id)}
									variant={model.isSelected ? "default" : "outline"}
								>
									<span>{model.displayName}</span>
									{model.isSelected && <CheckCircle className="h-4 w-4" />}
								</Button>
							))}
						</div>
					) : (
						<div className="rounded-md bg-muted p-4 text-center">
							<p>No models available for this provider.</p>
						</div>
					)}

					{/* Help text */}
					<p className="mt-4 text-muted-foreground text-sm">
						Select a model to use with {data.provider.displayName}.
						{data.currentModel ? ` Current model: ${data.currentModel}` : ""}
					</p>
				</div>
			</CardContent>
		</Card>
	);
}
