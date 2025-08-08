// Example safe JavaScript code that should not trigger security warnings
// This demonstrates secure coding practices

const fs = require('fs');
const crypto = require('crypto');
const path = require('path');

// 1. Safe string processing without eval
function processUserInput(userInput) {
    // Safe: Use proper parsing instead of eval
    try {
        return JSON.parse(userInput);
    } catch (error) {
        return null;
    }
}

// 2. Safe module loading with whitelist
function loadModule(moduleName) {
    // Safe: Use whitelist of allowed modules
    const allowedModules = ['lodash', 'moment', 'axios'];
    if (allowedModules.includes(moduleName)) {
        return require(moduleName);
    }
    throw new Error('Module not allowed');
}

// 3. Safe object property assignment
function updateUserData(key, value) {
    // Safe: Validate and sanitize input
    const allowedKeys = ['name', 'email', 'age'];
    if (allowedKeys.includes(key)) {
        const userData = {};
        userData[key] = value;
        return userData;
    }
    throw new Error('Invalid key');
}

// 4. Safe file operations with path validation
function readConfigFile() {
    // Safe: Use literal paths and path validation
    const configPath = path.join(__dirname, 'config.json');
    if (fs.existsSync(configPath)) {
        return fs.readFileSync(configPath, 'utf8');
    }
    return null;
}

// 5. Secure random number generation
function generateSecureToken() {
    // Safe: Use cryptographically secure random generation
    return crypto.randomBytes(32).toString('hex');
}

// 6. Safe delayed execution
function scheduleTask(callback, delay) {
    // Safe: Use function reference instead of string
    setTimeout(callback, delay);
}

// 7. Safe function creation
function createValidator(rules) {
    // Safe: Use proper function construction
    return function(input) {
        return rules.every(rule => rule(input));
    };
}

// 8. Safe regular expression usage
function validateEmail(email) {
    // Safe: Use literal regex patterns
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
}

// 9. Safe buffer operations
function processData(data) {
    // Safe: Use safe buffer allocation
    const buffer = Buffer.alloc(data.length);
    buffer.write(data);
    return buffer;
}

// 10. Safe command execution with validation
function executeAllowedCommand(command) {
    // Safe: Validate against whitelist
    const allowedCommands = ['ls', 'pwd', 'date'];
    if (allowedCommands.includes(command)) {
        const { execSync } = require('child_process');
        return execSync(command, { encoding: 'utf8' });
    }
    throw new Error('Command not allowed');
}

// 11. Safe secret comparison
function compareSecrets(userSecret, actualSecret) {
    // Safe: Use constant-time comparison
    return crypto.timingSafeEqual(
        Buffer.from(userSecret),
        Buffer.from(actualSecret)
    );
}

// 12. Safe DOM manipulation
function createScriptElement(src) {
    // Safe: Validate URL before creating script
    if (src.startsWith('https://') || src.startsWith('/')) {
        const script = document.createElement('script');
        script.src = src;
        return script;
    }
    throw new Error('Invalid script source');
}

// 13. Input sanitization helper
function sanitizeInput(input) {
    // Safe: Proper input sanitization
    return input
        .replace(/[<>]/g, '') // Remove potential HTML tags
        .trim()
        .substring(0, 1000); // Limit length
}

// 14. Safe error handling
function safeOperation(operation) {
    try {
        return operation();
    } catch (error) {
        // Safe: Don't expose internal errors
        console.error('Operation failed:', error.message);
        return null;
    }
}

// Export functions
module.exports = {
    processUserInput,
    loadModule,
    updateUserData,
    readConfigFile,
    generateSecureToken,
    scheduleTask,
    createValidator,
    validateEmail,
    processData,
    executeAllowedCommand,
    compareSecrets,
    createScriptElement,
    sanitizeInput,
    safeOperation
};