// Package stealth - browser fingerprint masking
// This is MANDATORY stealth technique #3
package stealth

import (
	"math/rand"

	"github.com/go-rod/rod"
	"github.com/go-rod/stealth"
	"github.com/rs/zerolog"
)

// Common user agents for realistic fingerprinting
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
}

// Common screen resolutions
var screenResolutions = []struct {
	Width  int
	Height int
}{
	{1920, 1080},
	{1366, 768},
	{1536, 864},
	{1440, 900},
	{1280, 720},
	{2560, 1440},
	{1680, 1050},
}

// Common timezones
var timezones = []string{
	"America/New_York",
	"America/Chicago",
	"America/Los_Angeles",
	"America/Denver",
	"Europe/London",
	"Europe/Paris",
}

// ApplyFingerprint applies fingerprint masking to a page
func ApplyFingerprint(page *rod.Page, logger zerolog.Logger) error {
	logger.Debug().Msg("Applying fingerprint masking")

	// Use go-rod/stealth for base evasions
	// This handles: navigator.webdriver, chrome.runtime, etc.
	_, err := page.EvalOnNewDocument(stealth.JS)
	if err != nil {
		return err
	}

	// Additional fingerprint customization
	fingerprintJS := generateFingerprintJS()
	_, err = page.EvalOnNewDocument(fingerprintJS)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to apply custom fingerprint JS")
		// Don't fail on this, stealth.JS is the critical part
	}

	return nil
}

// generateFingerprintJS creates JavaScript to customize browser fingerprint
func generateFingerprintJS() string {
	// Select random values for fingerprint
	resolution := screenResolutions[rand.Intn(len(screenResolutions))]
	timezone := timezones[rand.Intn(len(timezones))]

	return `
		// Override screen properties
		Object.defineProperty(screen, 'width', { get: () => ` + itoa(resolution.Width) + ` });
		Object.defineProperty(screen, 'height', { get: () => ` + itoa(resolution.Height) + ` });
		Object.defineProperty(screen, 'availWidth', { get: () => ` + itoa(resolution.Width) + ` });
		Object.defineProperty(screen, 'availHeight', { get: () => ` + itoa(resolution.Height-40) + ` });
		Object.defineProperty(screen, 'colorDepth', { get: () => 24 });
		Object.defineProperty(screen, 'pixelDepth', { get: () => 24 });

		// Override navigator properties
		Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => ` + itoa(4+rand.Intn(13)) + ` });
		Object.defineProperty(navigator, 'deviceMemory', { get: () => ` + itoa([]int{4, 8, 16}[rand.Intn(3)]) + ` });
		Object.defineProperty(navigator, 'maxTouchPoints', { get: () => 0 });

		// Override timezone
		const originalDateTimeFormat = Intl.DateTimeFormat;
		Intl.DateTimeFormat = function(locale, options) {
			options = options || {};
			options.timeZone = options.timeZone || '` + timezone + `';
			return new originalDateTimeFormat(locale, options);
		};

		// Override plugins to look more realistic
		Object.defineProperty(navigator, 'plugins', {
			get: () => {
				const plugins = [
					{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
					{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
					{ name: 'Native Client', filename: 'internal-nacl-plugin', description: '' }
				];
				plugins.length = 3;
				return plugins;
			}
		});

		// Override WebGL vendor/renderer
		const getParameterProxyHandler = {
			apply: function(target, thisArg, args) {
				const param = args[0];
				const gl = thisArg;
				
				// UNMASKED_VENDOR_WEBGL
				if (param === 37445) {
					return 'Google Inc. (NVIDIA)';
				}
				// UNMASKED_RENDERER_WEBGL
				if (param === 37446) {
					return 'ANGLE (NVIDIA, NVIDIA GeForce GTX 1080 Direct3D11 vs_5_0 ps_5_0, D3D11)';
				}
				
				return target.apply(thisArg, args);
			}
		};

		// Apply WebGL override if WebGL context exists
		try {
			const canvas = document.createElement('canvas');
			const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
			if (gl) {
				const originalGetParameter = gl.getParameter;
				gl.getParameter = new Proxy(originalGetParameter, getParameterProxyHandler);
			}
		} catch(e) {}

		// Disable automation detection via permissions
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => {
			if (parameters.name === 'notifications') {
				return Promise.resolve({ state: Notification.permission });
			}
			return originalQuery(parameters);
		};

		// Add realistic battery API (if supported)
		if ('getBattery' in navigator) {
			navigator.getBattery = () => Promise.resolve({
				charging: true,
				chargingTime: 0,
				dischargingTime: Infinity,
				level: 1.0,
				addEventListener: () => {},
				removeEventListener: () => {}
			});
		}

		// Override connection info
		if ('connection' in navigator) {
			Object.defineProperty(navigator.connection, 'effectiveType', { get: () => '4g' });
			Object.defineProperty(navigator.connection, 'rtt', { get: () => 50 });
			Object.defineProperty(navigator.connection, 'downlink', { get: () => 10 });
		}
	`
}

// GetRandomUserAgent returns a random user agent string
func GetRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// GetRandomResolution returns a random screen resolution
func GetRandomResolution() (int, int) {
	res := screenResolutions[rand.Intn(len(screenResolutions))]
	return res.Width, res.Height
}

// itoa converts int to string (simple helper to avoid import)
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	
	var result []byte
	negative := i < 0
	if negative {
		i = -i
	}
	
	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}
	
	if negative {
		result = append([]byte{'-'}, result...)
	}
	
	return string(result)
}
