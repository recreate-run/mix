package adblock

// AdBlockScript is injected into pages to block ads via CSS hiding
// This is a lightweight alternative to full extensions like uBlock Origin
// Works reliably in automation mode where extension content scripts may not inject
const AdBlockScript = `
(function() {
	'use strict';

	// Common ad selectors based on uBlock Origin's cosmetic filters
	const adSelectors = [
		// Generic ad classes
		'.ad', '.ads', '.ad-banner', '.ad-slot', '.ad-container', '.ad-content',
		'.advertisement', '.advertising', '.adsbygoogle', '.sponsored', '.sponsor',

		// Google Ads
		'#google_ads_iframe', '.google-ad', 'ins.adsbygoogle',

		// Specific ad networks
		'[id^="google_ads"]', '[class*="google-ad"]', '[id*="doubleclick"]',
		'[class*="doubleclick"]', '[id*="ad-banner"]', '[id*="ad-slot"]',

		// Common ad containers
		'#ad-top', '#ad-bottom', '#ad-left', '#ad-right', '#ad-header',
		'#ad-footer', '#ad-sidebar', '.ad-zone', '.adzone',

		// Specific patterns
		'[class*="ad-wrapper"]', '[class*="ad-block"]', '[id*="ad-wrapper"]',
		'[class*="sponsored"]', '[data-ad-slot]', '[data-google-query-id]'
	];

	// Block ads by hiding them with CSS
	function blockAds() {
		adSelectors.forEach(selector => {
			try {
				document.querySelectorAll(selector).forEach(el => {
					el.style.setProperty('display', 'none', 'important');
					el.style.setProperty('visibility', 'hidden', 'important');
					el.style.setProperty('opacity', '0', 'important');
					el.style.setProperty('height', '0', 'important');
					el.style.setProperty('width', '0', 'important');
				});
			} catch (e) {
				// Ignore selector errors
			}
		});
	}

	// Run immediately
	blockAds();

	// Run after DOM loads
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', blockAds);
	}

	// Watch for dynamic content
	const observer = new MutationObserver(blockAds);
	observer.observe(document.documentElement, {
		childList: true,
		subtree: true
	});

	// Mark that ad blocking is active
	window.__adBlockActive = true;

})();
`
