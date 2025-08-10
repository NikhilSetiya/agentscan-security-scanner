"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
const assert = __importStar(require("assert"));
const vscode = __importStar(require("vscode"));
const ConfigurationManager_1 = require("../../services/ConfigurationManager");
suite('Extension Test Suite', () => {
    vscode.window.showInformationMessage('Start all tests.');
    test('Extension should be present', () => {
        assert.ok(vscode.extensions.getExtension('agentscan.agentscan-security'));
    });
    test('Should activate extension', async () => {
        const extension = vscode.extensions.getExtension('agentscan.agentscan-security');
        if (extension) {
            await extension.activate();
            assert.ok(extension.isActive);
        }
    });
    test('Should register commands', async () => {
        const commands = await vscode.commands.getCommands(true);
        const expectedCommands = [
            'agentscan.scanFile',
            'agentscan.scanWorkspace',
            'agentscan.clearFindings',
            'agentscan.showSettings',
            'agentscan.suppressFinding'
        ];
        expectedCommands.forEach(command => {
            assert.ok(commands.includes(command), `Command ${command} should be registered`);
        });
    });
});
suite('Configuration Manager Tests', () => {
    let config;
    setup(() => {
        config = new ConfigurationManager_1.ConfigurationManager();
    });
    test('Should have default configuration values', () => {
        assert.strictEqual(config.getServerUrl(), 'http://localhost:8080');
        assert.strictEqual(config.isRealTimeScanningEnabled(), true);
        assert.strictEqual(config.getScanDebounceMs(), 1000);
        assert.strictEqual(config.getSeverityThreshold(), 'medium');
        assert.strictEqual(config.isInlineAnnotationsEnabled(), true);
        assert.strictEqual(config.isWebSocketEnabled(), true);
    });
    test('Should validate supported languages', () => {
        assert.ok(config.isLanguageSupported('javascript'));
        assert.ok(config.isLanguageSupported('typescript'));
        assert.ok(config.isLanguageSupported('python'));
        assert.ok(config.isLanguageSupported('go'));
        assert.ok(config.isLanguageSupported('java'));
        assert.ok(!config.isLanguageSupported('unsupported'));
    });
    test('Should filter findings by severity threshold', () => {
        // Test with medium threshold (default)
        assert.ok(config.shouldShowFinding('high'));
        assert.ok(config.shouldShowFinding('medium'));
        assert.ok(!config.shouldShowFinding('low'));
    });
    test('Should validate configuration', () => {
        const validation = config.validateConfiguration();
        // Should fail validation due to missing API key
        assert.strictEqual(validation.isValid, false);
        assert.ok(validation.errors.some(error => error.includes('API key')));
    });
    test('Should generate WebSocket URL correctly', () => {
        const wsUrl = config.getWebSocketUrl();
        assert.ok(wsUrl.startsWith('ws://'));
        assert.ok(wsUrl.includes('/ws'));
    });
});
//# sourceMappingURL=extension.test.js.map