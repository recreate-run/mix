use std::sync::{Arc, Mutex};
use std::env;
use tauri::AppHandle;
use tauri_plugin_shell::{process::{CommandEvent, CommandChild}, ShellExt};
use tokio::time::{sleep, Duration};

#[derive(Debug)]
pub struct SidecarManager {
    child: Arc<Mutex<Option<CommandChild>>>,
    error_message: Arc<Mutex<Option<String>>>,
}

impl Drop for SidecarManager {
    fn drop(&mut self) {
        if let Ok(mut child_guard) = self.child.lock() {
            if let Some(child) = child_guard.take() {
                let _ = child.kill();
                println!("Sidecar process terminated during cleanup");
            }
        }
    }
}

impl SidecarManager {
    pub fn new() -> Self {
        Self {
            child: Arc::new(Mutex::new(None)),
            error_message: Arc::new(Mutex::new(None)),
        }
    }

    pub async fn start_sidecar(&self, app: &AppHandle) -> Result<(), String> {
        // Check if already running
        if self.child.lock().unwrap().is_some() {
            return Ok(());
        }

        // Clear any previous error
        *self.error_message.lock().unwrap() = None;

        // Get sidecar name from environment variable (defaults to "mix")
        let sidecar_name = env::var("SIDECAR_NAME")
            .unwrap_or_else(|_| "mix".to_string());

        // Create sidecar command
        let sidecar_command = match app.shell().sidecar(&sidecar_name) {
            Ok(cmd) => cmd,
            Err(e) => {
                let error = format!("Failed to create sidecar command '{}': {}", sidecar_name, e);
                *self.error_message.lock().unwrap() = Some(error.clone());
                return Err(error);
            }
        };

        println!("Starting sidecar '{}' with args: -c /Users/sarathmenon/Documents/startup/image_generation/mix --http-port 8088 --dangerously-skip-permissions -d", sidecar_name);
        let command = sidecar_command.args(["-c", "/Users/sarathmenon/Documents/startup/image_generation/mix", "--http-port", "8088", "--dangerously-skip-permissions", "-d"]);
        
        match command.spawn() {
            Ok((mut rx, child)) => {
                // Store the child process
                *self.child.lock().unwrap() = Some(child);

                // Simple monitoring task for logging
                let error_message = Arc::clone(&self.error_message);
                let child_ref = Arc::clone(&self.child);

                tokio::spawn(async move {
                    while let Some(event) = rx.recv().await {
                        match event {
                            CommandEvent::Stdout(data) => {
                                println!("Go server stdout: {}", String::from_utf8_lossy(&data));
                            }
                            CommandEvent::Stderr(data) => {
                                println!("Go server stderr: {}", String::from_utf8_lossy(&data));
                            }
                            CommandEvent::Error(err) => {
                                *error_message.lock().unwrap() = Some(format!("Process error: {}", err));
                                *child_ref.lock().unwrap() = None;
                                break;
                            }
                            CommandEvent::Terminated(payload) => {
                                println!("Go server terminated with code: {:?}", payload.code);
                                *child_ref.lock().unwrap() = None;
                                if payload.code != Some(0) {
                                    *error_message.lock().unwrap() = Some(format!(
                                        "Process terminated with code: {:?}",
                                        payload.code
                                    ));
                                }
                                break;
                            }
                            _ => {}
                        }
                    }
                });

                // Wait a moment for the server to start
                sleep(Duration::from_millis(1000)).await;
                Ok(())
            }
            Err(e) => {
                let error = format!("Failed to spawn sidecar: {}", e);
                *self.error_message.lock().unwrap() = Some(error.clone());
                Err(error)
            }
        }
    }

    pub async fn stop_sidecar(&self, _app: &AppHandle) -> Result<(), String> {
        if let Some(child) = self.child.lock().unwrap().take() {
            match child.kill() {
                Ok(_) => {
                    println!("Sidecar process stopped successfully");
                    Ok(())
                }
                Err(e) => {
                    let error = format!("Failed to kill sidecar process: {}", e);
                    *self.error_message.lock().unwrap() = Some(error.clone());
                    Err(error)
                }
            }
        } else {
            Ok(()) // Already stopped
        }
    }

    pub async fn health_check(&self) -> Result<String, String> {
        if self.child.lock().unwrap().is_none() {
            return Err("Sidecar is not running".to_string());
        }

        match reqwest::get("http://localhost:8088/api/health").await {
            Ok(response) => {
                if response.status().is_success() {
                    match response.json::<serde_json::Value>().await {
                        Ok(data) => {
                            if let Some(status) = data.get("status").and_then(|s| s.as_str()) {
                                Ok(format!("Mix health check: {}", status))
                            } else {
                                Ok("Mix health check successful".to_string())
                            }
                        }
                        Err(e) => Err(format!("Failed to parse response: {}", e)),
                    }
                } else {
                    Err(format!(
                        "Health check failed with status: {}",
                        response.status()
                    ))
                }
            }
            Err(e) => Err(format!("Health check request failed: {}", e)),
        }
    }

    pub fn is_running(&self) -> bool {
        self.child.lock().unwrap().is_some()
    }

    pub fn get_error(&self) -> Option<String> {
        self.error_message.lock().unwrap().clone()
    }

    pub async fn send_prompt(&self, prompt: &str) -> Result<String, String> {
        if self.child.lock().unwrap().is_none() {
            return Err("Sidecar is not running".to_string());
        }

        let client = reqwest::Client::new();
        let payload = serde_json::json!({
            "prompt": prompt
        });

        match client
            .post("http://localhost:8088/api/prompt")
            .json(&payload)
            .send()
            .await
        {
            Ok(response) => {
                if response.status().is_success() {
                    match response.text().await {
                        Ok(text) => Ok(text),
                        Err(e) => Err(format!("Failed to read response: {}", e)),
                    }
                } else {
                    Err(format!("Request failed with status: {}", response.status()))
                }
            }
            Err(e) => Err(format!("Request failed: {}", e)),
        }
    }
}