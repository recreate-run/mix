import { relaunch } from '@tauri-apps/plugin-process';
import { check, type Update } from '@tauri-apps/plugin-updater';
import { Download, RefreshCw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Button } from './ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from './ui/card';
import { Progress } from './ui/progress';

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
        err instanceof Error ? err.message : 'Failed to check for updates'
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
          case 'Started':
            contentLength = event.data.contentLength || 0;
            break;
          case 'Progress':
            downloaded += event.data.chunkLength;
            if (contentLength > 0) {
              setDownloadProgress((downloaded / contentLength) * 100);
            }
            break;
          case 'Finished':
            setDownloadProgress(100);
            break;
        }
      });

      // Restart the app after successful installation
      await relaunch();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to download update'
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
      <Card className="mx-auto max-w-md">
        <CardHeader>
          <CardTitle className="text-red-600">Update Error</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="mb-4 text-red-600 text-sm">{error}</p>
          <Button disabled={isChecking} onClick={checkForUpdates}>
            <RefreshCw
              className={`mr-2 h-4 w-4 ${isChecking ? 'animate-spin' : ''}`}
            />
            Try Again
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (!update) {
    return (
      <Card className="mx-auto max-w-md">
        <CardHeader>
          <CardTitle>App Updates</CardTitle>
          <CardDescription>
            {isChecking
              ? 'Checking for updates...'
              : 'You have the latest version'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button disabled={isChecking} onClick={checkForUpdates}>
            <RefreshCw
              className={`mr-2 h-4 w-4 ${isChecking ? 'animate-spin' : ''}`}
            />
            Check for Updates
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mx-auto max-w-md">
      <CardHeader>
        <CardTitle>Update Available</CardTitle>
        <CardDescription>
          Version {update.version} is ready to install
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {update.body && (
          <div className="rounded bg-gray-50 p-3 text-sm">
            <strong>Release Notes:</strong>
            <div className="mt-1 whitespace-pre-wrap">{update.body}</div>
          </div>
        )}

        {isDownloading && (
          <div className="space-y-2">
            <div className="text-gray-600 text-sm">
              Downloading update... {Math.round(downloadProgress)}%
            </div>
            <Progress value={downloadProgress} />
          </div>
        )}

        <div className="flex gap-2">
          <Button
            className="flex-1"
            disabled={isDownloading}
            onClick={downloadAndInstall}
          >
            <Download className="mr-2 h-4 w-4" />
            {isDownloading ? 'Installing...' : 'Download & Install'}
          </Button>
          <Button
            disabled={isDownloading}
            onClick={() => setUpdate(null)}
            variant="outline"
          >
            Later
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
