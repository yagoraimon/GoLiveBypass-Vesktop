import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync, execSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const upstream = JSON.parse(fs.readFileSync(path.join(root, 'upstream.json')));
const version = fs.readFileSync(path.join(root, 'installer/main.go'), 'utf8').match(/var version = "([^"]+)"/)[1];
if (process.env.GITHUB_REF?.startsWith('refs/tags/')) assert.equal(process.env.GITHUB_REF, `refs/tags/v${version}`, 'Tag and installer version must match');
const win = process.platform === 'win32';
assert(process.arch === 'x64' && (win || process.platform === 'linux'), 'Windows/Linux x64 required');
const vcs = path.join(root, '.build', 'Vencord');
const run = (cmd, args, cwd = root, extra = {}) => execFileSync(cmd, args, { cwd, stdio: 'inherit', env: { ...process.env, ...extra } });
fs.mkdirSync(path.dirname(vcs), { recursive: true });
if (!fs.existsSync(path.join(vcs, '.git'))) {
    fs.mkdirSync(vcs, { recursive: true });
    run('git', ['init'], vcs);
    run('git', ['remote', 'add', 'origin', 'https://github.com/Vendicated/Vencord.git'], vcs);
    run('git', ['fetch', '--depth', '1', 'origin', upstream.vencordCommit], vcs);
    run('git', ['checkout', '--detach', 'FETCH_HEAD'], vcs);
}
assert.equal(execFileSync('git', ['rev-parse', 'HEAD'], { cwd: vcs, encoding: 'utf8' }).trim(), upstream.vencordCommit);
const plugin = path.join(vcs, 'src', 'userplugins', 'goLiveBypass');
fs.cpSync(path.join(root, 'goLiveBypass'), plugin, { recursive: true });
execSync('pnpm install --frozen-lockfile', { cwd: vcs, stdio: 'inherit' });
execSync('pnpm build --standalone --disable-updater', { cwd: vcs, stdio: 'inherit' });

const payload = path.join(root, 'installer', 'payload');
fs.mkdirSync(payload, { recursive: true });
for (const file of fs.readdirSync(path.join(vcs, 'dist'))) {
    if (file.startsWith('vencordDesktop') && !file.endsWith('.map')) fs.copyFileSync(path.join(vcs, 'dist', file), path.join(payload, file));
}
const helper = path.join(payload, 'bin', win ? 'win32-x64' : 'linux-x64', win ? 'proton-confgen.exe' : 'proton-confgen');
fs.mkdirSync(path.dirname(helper), { recursive: true });
run('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', helper, './cmd/protonvpn-wg'], path.join(root, 'tools', 'proton-confgen'), { CGO_ENABLED: '0' });
fs.copyFileSync(path.join(vcs, 'LICENSE'), path.join(payload, 'Vencord-LICENSE'));
fs.copyFileSync(path.join(root, 'LICENSE'), path.join(payload, 'GoLiveBypass-LICENSE'));
fs.copyFileSync(path.join(root, 'tools', 'proton-confgen', 'LICENSE'), path.join(payload, 'Proton-helper-LICENSE'));
fs.copyFileSync(path.join(root, 'upstream.json'), path.join(payload, 'upstream.json'));
fs.writeFileSync(path.join(payload, 'package.json'), '{}\n');
assert(fs.readFileSync(path.join(payload, 'vencordDesktopRenderer.js'), 'utf8').includes('GoLiveBypass'), 'Plugin missing from bundle');
assert(fs.statSync(helper).size > 100_000, 'Helper missing');
const release = path.join(root, 'release');
fs.mkdirSync(release, { recursive: true });
const asset = win ? 'golive-vesktop-windows-x64.exe' : 'golive-vesktop-linux-x64';
const binary = path.join(release, asset);
run('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', binary, '.'], path.join(root, 'installer'), { CGO_ENABLED: '0' });
const hash = createHash('sha256').update(fs.readFileSync(binary)).digest('hex');
fs.writeFileSync(binary + '.sha256', `${hash}  ${asset}\n`);
console.log(`Ready: ${asset} SHA-256 ${hash}`);
