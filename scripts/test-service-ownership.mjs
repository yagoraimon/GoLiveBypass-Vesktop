import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { spawnSync } from 'node:child_process';
import { stripTypeScriptTypes } from 'node:module';

const root = process.argv[2] || path.resolve(import.meta.dirname, '..');
const source = fs.readFileSync(path.join(root, 'goLiveBypass/vpn-windows.ts'), 'utf8');
const quoteStart = source.indexOf('function quotePowerShell(');
const quoteEnd = source.indexOf('function normalizedPath(', quoteStart);
const scriptStart = source.indexOf('function serviceScript(');
const scriptEnd = source.indexOf('function elevatedPowerShellArgs(', scriptStart);
assert(quoteStart > 0 && quoteEnd > quoteStart && scriptStart > 0 && scriptEnd > scriptStart);
const context = vm.createContext({});
vm.runInContext(stripTypeScriptTypes(source.slice(quoteStart, quoteEnd) + source.slice(scriptStart, scriptEnd)), context);
const executable = 'C:\\WireSock\\wiresock-client.exe';
const config = 'C:\\Private\\wiresock-discord.conf';
const expected = `"${executable}" service -config "${config}" -log-level info -network-lock disabled`;
const script = context.serviceScript(executable, config);
const shell = process.platform === 'win32' ? 'powershell.exe' : 'pwsh';
for (const scenario of ['foreign', 'own', 'stopped', 'changed-before-retarget']) {
    const command = scenario === 'foreign' ? '"C:\\Other\\vpn.exe" -config "C:\\Other\\vpn.conf"' : expected;
    const mock = `
$script:reads=0
function Get-Service {
    $service=[PSCustomObject]@{Status=${context.quotePowerShell(scenario === 'stopped' ? 'Stopped' : 'Running')}}
    $service | Add-Member ScriptMethod WaitForStatus {}
    return $service
}
function Get-CimInstance {
    $script:reads++
    $command=${context.quotePowerShell(command)}
    if(${context.quotePowerShell(scenario)} -eq 'changed-before-retarget' -and $script:reads -gt 1){$command='other-profile'}
    return [PSCustomObject]@{PathName=$command;State=${context.quotePowerShell(scenario === 'stopped' ? 'Stopped' : 'Running')}}
}
function Stop-Service { [Console]::Out.WriteLine('STOP') }
function Invoke-CimMethod { [Console]::Out.WriteLine('CHANGE'); return [PSCustomObject]@{ReturnValue=0} }
function Start-Service { [Console]::Out.WriteLine('START') }
`;
    const result = spawnSync(shell, ['-NoProfile', '-NonInteractive', '-EncodedCommand', Buffer.from(mock + script, 'utf16le').toString('base64')], { encoding: 'utf8', timeout: 15000 });
    assert.ifError(result.error);
    if (scenario === 'foreign' || scenario === 'changed-before-retarget') {
        assert.equal(result.status, 1, `${scenario}: conflicting profile must be rejected`);
        assert.match(result.stderr, /outro perfil/);
        assert(!result.stdout.includes('CHANGE') && !result.stdout.includes('START'));
        if (scenario === 'foreign') assert(!result.stdout.includes('STOP'), 'Foreign service must not be stopped');
    } else {
        assert.equal(result.status, 0, `${scenario}: owned/stopped service must still activate: ${result.stderr}`);
        assert(result.stdout.includes('CHANGE') && result.stdout.includes('START'));
    }
}
console.log('PASS: elevated activation rechecks service ownership before stopping or retargeting.');
