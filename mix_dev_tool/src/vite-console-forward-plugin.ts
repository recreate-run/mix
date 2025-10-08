import type { IncomingMessage } from "node:http";
import type { Plugin } from "vite";
import { createLogger } from "vite";

interface LogEntry {
	level: string;
	message: string;
	timestamp: Date;
	url?: string;
	userAgent?: string;
	stacks?: string[];
}

interface ClientLogRequest {
	logs: LogEntry[];
}

interface ConsoleForwardOptions {
	/**
	 * Whether to enable console forwarding (default: true in dev mode)
	 */
	enabled?: boolean;
	/**
	 * API endpoint path (default: '/api/debug/client-logs')
	 */
	endpoint?: string;
	/**
	 * Console levels to forward (default: ['log', 'warn', 'error', 'info', 'debug'])
	 */
	levels?: ("log" | "warn" | "error" | "info" | "debug")[];
}

const logger = createLogger("info", {
	prefix: "[browser]",
});

export function consoleForwardPlugin(
	options: ConsoleForwardOptions = {},
): Plugin {
	const {
		enabled = true,
		endpoint = "/api/debug/client-logs",
		levels = ["log", "warn", "error", "info", "debug"],
	} = options;

	const virtualModuleId = "virtual:console-forward";
	const resolvedVirtualModuleId = `\0${virtualModuleId}`;

	return {
		name: "console-forward",

		resolveId(id) {
			if (id === virtualModuleId) {
				return resolvedVirtualModuleId;
			}
		},

		transformIndexHtml: {
			order: "pre",
			handler(html) {
				if (!enabled) {
					return html;
				}

				// Check if the virtual module is already imported
				if (html.includes("virtual:console-forward")) {
					return html;
				}

				// Inject the import script in the head section
				return html.replace(
					/<head[^>]*>/i,
					(match) =>
						`${match}\n    <script type="module">import "virtual:console-forward";</script>`,
				);
			},
		},

		load(id) {
			if (id === resolvedVirtualModuleId) {
				if (!enabled) {
					return "export default {};";
				}

				// Create the console forwarding code
				return `
// Console forwarding module
const originalMethods = {
  log: console.log.bind(console),
  warn: console.warn.bind(console),
  error: console.error.bind(console),
  info: console.info.bind(console),
  debug: console.debug.bind(console),
};

const logBuffer = [];
let flushTimeout = null;
const FLUSH_DELAY = 100;
const MAX_BUFFER_SIZE = 50;

function createLogEntry(level, args) {
  const stacks = [];
  
  const message = args.map((arg) => {
    if (arg === undefined) return "undefined";
    if (arg === null) return "null";
    if (typeof arg === "string") return arg;
    if (typeof arg === "number" || typeof arg === "boolean") return String(arg);
    if (arg instanceof Error || typeof arg.stack === "string") {
      let stringifiedError = arg.toString();
      if (arg.stack) {
        let stack = arg.stack.toString();
        if (stack.startsWith(stringifiedError)) {
          stack = stack.slice(stringifiedError.length).trimStart();
        }
        if (stack) {
          stacks.push(stack);
        }
      }
      return stringifiedError;
    }
    if (typeof arg === "object" && arg !== null) {
      try {
        return JSON.stringify(arg, null, 2);
      } catch (e) {
        // Fallback for objects that can't be stringified (circular refs, etc.)
        return String(arg);
      }
    }
    return String(arg);
  }).join(" ");
  
  return {
    level,
    message,
    timestamp: new Date(),
    url: window.location.href,
    userAgent: navigator.userAgent,
    stacks,
  };
}

async function sendLogs(logs) {
  try {
    await fetch("${endpoint}", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ logs }),
    });
  } catch (error) {
    // Fail silently in production
  }
}

function flushLogs() {
  if (logBuffer.length === 0) return;
  const logsToSend = [...logBuffer];
  logBuffer.length = 0;
  sendLogs(logsToSend);
  if (flushTimeout) {
    clearTimeout(flushTimeout);
    flushTimeout = null;
  }
}

function addToBuffer(entry) {
  logBuffer.push(entry);
  if (logBuffer.length >= MAX_BUFFER_SIZE) {
    flushLogs();
    return;
  }
  if (!flushTimeout) {
    flushTimeout = setTimeout(flushLogs, FLUSH_DELAY);
  }
}

// Patch console methods
${levels
	.map(
		(level) => `
console.${level} = function(...args) {
  originalMethods.${level}(...args);
  const entry = createLogEntry("${level}", args);
  addToBuffer(entry);
};`,
	)
	.join("")}

// Cleanup handlers
window.addEventListener("beforeunload", flushLogs);
setInterval(flushLogs, 10000);

export default { flushLogs };
        `;
			}
		},
		configureServer(server) {
			// Add API endpoint to handle forwarded console logs
			server.middlewares.use(endpoint, (req, res, next) => {
				const request = req as IncomingMessage & { method?: string };
				if (request.method !== "POST") {
					return next();
				}

				let body = "";
				request.setEncoding?.("utf8");

				request.on("data", (chunk: string) => {
					body += chunk;
				});

				request.on("end", () => {
					try {
						const { logs }: ClientLogRequest = JSON.parse(body);

						// Forward each log to the Vite dev server console using Vite's logger
						logs.forEach((log) => {
							const location = log.url ? ` (${log.url})` : "";
							let message = `[${log.level}] ${log.message}${location}`;

							// Add stack traces if available
							if (log.stacks && log.stacks.length > 0) {
								message +=
									"\n" +
									log.stacks
										.map((stack) =>
											stack
												.split("\n")
												.map((line) => `    ${line}`)
												.join("\n"),
										)
										.join("\n");
							}

							// Use Vite's logger for consistent formatting
							const logOptions = { timestamp: false };
							switch (log.level) {
								case "error": {
									const error =
										log.stacks && log.stacks.length > 0
											? new Error(log.stacks.join("\n"))
											: null;
									logger.error(message, { ...logOptions, error });
									break;
								}
								case "warn":
									logger.warn(message, logOptions);
									break;
								case "info":
									logger.info(message, logOptions);
									break;
								case "debug":
									logger.info(message, logOptions);
									break;
								default:
									logger.info(message, logOptions);
							}
						});

						res.writeHead(200, { "Content-Type": "application/json" });
						res.end(JSON.stringify({ success: true }));
					} catch (error) {
						server.config.logger.error("Error processing client logs:", {
							timestamp: true,
							error: error as Error,
						});
						res.writeHead(400, { "Content-Type": "application/json" });
						res.end(JSON.stringify({ error: "Invalid JSON" }));
					}
				});
			});
		},
	};
}
