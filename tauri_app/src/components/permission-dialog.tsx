import { Badge } from "@/components/ui/badge";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Label } from "@/components/ui/label";
import type { SSEPermissionRequest } from "@/hooks/usePersistentSSE";
import { useState } from "react";

interface PermissionDialogProps {
	permissionRequest: SSEPermissionRequest;
	onGrant: (id: string) => Promise<void>;
	onDeny: (id: string) => Promise<void>;
	onClose: () => void;
}

export function PermissionDialog({
	permissionRequest,
	onGrant,
	onDeny,
	onClose,
}: PermissionDialogProps) {
	const [isProcessing, setIsProcessing] = useState(false);

	const handleGrant = async () => {
		setIsProcessing(true);
		try {
			await onGrant(permissionRequest.id);
			onClose();
		} catch (error) {
			console.error("Failed to grant permission:", error);
		} finally {
			setIsProcessing(false);
		}
	};

	const handleDeny = async () => {
		setIsProcessing(true);
		try {
			await onDeny(permissionRequest.id);
			onClose();
		} catch (error) {
			console.error("Failed to deny permission:", error);
		} finally {
			setIsProcessing(false);
		}
	};

	return (
		<AlertDialog open={true} onOpenChange={onClose}>
			<AlertDialogContent className="sm:max-w-md">
				<AlertDialogHeader>
					<AlertDialogTitle className="flex items-center gap-2">
						Permission Required
						<Badge variant="outline">{permissionRequest.toolName}</Badge>
					</AlertDialogTitle>
					<AlertDialogDescription>
						A tool is requesting permission to access files outside your working
						directory.
					</AlertDialogDescription>
				</AlertDialogHeader>

				<Label className="text-md">Command</Label>
				<pre className="text-sm bg-muted p-3 rounded font-mono overflow-x-auto whitespace-pre-wrap">
					{permissionRequest.action.replace("Execute command: ", "")}
				</pre>

				<AlertDialogFooter className="gap-2">
					<AlertDialogCancel onClick={handleDeny} disabled={isProcessing}>
						Deny
					</AlertDialogCancel>
					<AlertDialogAction onClick={handleGrant} disabled={isProcessing}>
						{isProcessing ? "Processing..." : "Grant"}
					</AlertDialogAction>
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
