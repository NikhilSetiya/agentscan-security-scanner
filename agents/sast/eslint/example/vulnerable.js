// Example vulnerable JavaScript code for testing ESLint Security Agent
// This file contains various security vulnerabilities that should be detected

const fs = require('fs');
const crypto = require('crypto');

// 1. Dangerous eval() usage - HIGH severity
function processUserCode(userInput) {
    // This should trigger: security/detect-eval-with-expression, no-eval
    eval("var result = " + userInput);
    return result;
}

// 2. Non-literal require() - HIGH severity  
function loadUserModule(moduleName) {
    // This should trigger: security/detect-non-literal-require
    return require(moduleName);
}

// 3. Object injection vulnerability - HIGH severity
function updateUserData(userInput) {
    var userData = {};
    // This should trigger: security/detect-object-injection
    userData[userInput.key] = userInput.value;
    return userData;
}

// 4. Non-literal filesystem path - MEDIUM severity
function readUserFile(filename) {
    // This should trigger: security/detect-non-literal-fs-filename
    return fs.readFileSync(filename, 'utf8');
}

// 5. Weak random number generation - LOW severity
function generateToken() {
    // This should trigger: security/detect-pseudoRandomBytes
    return crypto.pseudoRandomBytes(32).toString('hex');
}

// 6. Implied eval through setTimeout - HIGH severity
function scheduleUserCode(code) {
    // This should trigger: no-implied-eval
    setTimeout(code, 1000);
}

// 7. Function constructor - HIGH severity
function createUserFunction(code) {
    // This should trigger: no-new-func
    return new Function('return ' + code);
}

// 8. Unsafe regular expression - MEDIUM severity
function validateInput(pattern, input) {
    // This should trigger: security/detect-unsafe-regex
    const regex = new RegExp(pattern);
    return regex.test(input);
}

// 9. Buffer without assertion - MEDIUM severity
function processBuffer(data) {
    // This should trigger: security/detect-buffer-noassert
    const buffer = Buffer.allocUnsafe(data.length);
    buffer.write(data);
    return buffer;
}

// 10. Child process execution - MEDIUM severity
function executeCommand(command) {
    const { exec } = require('child_process');
    // This should trigger: security/detect-child-process
    exec(command, (error, stdout, stderr) => {
        console.log(stdout);
    });
}

// 11. Timing attack vulnerability - LOW severity
function compareSecrets(userSecret, actualSecret) {
    // This should trigger: security/detect-possible-timing-attacks
    return userSecret === actualSecret;
}

// 12. Script URL usage - MEDIUM severity (if detected)
function createScriptElement(url) {
    const script = document.createElement('script');
    // This should trigger: no-script-url (if rule is active)
    script.src = 'javascript:' + url;
    return script;
}

// Export functions for testing
module.exports = {
    processUserCode,
    loadUserModule,
    updateUserData,
    readUserFile,
    generateToken,
    scheduleUserCode,
    createUserFunction,
    validateInput,
    processBuffer,
    executeCommand,
    compareSecrets,
    createScriptElement
};