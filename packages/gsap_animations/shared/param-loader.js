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
    // Extract animation name from URL path: /animations/{name}/preview
    const path = window.location.pathname;
    const parts = path.split('/').filter(p => p);

    const animationsIndex = parts.findIndex(p => p === 'animations');
    if (animationsIndex !== -1 && animationsIndex + 1 < parts.length) {
        return parts[animationsIndex + 1];
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

    const apiResponse = await fetch(`/animations/${animationName}/schema`);
    if (!apiResponse.ok) {
        throw new Error(`Failed to load schema for animation: ${animationName} (${apiResponse.status})`);
    }

    const schema = await apiResponse.json();
    schemaCache.set(animationName, schema);
    return schema;
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