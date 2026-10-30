import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { stripTypeScriptTypes } from 'node:module';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = fs.readFileSync(process.argv[2] || path.join(root, 'goLiveBypass/native.ts'), 'utf8');
const start = source.indexOf('export async function configurePluginUpdates(');
const end = source.indexOf('function readInstalledPluginVersion(', start);
assert(start >= 0 && end > start);
let timers = 0;
const context = vm.createContext({
    pluginUpdatePolicy: { enabled: true, channel: 'stable' },
    clearPluginUpdateTimers() {},
    policyFrom(value) { return { enabled: value?.enabled !== false, channel: 'stable' }; },
    PLUGIN_UPDATE_INITIAL_DELAY_MS: 1,
    PLUGIN_UPDATE_INTERVAL_MS: 1,
    setTimeout() { timers++; return { unref() {} }; },
    setInterval() { timers++; return { unref() {} }; }
});
const code = stripTypeScriptTypes(source.slice(start, end).replaceAll('export ', ''));
vm.runInContext(code + ';globalThis.api = {configurePluginUpdates, getPluginUpdateStatus, checkPluginUpdate, updatePlugin};', context);
assert.equal((await context.api.configurePluginUpdates(null, { enabled: true })).enabled, false);
assert.equal(timers, 0, 'Source update must never be scheduled');
assert.equal((await context.api.getPluginUpdateStatus(null)).pending, false);
assert.equal((await context.api.checkPluginUpdate(null)).available, false);
const update = await context.api.updatePlugin(null);
assert.equal(update.ok, false);
assert.match(update.error, /yagoraimon\/GoLiveBypass-Vesktop/);

const hookStart = source.indexOf('// Installer shutdown handshake.');
const hookEnd = source.indexOf('// End installer shutdown handshake.', hookStart);
assert(hookStart >= 0 && hookEnd > hookStart);
let callback, request = 'invalid', quit = 0, unwatched = 0;
const hookContext = vm.createContext({
    process: { env: { VENCORD_USER_DATA_DIR: '/selected-profile' } },
    join: path.join,
    rmSync() {},
    watchFile(file, options, fn) { assert.equal(options.persistent, false); callback = fn; },
    unwatchFile() { unwatched++; },
    readFileSync() { return request; },
    writeFileSync(file, content) { request = content; },
    app: { getPath() { return '/wrong-profile'; }, quit() { quit++; }, on() {} },
    log() {}
});
vm.runInContext(source.slice(hookStart, hookEnd), hookContext);
callback({ nlink: 1, mtimeMs: 1 }, { mtimeMs: 0 });
assert.equal(quit, 0, 'Unknown request must be ignored');
request = 'quit';
callback({ nlink: 1, mtimeMs: 2 }, { mtimeMs: 1 });
assert.equal(request, 'accepted');
assert.equal(quit, 1);
assert.equal(unwatched, 1);
console.log('PASS: packaged updater stays disabled and shutdown uses app.quit().');
