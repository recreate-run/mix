import {
	AlertCircle,
	AlertTriangle,
	HelpCircle,
	Info,
	Timer,
} from "lucide-react";
import type { Type as NotificationResponseType } from "mix-typescript-sdk/models/operations/respondtonotification";
import { useEffect, useId, useState } from "react";
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
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import type { SSENotificationRequest } from "@/hooks/usePersistentSSE";

interface NotificationDialogProps {
	notification: SSENotificationRequest;
	onRespond: (
		id: string,
		response: { type: NotificationResponseType; value?: string },
	) => Promise<void>;
}

export function NotificationDialog({
	notification,
	onRespond,
}: NotificationDialogProps) {
	const [isProcessing, setIsProcessing] = useState(false);
	const textInputId = useId();
	const [textValue, setTextValue] = useState("");
	const [choiceValue, setChoiceValue] = useState("");
	const [remainingTime, setRemainingTime] = useState(notification.timeout);

	// Countdown timer
	useEffect(() => {
		const timer = setInterval(() => {
			setRemainingTime((prev) => {
				if (prev <= 1) {
					clearInterval(timer);
					return 0;
				}
				return prev - 1;
			});
		}, 1000);

		return () => clearInterval(timer);
	}, []);

	// Auto-close on timeout
	useEffect(() => {
		if (remainingTime === 0) {
			// Notification will timeout on backend, just close the dialog
			handleClose();
		}
	}, [remainingTime]);

	const handleClose = () => {
		// Dialog closes, backend will handle timeout
	};

	const handleRespond = async () => {
		setIsProcessing(true);
		try {
			const response: { type: NotificationResponseType; value?: string } = {
				type: notification.responseType as NotificationResponseType,
			};

			if (notification.responseType === "text") {
				response.value = textValue;
			} else if (notification.responseType === "choice") {
				response.value = choiceValue;
			}

			await onRespond(notification.id, response);
		} catch (error) {
			console.error("Failed to respond to notification:", error);
		} finally {
			setIsProcessing(false);
		}
	};

	// Get icon and color based on notification type
	const getTypeConfig = () => {
		switch (notification.type) {
			case "info":
				return {
					icon: <Info className="h-5 w-5 text-blue-500" />,
					badgeVariant: "default" as const,
					color: "text-blue-500",
				};
			case "warning":
				return {
					icon: <AlertTriangle className="h-5 w-5 text-yellow-500" />,
					badgeVariant: "secondary" as const,
					color: "text-yellow-500",
				};
			case "error":
				return {
					icon: <AlertCircle className="h-5 w-5 text-red-500" />,
					badgeVariant: "destructive" as const,
					color: "text-red-500",
				};
			case "question":
				return {
					icon: <HelpCircle className="h-5 w-5 text-purple-500" />,
					badgeVariant: "outline" as const,
					color: "text-purple-500",
				};
			default:
				return {
					icon: <Info className="h-5 w-5" />,
					badgeVariant: "default" as const,
					color: "",
				};
		}
	};

	const typeConfig = getTypeConfig();
	const canSubmit =
		notification.responseType === "acknowledge" ||
		(notification.responseType === "text" && textValue.trim().length > 0) ||
		(notification.responseType === "choice" && choiceValue !== "");

	return (
		<AlertDialog onOpenChange={handleClose} open={true}>
			<AlertDialogContent className="sm:max-w-md">
				<AlertDialogHeader>
					<AlertDialogTitle className="flex items-center gap-2">
						{typeConfig.icon}
						{notification.title}
						<Badge variant={typeConfig.badgeVariant}>{notification.type}</Badge>
					</AlertDialogTitle>
					<AlertDialogDescription className="flex items-center justify-between">
						<span>{notification.message}</span>
						<span className="flex items-center gap-1 text-xs text-muted-foreground">
							<Timer className="h-3 w-3" />
							{remainingTime}s
						</span>
					</AlertDialogDescription>
				</AlertDialogHeader>

				{/* Dynamic content based on response type */}
				<div className="space-y-4">
					{notification.responseType === "text" && (
						<div className="space-y-2">
							<Label htmlFor={textInputId}>Your Response</Label>
							<Input
								id={textInputId}
								value={textValue}
								onChange={(e) => setTextValue(e.target.value)}
								placeholder="Enter your response..."
								autoFocus
								onKeyDown={(e) => {
									if (e.key === "Enter" && canSubmit && !isProcessing) {
										handleRespond();
									}
								}}
							/>
						</div>
					)}

					{notification.responseType === "choice" && notification.choices && (
						<div className="space-y-2">
							<Label>Select an Option</Label>
							<RadioGroup value={choiceValue} onValueChange={setChoiceValue}>
								{notification.choices.map((choice) => (
									<div key={choice} className="flex items-center space-x-2">
										<RadioGroupItem value={choice} id={choice} />
										<Label htmlFor={choice} className="cursor-pointer">
											{choice}
										</Label>
									</div>
								))}
							</RadioGroup>
						</div>
					)}
				</div>

				<AlertDialogFooter className="gap-2">
					{notification.responseType === "acknowledge" ? (
						<AlertDialogAction
							disabled={isProcessing || remainingTime === 0}
							onClick={handleRespond}
						>
							{isProcessing ? "Processing..." : "OK"}
						</AlertDialogAction>
					) : (
						<>
							<AlertDialogCancel disabled={isProcessing}>
								Cancel
							</AlertDialogCancel>
							<AlertDialogAction
								disabled={!canSubmit || isProcessing || remainingTime === 0}
								onClick={handleRespond}
							>
								{isProcessing ? "Submitting..." : "Submit"}
							</AlertDialogAction>
						</>
					)}
				</AlertDialogFooter>
			</AlertDialogContent>
		</AlertDialog>
	);
}
