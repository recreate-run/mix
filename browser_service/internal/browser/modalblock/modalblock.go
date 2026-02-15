package modalblock

// ModalBlockScript is a JavaScript snippet that blocks HTML modal popups, overlays, and dialogs.
// It runs before page content loads and uses MutationObserver to catch dynamically created modals.
// Targets: cookie banners, location popups, newsletter overlays, and generic modal dialogs.
const ModalBlockScript = `(function() {
	try {
		// CSS to hide common modal patterns
		var hideModalCSS = "" +
			"/* Generic modal and dialog patterns */" +
			"[role='dialog']," +
			"[role='alertdialog']," +
			".modal," +
			".popup," +
			".overlay," +
			".popup-overlay," +
			".modal-overlay," +
			".dialog," +
			"" +
			"/* Cookie and consent banners */" +
			"[class*='cookie']," +
			"[class*='consent']," +
			"[class*='gdpr']," +
			"[id*='cookie']," +
			"[id*='consent']," +
			"[id*='gdpr']," +
			"#onetrust-consent-sdk," +
			".cky-consent-container," +
			"" +
			"/* Newsletter and subscription popups */" +
			"[class*='newsletter']," +
			"[class*='subscribe']," +
			"[class*='subscription']," +
			"[id*='newsletter']," +
			"[id*='popup']," +
			"" +
			"/* Location and delivery popups (Amazon-style) */" +
			"[class*='location']," +
			"[class*='delivery']," +
			"[data-testid*='modal']," +
			"[data-testid*='dialog']" +
			"{" +
			"  display: none !important;" +
			"  visibility: hidden !important;" +
			"  opacity: 0 !important;" +
			"  pointer-events: none !important;" +
			"}" +
			"" +
			"/* Remove backdrop/overlay darkening */" +
			".modal-backdrop," +
			".overlay-backdrop," +
			"[class*='backdrop']" +
			"{" +
			"  display: none !important;" +
			"}" +
			"" +
			"/* Prevent body scroll lock (modals often disable scrolling) */" +
			"body {" +
			"  overflow: auto !important;" +
			"}";

		// Function to inject CSS
		function injectCSS() {
			if (!document.head && !document.documentElement) {
				// DOM not ready yet, try again soon
				setTimeout(injectCSS, 10);
				return;
			}

			var styleEl = document.createElement('style');
			styleEl.textContent = hideModalCSS;
			styleEl.id = '__modal-blocker-styles';
			(document.head || document.documentElement).appendChild(styleEl);
		}

		// Function to remove modal elements from DOM
		function removeModals() {
			if (!document.body) {
				return; // Body not ready yet
			}

			var selectors = [
				"[role='dialog']",
				"[role='alertdialog']",
				'.modal',
				'.popup',
				'.overlay',
				"[class*='cookie']",
				"[class*='consent']",
				"[id*='cookie']",
				"[id*='popup']",
				"[data-testid*='modal']",
				"[data-testid*='dialog']"
			];

			selectors.forEach(function(selector) {
				document.querySelectorAll(selector).forEach(function(el) {
					// Check if it's likely a modal (position fixed/absolute, high z-index)
					var styles = window.getComputedStyle(el);
					var isModal = (
						(styles.position === 'fixed' || styles.position === 'absolute') &&
						parseInt(styles.zIndex) > 100
					) || selector.indexOf('modal') >= 0 || selector.indexOf('dialog') >= 0 || selector.indexOf('popup') >= 0;

					if (isModal) {
						el.remove();
					}
				});
			});

			// Remove body scroll lock classes
			if (document.body) {
				document.body.classList.remove('modal-open', 'no-scroll', 'overflow-hidden');
				document.body.style.overflow = '';
			}
		}

		// Inject CSS as early as possible
		injectCSS();

		// Watch for dynamically added modals
		function startObserver() {
			if (!document.body) {
				// Body not ready, wait for DOM
				if (document.readyState === 'loading') {
					document.addEventListener('DOMContentLoaded', startObserver);
				} else {
					setTimeout(startObserver, 10);
				}
				return;
			}

			// Remove existing modals
			removeModals();

			// Set up mutation observer
			var observer = new MutationObserver(function() {
				removeModals();
			});

			observer.observe(document.body, {
				childList: true,
				subtree: true
			});
		}

		// Start observing
		startObserver();

		// Mark as active for debugging
		window.__modalBlockActive = true;

		console.log('[ModalBlocker] Active - blocking HTML modals and overlays');
	} catch (e) {
		console.error('[ModalBlocker] Error:', e);
		window.__modalBlockError = e.message;
	}
})();
`
