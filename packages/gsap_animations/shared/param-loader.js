/**
 * Shared parameter loader for GSAP animations
 * Loads parameters from schema.json to ensure single source of truth
 */

// Cache to avoid repeated schema fetches
const schemaCache = new Map();

/**
 * Gets the current animation name from the URL path
 * @returns {string} Animation name (e.g., 'bounce-overlay')
 */
function getCurrentAnimationName() {
    // Extract animation name from URL path like '/gsap_animations/bounce-overlay/'
    const path = window.location.pathname;
    const parts = path.split('/').filter(p => p);

    // Find gsap_animations in the path and get the next part
    const gsapIndex = parts.findIndex(p => p === 'gsap_animations');
    if (gsapIndex !== -1 && gsapIndex + 1 < parts.length) {
        return parts[gsapIndex + 1];
    }

    throw new Error('Could not determine animation name from URL');
}

/**
 * Fetches and parses the schema.json for the current animation
 * @returns {Promise<Object>} Parsed schema object
 */
async function fetchAnimationSchema() {
    const animationName = getCurrentAnimationName();

    // Check cache first
    if (schemaCache.has(animationName)) {
        return schemaCache.get(animationName);
    }

    try {
        // Try API endpoint first (when served via Go backend)
        const apiResponse = await fetch(`/api/gsap_animations/${animationName}`);
        if (apiResponse.ok) {
            const schema = await apiResponse.json();
            schemaCache.set(animationName, schema);
            return schema;
        }
    } catch (e) {
        console.warn('API fetch failed, trying direct file access:', e);
    }

    try {
        // Fallback to direct file access (for development/local files)
        const fileResponse = await fetch(`./schema.json`);
        if (fileResponse.ok) {
            const schema = await fileResponse.json();
            schemaCache.set(animationName, schema);
            return schema;
        }
    } catch (e) {
        console.warn('Direct file fetch failed:', e);
    }

    throw new Error(`Could not load schema for animation: ${animationName}`);
}

/**
 * Extracts default parameter values from schema
 * @param {Object} schema - The parsed schema object
 * @returns {Object} Object with parameter names as keys and default values as values
 */
function extractDefaults(schema) {
    const defaults = {};

    if (schema.parameters && Array.isArray(schema.parameters)) {
        schema.parameters.forEach(param => {
            if (param.name && param.default !== undefined) {
                defaults[param.name] = param.default;
            }
        });
    }

    return defaults;
}

/**
 * Parses URL parameters and converts them to appropriate types
 * @param {Object} defaults - Default values to use for type inference
 * @returns {Object} Parsed URL parameters
 */
function parseUrlParams(defaults) {
    const urlParams = new URLSearchParams(window.location.search);
    const parsedParams = {};

    for (const [key, value] of urlParams) {
        // Skip if this parameter isn't in defaults (unknown parameter)
        if (!(key in defaults)) {
            continue;
        }

        // Convert based on the type of the default value
        const defaultValue = defaults[key];

        if (typeof defaultValue === 'boolean') {
            parsedParams[key] = value === 'true';
        } else if (typeof defaultValue === 'number') {
            const numValue = parseFloat(value);
            parsedParams[key] = isNaN(numValue) ? defaultValue : numValue;
        } else {
            // String or other types - use as-is
            parsedParams[key] = value;
        }
    }

    return parsedParams;
}

/**
 * Loads parameters for the current animation from schema.json with URL overrides
 * @returns {Promise<Object>} Final parameter values
 */
async function loadParameters() {
    try {
        // Fetch schema and extract defaults
        const schema = await fetchAnimationSchema();
        const defaults = extractDefaults(schema);

        // Parse URL parameters
        const urlOverrides = parseUrlParams(defaults);

        // Merge defaults with URL overrides
        const finalParams = { ...defaults, ...urlOverrides };

        console.log('Parameter loading results:');
        console.log('- Animation:', getCurrentAnimationName());
        console.log('- Schema defaults:', defaults);
        console.log('- URL overrides:', urlOverrides);
        console.log('- Final parameters:', finalParams);

        return finalParams;
    } catch (error) {
        console.error('Failed to load parameters:', error);
        throw error;
    }
}

// Export for use in animations
window.AnimationParams = {
    loadParameters,
    getCurrentAnimationName,
    fetchAnimationSchema
};