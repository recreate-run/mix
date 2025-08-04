// use std::sync::{Arc, Mutex};
// use tauri::AppHandle;
// use tauri_plugin_shell::{process::CommandEvent, ShellExt};
// use tokio::time::{sleep, Duration};

// #[derive(Debug, Clone)]
// pub struct SidecarManager {
//     pub is_running: Arc<Mutex<bool>>,
//     pub child_id: Arc<Mutex<Option<u32>>>,
//     pub error_message: Arc<Mutex<Option<String>>>,
// }

// impl SidecarManager {
//     pub fn new() -> Self {
//         Self {
//             is_running: Arc::new(Mutex::new(false)),
//             child_id: Arc::new(Mutex::new(None)),
//             error_message: Arc::new(Mutex::new(None)),
//         }
//     }

//     pub async fn start_sidecar(&self, app: &AppHandle) -> Result<(), String> {
//         // Check if already running
//         if *self.is_running.lock().unwrap() {
//             return Ok(());
//         }

//         // Clear any previous error
//         *self.error_message.lock().unwrap() = None;

//         let shell = app.shell();

//         match shell.sidecar("mix") {
//             Ok(command) => {
//                 let command = command.args(["--http-mode"]);
//                 match command.spawn() {
//                     Ok((mut rx, child)) => {
//                         let child_id = child.pid();
//                         *self.child_id.lock().unwrap() = Some(child_id);
//                         *self.is_running.lock().unwrap() = true;

//                         // Spawn a task to monitor the process
//                         let is_running = Arc::clone(&self.is_running);
//                         let error_message = Arc::clone(&self.error_message);
//                         let child_id_clone = Arc::clone(&self.child_id);

//                         tokio::spawn(async move {
//                             while let Some(event) = rx.recv().await {
//                                 match event {
//                                     CommandEvent::Stdout(data) => {
//                                         println!(
//                                             "Go server stdout: {}",
//                                             String::from_utf8_lossy(&data)
//                                         );
//                                     }
//                                     CommandEvent::Stderr(data) => {
//                                         println!(
//                                             "Go server stderr: {}",
//                                             String::from_utf8_lossy(&data)
//                                         );
//                                     }
//                                     CommandEvent::Error(err) => {
//                                         *error_message.lock().unwrap() =
//                                             Some(format!("Process error: {}", err));
//                                         *is_running.lock().unwrap() = false;
//                                         *child_id_clone.lock().unwrap() = None;
//                                         break;
//                                     }
//                                     CommandEvent::Terminated(payload) => {
//                                         println!(
//                                             "Go server terminated with code: {:?}",
//                                             payload.code
//                                         );
//                                         *is_running.lock().unwrap() = false;
//                                         *child_id_clone.lock().unwrap() = None;
//                                         if payload.code != Some(0) {
//                                             *error_message.lock().unwrap() = Some(format!(
//                                                 "Process terminated with code: {:?}",
//                                                 payload.code
//                                             ));
//                                         }
//                                         break;
//                                     }
//                                     _ => {
//                                         // Handle any other variants that might exist
//                                     }
//                                 }
//                             }
//                         });

//                         // Wait a moment for the server to start
//                         sleep(Duration::from_millis(1000)).await;

//                         Ok(())
//                     }
//                     Err(e) => {
//                         let error = format!("Failed to spawn sidecar: {}", e);
//                         *self.error_message.lock().unwrap() = Some(error.clone());
//                         Err(error)
//                     }
//                 }
//             }
//             Err(e) => {
//                 let error = format!("Failed to create sidecar command: {}", e);
//                 *self.error_message.lock().unwrap() = Some(error.clone());
//                 Err(error)
//             }
//         }
//     }

//     pub async fn stop_sidecar(&self, app: &AppHandle) -> Result<(), String> {
//         if !*self.is_running.lock().unwrap() {
//             return Ok(());
//         }

//         if let Some(pid) = *self.child_id.lock().unwrap() {
//             let _shell = app.shell();

//             // Try to kill the process
//             #[cfg(unix)]
//             {
//                 use std::process::Command;
//                 match Command::new("kill").arg(pid.to_string()).output() {
//                     Ok(_) => {
//                         *self.is_running.lock().unwrap() = false;
//                         *self.child_id.lock().unwrap() = None;
//                         Ok(())
//                     }
//                     Err(e) => {
//                         let error = format!("Failed to kill process: {}", e);
//                         *self.error_message.lock().unwrap() = Some(error.clone());
//                         Err(error)
//                     }
//                 }
//             }

//             #[cfg(windows)]
//             {
//                 use std::process::Command;
//                 match Command::new("taskkill")
//                     .args(&["/F", "/PID", &pid.to_string()])
//                     .output()
//                 {
//                     Ok(_) => {
//                         *self.is_running.lock().unwrap() = false;
//                         *self.child_id.lock().unwrap() = None;
//                         Ok(())
//                     }
//                     Err(e) => {
//                         let error = format!("Failed to kill process: {}", e);
//                         *self.error_message.lock().unwrap() = Some(error.clone());
//                         Err(error)
//                     }
//                 }
//             }
//         } else {
//             Err("No process ID available".to_string())
//         }
//     }

//     pub async fn health_check(&self) -> Result<String, String> {
//         if !*self.is_running.lock().unwrap() {
//             return Err("Sidecar is not running".to_string());
//         }

//         match reqwest::get("http://localhost:8080/api/health").await {
//             Ok(response) => {
//                 if response.status().is_success() {
//                     match response.json::<serde_json::Value>().await {
//                         Ok(data) => {
//                             if let Some(status) = data.get("status").and_then(|s| s.as_str()) {
//                                 Ok(format!("Mix health check: {}", status))
//                             } else {
//                                 Ok("Mix health check successful".to_string())
//                             }
//                         }
//                         Err(e) => Err(format!("Failed to parse response: {}", e)),
//                     }
//                 } else {
//                     Err(format!(
//                         "Health check failed with status: {}",
//                         response.status()
//                     ))
//                 }
//             }
//             Err(e) => Err(format!("Health check request failed: {}", e)),
//         }
//     }

//     pub fn is_running(&self) -> bool {
//         *self.is_running.lock().unwrap()
//     }

//     pub fn get_error(&self) -> Option<String> {
//         self.error_message.lock().unwrap().clone()
//     }

//     pub async fn send_prompt(&self, prompt: &str) -> Result<String, String> {
//         if !*self.is_running.lock().unwrap() {
//             return Err("Sidecar is not running".to_string());
//         }

//         let client = reqwest::Client::new();
//         let payload = serde_json::json!({
//             "prompt": prompt
//         });

//         match client
//             .post("http://localhost:8080/api/prompt")
//             .json(&payload)
//             .send()
//             .await
//         {
//             Ok(response) => {
//                 if response.status().is_success() {
//                     match response.text().await {
//                         Ok(text) => Ok(text),
//                         Err(e) => Err(format!("Failed to read response: {}", e)),
//                     }
//                 } else {
//                     Err(format!("Request failed with status: {}", response.status()))
//                 }
//             }
//             Err(e) => Err(format!("Request failed: {}", e)),
//         }
//     }
// }
