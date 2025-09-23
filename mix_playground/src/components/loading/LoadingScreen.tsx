import { useEffect, useRef, useState } from 'react';
import { gsap } from 'gsap';
import { DotFlow } from '../../../components/gsap/dot-flow';

// Commands for the opening screen animation - Mix-specific creative workflow examples
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

export function LoadingScreen({ duration = 10, onComplete }: LoadingScreenProps) {
  // Ensure onComplete is called after duration
  useEffect(() => {
    const timer = setTimeout(() => {
      if (onComplete) onComplete();
    }, duration * 1000);
    
    return () => clearTimeout(timer);
  }, [duration, onComplete]);
  const logoRef = useRef(null);
  const containerRef = useRef(null);
  const cursorRef = useRef(null);
  const textRef = useRef(null);
  const [animationComplete, setAnimationComplete] = useState(false);
  
  useEffect(() => {
    const tl = gsap.timeline({ 
      onComplete: () => {
        setAnimationComplete(true);
        if (onComplete) onComplete();
      }
    });
    
    // Force the animation to take exactly the specified duration
    tl.totalDuration(duration);
    
    // Initial fade in
    tl.fromTo(logoRef.current,
      { opacity: 0, scale: 0.8 },
      { opacity: 1, scale: 1, duration: 2, ease: "power2.out" }
    );
    
    // Subtle pulse animation for the logo
    tl.to(logoRef.current, {
      scale: 1.05, 
      duration: 1.5, 
      repeat: 2, 
      yoyo: true,
      ease: "sine.inOut"
    }, ">-0.5");
    
    // Fade in the tagline text
    tl.fromTo(textRef.current,
      { opacity: 0 },
      { opacity: 1, duration: 1.5, ease: "power1.inOut" },
      ">-1"
    );
    
    // Continuous animations independent of the timeline
    
    // Animate the cursor blinking
    gsap.to(cursorRef.current, {
      opacity: 0,
      duration: 0.7,
      repeat: -1,
      yoyo: true,
      ease: "power2.inOut"
    });

    // Subtle container floating animation
    gsap.to(containerRef.current, {
      y: 8,
      duration: 3,
      repeat: -1,
      yoyo: true,
      ease: "sine.inOut"
    });
    
    return () => {
      tl.kill();
      gsap.killTweensOf([logoRef.current, cursorRef.current, containerRef.current]);
    };
  }, [duration, onComplete]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-background to-background/90">
      <div ref={containerRef} className="text-center p-8 rounded-lg bg-background/50 backdrop-blur-sm">
        <div className="mb-4 relative">
          <div ref={logoRef} className="text-4xl font-bold text-primary mb-2">mix</div>
          <div className="text-sm text-muted-foreground flex justify-center items-center">
            <span>AI-powered creative workflows</span>
            <span ref={cursorRef} className="ml-1 h-4 w-0.5 bg-primary inline-block"></span>
          </div>
        </div>
        
        {/* Animated Commands with enhanced styling */}
        <div className="relative animate-in slide-in-from-bottom duration-1000 delay-500">
          <div className="absolute -inset-4 bg-gradient-to-r from-blue-500/10 via-purple-500/10 to-pink-500/10 rounded-xl blur-xl"></div>
          <div className="relative bg-card/50 backdrop-blur-sm border border-border/50 rounded-xl p-6 shadow-2xl [&_.dot-flow-container]:text-foreground [&_.dot-loader_.h-1\\.5]:bg-muted/30 [&_.dot-loader_.active]:bg-primary">
            <DotFlow items={commands} isPlaying={true} />
          </div>
        </div>

        {/* Loading text */}
        <p ref={textRef} className="text-sm text-muted-foreground/80 animate-in slide-in-from-bottom duration-1000 delay-700 mt-4">Loading your creative space...</p>
      </div>
    </div>
  );
}