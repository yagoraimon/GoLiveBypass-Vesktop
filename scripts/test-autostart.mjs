import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { stripTypeScriptTypes } from 'node:module';

const root = process.argv[2] || path.resolve(import.meta.dirname, '..');
const source = fs.readFileSync(path.join(root, 'goLiveBypass/vpn-controller.ts'), 'utf8');
const code = stripTypeScriptTypes(source.slice(source.indexOf('export class PluginVpnController')).replaceAll('export ', ''));
const marker = '--golive-vpn-relaunch';
const native = fs.readFileSync(path.join(root, 'goLiveBypass/native.ts'), 'utf8');
const bridgeStart = native.indexOf('function controllerSettings()');
const bridgeEnd = native.indexOf('const LEGACY_EVENT_MAP', bridgeStart);
assert(bridgeStart > 0 && bridgeEnd > bridgeStart);
const bridgeCode = stripTypeScriptTypes(native.slice(bridgeStart, bridgeEnd));

function fixture(platform = 'win32', inspection = { reliable: true, active: false, owned: false }, options = {}) {
    const starts = [];
    const relaunches = [];
    const args = ['vesktop', ...(options.args || [])];
    const context = vm.createContext({
        path,
        pluginSettings: () => ({ vpnAutoStart: options.autoStart }),
        process: { platform, argv: args, pid: 42 },
        PROFILE_FILE: 'wireguard.conf', CUSTOM_PROFILE_FILE: 'custom.conf', PROFILE_ACCOUNT_FILE: 'account.json',
        ACTIVE_ROUTE_FILE: 'route.json', SERVICE_CONFIG_FILE: 'service.conf', OWNER_FILE: 'owner.lock',
        VPN_RELAUNCH_ARG: marker,
        isLinux: () => platform === 'linux',
        isSupportedWindowsArchitecture: () => true,
        isUnknownWireSockInspection: value => !value.reliable,
        safeDiagnosticDetail: String,
        windows: { inspectWireSockUntilReliableAsync: async () => options.reinspection || inspection },
        linux: { inspectLinuxNetworkSync: () => inspection },
        app: { relaunch: value => relaunches.push(value), exit() {} },
        errorMessage: String,
    });
    vm.runInContext(bridgeCode + code + ';globalThis.Controller = PluginVpnController;', context);
    const controller = new context.Controller({
        dataDir: '/vpn', guiDataDir: '/gui', isEnabled: () => options.enabled !== false,
        readSettings: () => context.controllerSettings(),
        log() {}, requestRelaunch: async () => true,
    });
    controller.readOwner = () => null;
    controller.inspectWindows = () => inspection;
    const realStart = controller.startInternal.bind(controller);
    controller.startInternal = async (...values) => { starts.push(values); return { success: true, state: 'active' }; };
    if (options.state) controller.state = options.state;
    if (options.suppressed) controller.automaticBootSuppressed = true;
    return { controller, starts, relaunches, realStart };
}

for (const platform of ['win32', 'linux']) {
    const clean = fixture(platform);
    await Promise.all([clean.controller.enableAutomatic(), clean.controller.enableAutomatic()]);
    assert.equal(clean.starts.length, 1, `${platform}: fresh opening must connect once`);
    assert.deepEqual(Array.from(clean.starts[0]), [true, false], 'Automatic start must relaunch without taking over another VPN');
    await clean.controller.enableAutomatic();
    assert.equal(clean.starts.length, 1, 'Renderer reload must not repeat automatic start');

    for (const options of [
        { autoStart: false }, { enabled: false }, { args: [marker] }, { suppressed: true },
        { state: 'recovery_required' }, { state: 'dependency_missing' }, { state: 'blocked_external' },
    ]) {
        const blocked = fixture(platform, undefined, options);
        await blocked.controller.enableAutomatic();
        assert.equal(blocked.starts.length, 0, `${platform}: automatic start must respect ${JSON.stringify(options)}`);
    }
    for (const inspection of [
        { reliable: false }, { reliable: true, active: true, owned: false },
        { reliable: true, active: false, owned: false, externalConflict: true },
    ]) {
        const blocked = fixture(platform, inspection);
        await blocked.controller.enableAutomatic();
        assert.equal(blocked.starts.length, 0, `${platform}: uncertain/external state must be preserved`);
    }
    const active = fixture(platform, { reliable: true, active: true, owned: true }, { args: [marker], autoStart: false });
    await active.controller.enableAutomatic();
    assert.deepEqual(Array.from(active.starts[0]), [false], 'Existing owned tunnel must be adopted without restarting');

    const failure = fixture(platform);
    failure.controller.startInternal = async () => { failure.starts.push('failed'); return { success: false, state: 'inactive' }; };
    await failure.controller.enableAutomatic();
    await failure.controller.enableAutomatic();
    assert.equal(failure.starts.length, 1, 'Failed activation must not retry on renderer startup');
}

for (const origin of ['gui', 'managed', 'external']) {
    const race = fixture('win32', undefined, { reinspection: { reliable: true, active: true, owned: false, origin } });
    race.controller.startInternal = race.realStart;
    const result = await race.controller.enableAutomatic();
    assert.equal(result.success, false, 'An external VPN appearing during activation must be preserved');
    assert.equal(result.state, 'blocked_external');
}

const windows = fixture();
await windows.controller.requestRelaunch();
assert(windows.relaunches[0].args.includes(marker), 'Windows relaunch must carry the automatic-start guard');

const argsStart = native.indexOf('const rawArgs = process.argv.slice(1);');
const argsEnd = native.indexOf('const safeEnvKeys = [', argsStart);
assert(argsStart > 0 && argsEnd > argsStart);
const argsCode = stripTypeScriptTypes(native.slice(argsStart, argsEnd));
for (const argv of [['vesktop', '--profile=test'], ['vesktop', marker]]) {
    const context = vm.createContext({ process: { argv }, VPN_RELAUNCH_ARG: marker });
    vm.runInContext(argsCode + ';globalThis.args = clientArgs;', context);
    assert.equal(context.args.filter(value => value === marker).length, 1, 'Linux relaunch must carry exactly one guard');
}
console.log('PASS: Windows/Linux automatic activation is single-attempt, respects conflicts and stops across relaunch.');
