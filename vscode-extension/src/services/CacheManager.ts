import * as vscode from 'vscode';
import * as crypto from 'crypto';
import { Finding } from './ApiClient';

interface CacheEntry {
    findings: Finding[];
    timestamp: number;
    contentHash: string;
    version: string;
}

interface ScanCache {
    [filePath: string]: CacheEntry;
}

export class CacheManager {
    private cache: ScanCache = {};
    private readonly CACHE_VERSION = '1.0.0';
    private readonly context: vscode.ExtensionContext;
    private maxAge: number;

    constructor(context: vscode.ExtensionContext, maxAgeSeconds: number = 300) {
        this.context = context;
        this.maxAge = maxAgeSeconds * 1000; // Convert to milliseconds
        this.loadCache();
    }

    /**
     * Get cached findings for a file if they exist and are still valid
     */
    getCachedFindings(filePath: string, content: string): Finding[] | null {
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
    cacheFindings(filePath: string, content: string, findings: Finding[]): void {
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
    invalidateFile(filePath: string): void {
        if (this.cache[filePath]) {
            delete this.cache[filePath];
            this.saveCache();
        }
    }

    /**
     * Invalidate all cache entries
     */
    invalidateAll(): void {
        this.cache = {};
        this.saveCache();
    }

    /**
     * Clean up expired cache entries
     */
    cleanup(): void {
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
    getStats(): { totalEntries: number; totalSize: number; oldestEntry: number | null } {
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
    hasCachedResults(filePath: string, content: string): boolean {
        return this.getCachedFindings(filePath, content) !== null;
    }

    /**
     * Update cache max age
     */
    updateMaxAge(maxAgeSeconds: number): void {
        this.maxAge = maxAgeSeconds * 1000;
    }

    private generateContentHash(content: string): string {
        return crypto.createHash('sha256').update(content).digest('hex');
    }

    private loadCache(): void {
        try {
            const cacheData = this.context.globalState.get<ScanCache>('agentscan.cache', {});
            this.cache = cacheData;
            
            // Clean up expired entries on load
            this.cleanup();
        } catch (error) {
            console.error('Failed to load cache:', error);
            this.cache = {};
        }
    }

    private saveCache(): void {
        try {
            this.context.globalState.update('agentscan.cache', this.cache);
        } catch (error) {
            console.error('Failed to save cache:', error);
        }
    }

    dispose(): void {
        this.saveCache();
    }
}