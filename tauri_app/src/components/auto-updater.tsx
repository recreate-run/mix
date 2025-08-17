import { useEffect, useState } from "react";
import { check, type Update } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";
import { Button } from "./ui/button";
import { Progress } from "./ui/progress";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "./ui/card";
import { Download, RefreshCw } from "lucide-react";

export function AutoUpdater() {
	const [update, setUpdate] = useState<Update | null>(null);
	const [isChecking, setIsChecking] = useState(false);
	const [isDownloading, setIsDownloading] = useState(false);
	const [downloadProgress, setDownloadProgress] = useState(0);
	const [error, setError] = useState<string | null>(null);

	const checkForUpdates = async () => {
		setIsChecking(true);
		setError(null);

		try {
			const updateInfo = await check();
			setUpdate(updateInfo);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to check for updates",
			);
		} finally {
			setIsChecking(false);
		}
	};

	const downloadAndInstall = async () => {
		if (!update) return;

		setIsDownloading(true);
		setDownloadProgress(0);

		try {
			let downloaded = 0;
			let contentLength = 0;

			await update.downloadAndInstall((event) => {
				switch (event.event) {
					case "Started":
						contentLength = event.data.contentLength || 0;
						break;
					case "Progress":
						downloaded += event.data.chunkLength;
						if (contentLength > 0) {
							setDownloadProgress((downloaded / contentLength) * 100);
						}
						break;
					case "Finished":
						setDownloadProgress(100);
						break;
				}
			});

			// Restart the app after successful installation
			await relaunch();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to download update",
			);
		} finally {
			setIsDownloading(false);
		}
	};

	// Check for updates on component mount
	useEffect(() => {
		checkForUpdates();
	}, []);

	if (error) {
		return (
			<Card className="max-w-md mx-auto">
				<CardHeader>
					<CardTitle className="text-red-600">Update Error</CardTitle>
				</CardHeader>
				<CardContent>
					<p className="text-sm text-red-600 mb-4">{error}</p>
					<Button onClick={checkForUpdates} disabled={isChecking}>
						<RefreshCw
							className={`h-4 w-4 mr-2 ${isChecking ? "animate-spin" : ""}`}
						/>
						Try Again
					</Button>
				</CardContent>
			</Card>
		);
	}

	if (!update) {
		return (
			<Card className="max-w-md mx-auto">
				<CardHeader>
					<CardTitle>App Updates</CardTitle>
					<CardDescription>
						{isChecking
							? "Checking for updates..."
							: "You have the latest version"}
					</CardDescription>
				</CardHeader>
				<CardContent>
					<Button onClick={checkForUpdates} disabled={isChecking}>
						<RefreshCw
							className={`h-4 w-4 mr-2 ${isChecking ? "animate-spin" : ""}`}
						/>
						Check for Updates
					</Button>
				</CardContent>
			</Card>
		);
	}

	return (
		<Card className="max-w-md mx-auto">
			<CardHeader>
				<CardTitle>Update Available</CardTitle>
				<CardDescription>
					Version {update.version} is ready to install
				</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				{update.body && (
					<div className="text-sm bg-gray-50 p-3 rounded">
						<strong>Release Notes:</strong>
						<div className="mt-1 whitespace-pre-wrap">{update.body}</div>
					</div>
				)}

				{isDownloading && (
					<div className="space-y-2">
						<div className="text-sm text-gray-600">
							Downloading update... {Math.round(downloadProgress)}%
						</div>
						<Progress value={downloadProgress} />
					</div>
				)}

				<div className="flex gap-2">
					<Button
						onClick={downloadAndInstall}
						disabled={isDownloading}
						className="flex-1"
					>
						<Download className="h-4 w-4 mr-2" />
						{isDownloading ? "Installing..." : "Download & Install"}
					</Button>
					<Button
						variant="outline"
						onClick={() => setUpdate(null)}
						disabled={isDownloading}
					>
						Later
					</Button>
				</div>
			</CardContent>
		</Card>
	);
}
