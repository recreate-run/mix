import { useEffect, useState } from "react";
import { DotFlow } from "./DotFlowCSS";

// Commands for the opening screen animation - Mix-specific workflow examples
const commands = [
	{
		title: "Create a marketing video from these screenshots",
		frames: [
			[21, 22, 23, 24, 25, 26, 27], // Top row formation
			[14, 15, 16, 28, 29, 30, 34, 35], // Expanding down
			[7, 8, 9, 21, 22, 23, 37, 38, 39], // Cross pattern
			[0, 1, 2, 14, 15, 16, 28, 29, 30, 42, 43, 44], // Full grid formation
			[7, 8, 9, 21, 22, 23, 35, 36, 37], // Contracting
			[14, 15, 16, 28, 29, 30, 34], // Final form
			[21, 22, 23, 24, 25, 26, 27], // Return to top
		],
		duration: 140,
		repeatCount: 1,
	},
	{
		title: "Analyze this user session recording",
		frames: [
			[0, 7, 14, 21, 28, 35, 42], // Scanning vertically
			[1, 8, 15, 22, 29, 36, 43],
			[2, 9, 16, 23, 30, 37, 44],
			[3, 10, 17, 24, 31, 38, 45],
			[4, 11, 18, 25, 32, 39, 46],
			[5, 12, 19, 26, 33, 40, 47],
			[6, 13, 20, 27, 34, 41, 48],
			[10, 11, 12, 17, 18, 19, 24, 25, 26, 31, 32, 33], // Highlighting center analysis
		],
		duration: 110,
		repeatCount: 1,
	},
	{
		title: "Edit this video: trim and add title overlay",
		frames: [
			[0, 1, 2, 3, 4, 5, 6], // Timeline frames
			[7, 8, 9, 10, 11, 12, 13],
			[14, 15, 16, 17, 18, 19, 20],
			[21, 22, 23, 24, 25, 26, 27], // Middle section highlight
			[14, 15, 16, 17, 18, 19, 20], // Trim back
			[7, 8, 9, 10, 11, 12, 13],
			[21, 22, 23, 24, 25, 26, 27], // Final edited section
		],
		duration: 160,
		repeatCount: 1,
	},
	{
		title: "Generate storyboard frames for concept",
		frames: [
			[10, 17, 24, 31], // Four corner frames
			[9, 10, 11, 16, 17, 18, 23, 24, 25, 30, 31, 32], // Expanding frames
			[2, 3, 4, 9, 10, 11, 16, 17, 18, 23, 24, 25, 30, 31, 32, 37, 38, 39], // Full storyboard
			[10, 11, 17, 18, 24, 25, 31, 32], // Refined frames
			[17, 24], // Key frames
			[10, 17, 24, 31], // Final four frames
		],
		duration: 180,
		repeatCount: 1,
	},
	{
		title: "Process batch images: resize and watermark",
		frames: [
			[0, 2, 4, 6], // Scattered images
			[7, 9, 11, 13],
			[14, 16, 18, 20],
			[21, 23, 25, 27], // Processing wave
			[28, 30, 32, 34],
			[35, 37, 39, 41],
			[42, 44, 46, 48],
			[0, 2, 4, 6, 42, 44, 46, 48], // Before and after
		],
		duration: 130,
		repeatCount: 1,
	},
];

interface LoadingScreenProps {
	duration?: number;
	onComplete?: () => void;
}

export function LoadingScreen({
	duration = 10,
	onComplete,
}: LoadingScreenProps) {
	const [logoVisible, setLogoVisible] = useState(false);
	const [animationStarted, setAnimationStarted] = useState(false);

	useEffect(() => {
		// Start animations immediately
		setAnimationStarted(true);

		// Logo fade in after 100ms
		const logoTimer = setTimeout(() => {
			setLogoVisible(true);
		}, 100);

		// Complete after duration
		const completeTimer = setTimeout(() => {
			if (onComplete) onComplete();
		}, duration * 1000);

		return () => {
			clearTimeout(logoTimer);
			clearTimeout(completeTimer);
		};
	}, [duration, onComplete]);

	return (
		<div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-background to-background/90">
			<div
				className={`rounded-lg bg-background/50 p-8 text-center backdrop-blur-sm transition-transform duration-3000 ease-in-out ${animationStarted ? "animate-float" : ""}`}
			>
				<div className="relative mb-4">
					<img
						alt="Mix Logo"
						className={`mx-auto size-48 object-contain transition-all duration-2000 ease-out ${
							logoVisible
								? "scale-100 animate-pulse-subtle opacity-100"
								: "scale-75 opacity-0"
						}`}
						src="/mix_logo_transparent.png"
					/>
					<div className="flex items-center justify-center text-muted-foreground text-sm">
						<span>The multimodal agents SDK</span>
						<span className="ml-1 inline-block h-4 w-0.5 animate-blink bg-primary" />
					</div>
				</div>

				{/* Animated Commands with enhanced styling */}
				<div className="slide-in-from-bottom relative animate-in delay-500 duration-1000">
					<div className="relative rounded-xl border border-border/50 bg-card/50 p-6 shadow-2xl backdrop-blur-sm [&_.dot-flow-container]:text-foreground [&_.dot-loader_.active]:bg-primary [&_.dot-loader_.h-1\\.5]:bg-muted/30">
						<DotFlow isPlaying={true} items={commands} />
					</div>
				</div>
			</div>

			<style
				dangerouslySetInnerHTML={{
					__html: `
          @keyframes float {
            0%, 100% { transform: translateY(0px); }
            50% { transform: translateY(8px); }
          }

          @keyframes pulse-subtle {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.05); }
          }

          @keyframes blink {
            0%, 100% { opacity: 1; }
            50% { opacity: 0; }
          }

          .animate-float {
            animation: float 3s ease-in-out infinite;
          }

          .animate-pulse-subtle {
            animation: pulse-subtle 1.5s ease-in-out 2;
            animation-delay: 1s;
          }

          .animate-blink {
            animation: blink 0.7s ease-in-out infinite;
          }
        `,
				}}
			/>
		</div>
	);
}
