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
exports.CacheManager = void 0;
const crypto = __importStar(require("crypto"));
class CacheManager {
    constructor(context, maxAgeSeconds = 300) {
        this.cache = {};
        this.CACHE_VERSION = '1.0.0';
        this.context = context;
        this.maxAge = maxAgeSeconds * 1000; // Convert to milliseconds
        this.loadCache();
    }
    /**
     * Get cached findings for a file if they exist and are still valid
     */
    getCachedFindings(filePath, content) {
        const entry = this.cache[filePath];
        if (!entry) {
            return null;
        }
        // Check if cache is expired
        if (Date.now() - entry.timestamp > this.maxAge) {
            delete this.cache[filePath];
            this.saveCache();
            return null;
        }
        // Check if content has changed
        const contentHash = this.generateContentHash(content);
        if (entry.contentHash !== contentHash) {
            delete this.cache[filePath];
            this.saveCache();
            return null;
        }
        // Check if cache version is compatible
        if (entry.version !== this.CACHE_VERSION) {
            delete this.cache[filePath];
            this.saveCache();
            return null;
        }
        return entry.findings;
    }
    /**
     * Cache findings for a file
     */
    cacheFindings(filePath, content, findings) {
        const contentHash = this.generateContentHash(content);
        this.cache[filePath] = {
            findings: findings,
            timestamp: Date.now(),
            contentHash: contentHash,
            version: this.CACHE_VERSION
        };
        this.saveCache();
    }
    /**
     * Invalidate cache for a specific file
     */
    invalidateFile(filePath) {
        if (this.cache[filePath]) {
            delete this.cache[filePath];
            this.saveCache();
        }
    }
    /**
     * Invalidate all cache entries
     */
    invalidateAll() {
        this.cache = {};
        this.saveCache();
    }
    /**
     * Clean up expired cache entries
     */
    cleanup() {
        const now = Date.now();
        let hasChanges = false;
        for (const [filePath, entry] of Object.entries(this.cache)) {
            if (now - entry.timestamp > this.maxAge) {
                delete this.cache[filePath];
                hasChanges = true;
            }
        }
        if (hasChanges) {
            this.saveCache();
        }
    }
    /**
     * Get cache statistics
     */
    getStats() {
        const entries = Object.values(this.cache);
        const totalEntries = entries.length;
        const totalSize = JSON.stringify(this.cache).length;
        const oldestEntry = entries.length > 0
            ? Math.min(...entries.map(e => e.timestamp))
            : null;
        return { totalEntries, totalSize, oldestEntry };
    }
    /**
     * Check if a file has cached results
     */
    hasCachedResults(filePath, content) {
        return this.getCachedFindings(filePath, content) !== null;
    }
    /**
     * Update cache max age
     */
    updateMaxAge(maxAgeSeconds) {
        this.maxAge = maxAgeSeconds * 1000;
    }
    generateContentHash(content) {
        return crypto.createHash('sha256').update(content).digest('hex');
    }
    loadCache() {
        try {
            const cacheData = this.context.globalState.get('agentscan.cache', {});
            this.cache = cacheData;
            // Clean up expired entries on load
            this.cleanup();
        }
        catch (error) {
            console.error('Failed to load cache:', error);
            this.cache = {};
        }
    }
    saveCache() {
        try {
            this.context.globalState.update('agentscan.cache', this.cache);
        }
        catch (error) {
            console.error('Failed to save cache:', error);
        }
    }
    dispose() {
        this.saveCache();
    }
}
exports.CacheManager = CacheManager;
//# sourceMappingURL=CacheManager.js.map