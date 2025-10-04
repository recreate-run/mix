import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { mix } from "@/lib/mix-sdk";

interface OAuthCodeDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	provider: string;
	oauthState: string;
	onSuccess?: () => void;
}

export function OAuthCodeDialog({
	open,
	onOpenChange,
	provider,
	oauthState,
	onSuccess,
}: OAuthCodeDialogProps) {
	const [code, setCode] = useState("");
	const [isSubmitting, setIsSubmitting] = useState(false);
	const queryClient = useQueryClient();

	const handleSubmit = async () => {
		if (!code.trim()) {
			toast.error("Please enter the authorization code");
			return;
		}

		setIsSubmitting(true);
		try {
			await mix.authentication.handleOAuthCallback({
				provider,
				code: code.trim(),
				state: oauthState,
			});

			toast.success("Authentication successful!");
			queryClient.invalidateQueries({ queryKey: ["providers"] });
			queryClient.invalidateQueries({ queryKey: ["preferences"] });

			setCode("");
			onOpenChange(false);
			onSuccess?.();
		} catch (error: any) {
			console.error("OAuth code submission failed:", error);
			toast.error(error.message || "Failed to complete OAuth authentication");
		} finally {
			setIsSubmitting(false);
		}
	};

	const handleKeyDown = (e: React.KeyboardEvent) => {
		if (e.key === "Enter" && !isSubmitting) {
			handleSubmit();
		}
	};

	return (
		<Dialog onOpenChange={onOpenChange} open={open}>
			<DialogContent className="sm:max-w-[425px]">
				<DialogHeader>
					<DialogTitle>Complete OAuth Authentication</DialogTitle>
					<DialogDescription>
						After authorizing in your browser, copy and paste the authorization
						code here to complete the authentication process.
					</DialogDescription>
				</DialogHeader>

				<div className="grid gap-4 py-4">
					<div className="grid gap-2">
						<Label htmlFor="auth-code">Authorization Code</Label>
						<Input
							className="font-mono text-sm"
							disabled={isSubmitting}
							id="auth-code"
							onChange={(e) => setCode(e.target.value)}
							onKeyDown={handleKeyDown}
							placeholder="Enter the authorization code from your browser"
							value={code}
						/>
					</div>
				</div>

				<DialogFooter>
					<Button
						disabled={isSubmitting}
						onClick={() => onOpenChange(false)}
						type="button"
						variant="outline"
					>
						Cancel
					</Button>
					<Button
						disabled={isSubmitting || !code.trim()}
						onClick={handleSubmit}
						type="button"
					>
						{isSubmitting ? "Authenticating..." : "Complete Authentication"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
