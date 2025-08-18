mod sidecar;
use sidecar::SidecarManager;

use objc2_app_kit::{NSColor, NSWindow};
use objc2::ffi::nil;
use objc2::runtime::AnyObject;
use window_vibrancy::{apply_vibrancy, NSVisualEffectMaterial};

#[cfg(target_os = "macos")]
use objc2_app_kit::{NSWorkspace, NSBitmapImageRep};
#[cfg(target_os = "macos")]
use objc2::{msg_send, ClassType};
#[cfg(target_os = "macos")]
use objc2::rc::autoreleasepool;
#[cfg(target_os = "macos")]
use objc2_foundation::ns_string;
#[cfg(target_os = "macos")]
use block2::StackBlock;
#[cfg(target_os = "macos")]
use std::ptr::NonNull;
#[cfg(target_os = "macos")]
use std::ffi::CStr;
#[cfg(target_os = "macos")]
use base64::engine::general_purpose;
#[cfg(target_os = "macos")]
use base64::Engine;

// use tauri::menu::{Menu, MenuItem};
// use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{Emitter, Manager, State};
use std::sync::Arc;
use std::env;

#[cfg(desktop)]
use tauri_plugin_global_shortcut::{Code, GlobalShortcutExt, Modifiers, Shortcut, ShortcutState};

// Learn more about Tauri commands at https://tauri.app/develop/calling-rust/

#[cfg(target_os = "macos")]
#[derive(serde::Serialize, Clone)]
struct AppMetadata {
    name: String,
    bundle_id: String,
    path: String,
}

#[cfg(target_os = "macos")]
#[tauri::command]
async fn get_app_list() -> Result<Vec<AppMetadata>, String> {
    unsafe {
        let workspace = NSWorkspace::sharedWorkspace();
        let apps = workspace.runningApplications();
        let mut result = Vec::with_capacity(apps.len());

        for app in apps.iter() {
            // Check if app is visible (not background-only)
            let is_hidden: bool = msg_send![&*app, isHidden];
            let activation_policy: i64 = msg_send![&*app, activationPolicy];
            
            // Only include regular GUI apps
            if is_hidden || activation_policy != 0 {
                continue;
            }

            // Get app name
            let name_ns: *mut AnyObject = msg_send![&*app, localizedName];
            if name_ns == nil {
                continue;
            }
            let utf8_ptr: *const std::os::raw::c_char = msg_send![name_ns, UTF8String];
            let name = CStr::from_ptr(utf8_ptr).to_string_lossy().into_owned();

            if name.is_empty() {
                continue;
            }

            // Get bundle identifier
            let bundle_id_ns: *mut AnyObject = msg_send![&*app, bundleIdentifier];
            let bundle_id = if bundle_id_ns != nil {
                let utf8_ptr: *const std::os::raw::c_char = msg_send![bundle_id_ns, UTF8String];
                CStr::from_ptr(utf8_ptr).to_string_lossy().into_owned()
            } else {
                // Fallback to process identifier if no bundle ID
                let pid: i32 = msg_send![&*app, processIdentifier];
                format!("pid_{}", pid)
            };

            // Get app path
            let bundle_url: *mut AnyObject = msg_send![&*app, bundleURL];
            let path = if bundle_url != nil {
                let path_ns: *mut AnyObject = msg_send![bundle_url, path];
                if path_ns != nil {
                    let utf8_ptr: *const std::os::raw::c_char = msg_send![path_ns, UTF8String];
                    CStr::from_ptr(utf8_ptr).to_string_lossy().into_owned()
                } else {
                    String::new()
                }
            } else {
                String::new()
            };

            result.push(AppMetadata {
                name,
                bundle_id,
                path,
            });
        }

        Ok(result)
    }
}

#[cfg(target_os = "macos")]
#[tauri::command]
async fn get_app_icon(bundle_id: String) -> Result<String, String> {
    unsafe {
        let workspace = NSWorkspace::sharedWorkspace();
        let apps = workspace.runningApplications();

        for app in apps.iter() {
            // Get bundle identifier for comparison
            let app_bundle_id_ns: *mut AnyObject = msg_send![&*app, bundleIdentifier];
            let app_bundle_id = if app_bundle_id_ns != nil {
                let utf8_ptr: *const std::os::raw::c_char = msg_send![app_bundle_id_ns, UTF8String];
                CStr::from_ptr(utf8_ptr).to_string_lossy().into_owned()
            } else {
                // Fallback to process identifier
                let pid: i32 = msg_send![&*app, processIdentifier];
                format!("pid_{}", pid)
            };

            if app_bundle_id != bundle_id {
                continue;
            }

            // Get app icon
            let icon: *mut AnyObject = msg_send![&*app, icon];
            if icon == nil {
                return Err("No icon available".to_string());
            }

            // Convert icon to PNG data
            let tiff_data: *mut AnyObject = msg_send![icon, TIFFRepresentation];
            if tiff_data == nil {
                return Err("Failed to get TIFF representation".to_string());
            }

            // Create bitmap representation from TIFF data
            let bitmap_rep: *mut AnyObject = msg_send![NSBitmapImageRep::class(), alloc];
            let bitmap_rep: *mut AnyObject = msg_send![bitmap_rep, initWithData: tiff_data];
            if bitmap_rep == nil {
                return Err("Failed to create bitmap representation".to_string());
            }

            // Convert to PNG data (NSBitmapImageFileTypePNG = 4)
            let png_data: *mut AnyObject = msg_send![bitmap_rep, representationUsingType: 4u64, properties: nil];
            if png_data == nil {
                return Err("Failed to convert to PNG".to_string());
            }

            // Extract bytes and base64-encode
            let bytes: *const u8 = msg_send![png_data, bytes];
            let len: usize = msg_send![png_data, length];
            let slice = std::slice::from_raw_parts(bytes, len);
            let b64 = general_purpose::STANDARD.encode(slice);

            return Ok(b64);
        }

        Err("App not found".to_string())
    }
}

#[cfg(not(target_os = "macos"))]
#[derive(serde::Serialize, Clone)]
struct AppMetadata {
    name: String,
    bundle_id: String,
    path: String,
}

#[cfg(not(target_os = "macos"))]
#[tauri::command]
async fn get_app_list() -> Result<Vec<AppMetadata>, String> {
    Ok(vec![])
}

#[cfg(not(target_os = "macos"))]
#[tauri::command]
async fn get_app_icon(_bundle_id: String) -> Result<String, String> {
    Err("Not supported on this platform".to_string())
}

#[tauri::command]
async fn start_sidecar(
    app: tauri::AppHandle,
    sidecar_manager: State<'_, Arc<SidecarManager>>,
) -> Result<(), String> {
    sidecar_manager.inner().start_sidecar(&app).await
}

#[tauri::command]
async fn stop_sidecar(
    app: tauri::AppHandle,
    sidecar_manager: State<'_, Arc<SidecarManager>>,
) -> Result<(), String> {
    sidecar_manager.inner().stop_sidecar(&app).await
}

#[tauri::command]
fn sidecar_status(sidecar_manager: State<'_, Arc<SidecarManager>>) -> bool {
    sidecar_manager.inner().is_running()
}

#[tauri::command]
async fn sidecar_health(sidecar_manager: State<'_, Arc<SidecarManager>>) -> Result<String, String> {
    sidecar_manager.inner().health_check().await
}

#[tauri::command]
fn sidecar_error(sidecar_manager: State<'_, Arc<SidecarManager>>) -> Option<String> {
    sidecar_manager.inner().get_error()
}

#[tauri::command]
async fn send_prompt(
    prompt: String,
    sidecar_manager: State<'_, Arc<SidecarManager>>,
) -> Result<String, String> {
    sidecar_manager.inner().send_prompt(&prompt).await
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let sidecar_manager = Arc::new(SidecarManager::new());

    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_macos_permissions::init())
        .manage(sidecar_manager.clone())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            get_app_list,
            get_app_icon,
            start_sidecar,
            stop_sidecar,
            sidecar_status,
            sidecar_health,
            sidecar_error,
            send_prompt
        ])
        .setup(move |app| {
            // Load environment variables from .env file
            dotenv::dotenv().ok();
            
            #[cfg(desktop)]
            app.handle().plugin(tauri_plugin_updater::Builder::new().build())?;
            app.handle().plugin(tauri_plugin_process::init())?;
            
            let window = app.get_webview_window("main").unwrap();

            #[cfg(target_os = "macos")]
            apply_vibrancy(&window, NSVisualEffectMaterial::HudWindow, None, None).expect("Unsupported platform! 'apply_vibrancy' is only supported on macOS");

            #[cfg(target_os = "windows")]
            apply_blur(&window, Some((18, 18, 18, 125))).expect("Unsupported platform! 'apply_blur' is only supported on Windows");

            // set background color only when building for macOS
            #[cfg(target_os = "macos")]
            {
                let ns_window = window.ns_window().unwrap();
                unsafe {
                    let bg_color = NSColor::colorWithRed_green_blue_alpha(23.0/ 255.0, 23.0/ 255.0, 23.0/ 255.0, 1.0);
                    let ns_window_ref = &*(ns_window as *const NSWindow);
                    ns_window_ref.setBackgroundColor(Some(&bg_color));
                }
            }

            // Clone for auto-start
            let startup_manager = sidecar_manager.clone();
            let startup_handle = app.handle().clone();

            // Check if sidecar should be auto-started (defaults to true for backward compatibility)
            let sidecar_enabled = env::var("SIDECAR_ENABLED")
                .unwrap_or_else(|_| "true".to_string())
                .to_lowercase() == "true";

            if sidecar_enabled {
                // Auto-start sidecar on app launch
                tauri::async_runtime::spawn(async move {
                    if let Err(e) = startup_manager.start_sidecar(&startup_handle).await {
                        eprintln!("Failed to auto-start sidecar: {}", e);
                    }
                });
            } else {
                println!("Sidecar auto-start disabled via SIDECAR_ENABLED environment variable");
            }


            // Create system tray
            // let quit_item = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
            // let show_item = MenuItem::with_id(app, "show", "Show", true, None::<&str>)?;
            // let hide_item = MenuItem::with_id(app, "hide", "Hide", true, None::<&str>)?;
            // // let sidecar_status_item =
            // //     MenuItem::with_id(app, "sidecar_status", "Sidecar Status", true, None::<&str>)?;

            // let tray_menu = Menu::with_items(
            //     app,
            //     &[&show_item, &hide_item, &quit_item],
            // )?;

            // let _tray = TrayIconBuilder::new()
            //     .icon(app.default_window_icon().unwrap().clone())
            //     .menu(&tray_menu)
            //     .show_menu_on_left_click(false)
            //     .on_menu_event(|app, event| match event.id.as_ref() {
            //         "quit" => {
            //             println!("Quit menu item clicked");
            //             app.exit(0);
            //         }
            //         "show" => {
            //             println!("Show menu item clicked");
            //             if let Some(window) = app.get_webview_window("main") {
            //                 let _ = window.show();
            //                 let _ = window.set_focus();
            //             }
            //         }
            //         "hide" => {
            //             println!("Hide menu item clicked");
            //             if let Some(window) = app.get_webview_window("main") {
            //                 let _ = window.hide();
            //             }
            //         }
            //         _ => {
            //             println!("Unhandled menu item: {:?}", event.id);
            //         }
            //     })
            //     .on_tray_icon_event(|tray, event| match event {
            //         TrayIconEvent::Click {
            //             button: MouseButton::Left,
            //             button_state: MouseButtonState::Up,
            //             ..
            //         } => {
            //             println!("Left click on tray icon");
            //             let app = tray.app_handle();
            //             if let Some(window) = app.get_webview_window("main") {
            //                 if window.is_visible().unwrap_or(false) {
            //                     let _ = window.hide();
            //                 } else {
            //                     let _ = window.show();
            //                     let _ = window.set_focus();
            //                 }
            //             }
            //         }
            //         TrayIconEvent::DoubleClick {
            //             button: MouseButton::Left,
            //             ..
            //         } => {
            //             println!("Double click on tray icon");
            //             let app = tray.app_handle();
            //             if let Some(window) = app.get_webview_window("main") {
            //                 let _ = window.show();
            //                 let _ = window.set_focus();
            //             }
            //         }
            //         _ => {
            //             println!("Unhandled tray event: {:?}", event);
            //         }
            //     })
            //     .build(app)?;

            // Register global shortcut for window toggle
            #[cfg(desktop)]
            {
                // Use Cmd+Shift+T on macOS, Ctrl+Shift+T on Windows/Linux
                #[cfg(target_os = "macos")]
                let toggle_shortcut = Shortcut::new(Some(Modifiers::SUPER | Modifiers::SHIFT), Code::KeyT);
                
                #[cfg(not(target_os = "macos"))]
                let toggle_shortcut = Shortcut::new(Some(Modifiers::CONTROL | Modifiers::SHIFT), Code::KeyT);

                app.handle().plugin(
                    tauri_plugin_global_shortcut::Builder::new().with_handler(move |_app, shortcut, event| {
                        if shortcut == &toggle_shortcut {
                            match event.state() {
                                ShortcutState::Pressed => {
                                    println!("Global shortcut pressed - toggling window visibility");
                                    if let Some(window) = _app.get_webview_window("main") {
                                        if window.is_visible().unwrap_or(false) {
                                            let _ = window.hide();
                                        } else {
                                            let _ = window.show();
                                            let _ = window.set_focus();
                                        }
                                    }
                                }
                                ShortcutState::Released => {
                                    // Handle release if needed
                                }
                            }
                        }
                    })
                    .build(),
                )?;

                app.global_shortcut().register(toggle_shortcut)?;
                println!("Global shortcut registered: Cmd+Shift+T (macOS) / Ctrl+Shift+T (Windows/Linux)");
            }

            // Set up NSWorkspace notifications for real-time app changes
            #[cfg(target_os = "macos")]
            {
                let app_handle = app.handle().clone();
                
                // Set up proper NSWorkspace notification observers
                std::thread::spawn(move || {
                    autoreleasepool(|_| {
                        unsafe {
                            // Get the shared workspace and its notification center
                            let workspace = NSWorkspace::sharedWorkspace();
                            let nc = workspace.notificationCenter();
                            
                            // Define notification names as NSString constants
                            let launch_notification = ns_string!("NSWorkspaceDidLaunchApplicationNotification");
                            let terminate_notification = ns_string!("NSWorkspaceDidTerminateApplicationNotification");
                            
                            // Create observer for app launches
                            let launch_app_handle = app_handle.clone();
                            let launch_block = StackBlock::new(move |_notif: NonNull<objc2_foundation::NSNotification>| {
                                let _ = launch_app_handle.emit("app-list-changed", ());
                            });
                            
                            // Create observer for app terminations
                            let term_app_handle = app_handle.clone();
                            let term_block = StackBlock::new(move |_notif: NonNull<objc2_foundation::NSNotification>| {
                                let _ = term_app_handle.emit("app-list-changed", ());
                            });
                            
                            // Register observers
                            let _launch_token = nc.addObserverForName_object_queue_usingBlock(
                                Some(launch_notification),
                                None,  // any sender
                                None,  // current thread queue
                                &launch_block,
                            );
                            
                            let _term_token = nc.addObserverForName_object_queue_usingBlock(
                                Some(terminate_notification),
                                None,  // any sender
                                None,  // current thread queue
                                &term_block,
                            );
                            
                            println!("NSWorkspace notification observers registered for real-time app changes");
                            
                            // Keep the thread alive to process notifications
                            loop {
                                std::thread::park();
                            }
                        }
                    });
                });
            }

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
