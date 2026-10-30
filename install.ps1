& {
    $ErrorActionPreference = 'Stop'
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    if (-not [Environment]::Is64BitOperatingSystem -or $env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') {
        throw 'Esta release atende Windows x64.'
    }
    $base = 'https://github.com/yagoraimon/GoLiveBypass-Vesktop/releases/latest/download'
    $asset = 'golive-vesktop-windows-x64.exe'
    $temporary = Join-Path ([IO.Path]::GetTempPath()) ('golive-vesktop-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $temporary | Out-Null
    try {
        $exe = Join-Path $temporary $asset
        Write-Host 'Baixando o plugin pronto para Vesktop...'
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $exe
        $checksum = (Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset.sha256").Content
        if ($checksum -is [byte[]]) { $checksum = [Text.Encoding]::UTF8.GetString($checksum) }
        $expected = (($checksum.Trim()) -split '\s+')[0]
        if ($expected -notmatch '^[a-fA-F0-9]{64}$' -or (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash -ne $expected) {
            throw 'SHA-256 nao confere. O instalador nao foi executado.'
        }
        & $exe
        if ($LASTEXITCODE -ne 0) { throw "Instalacao interrompida (codigo $LASTEXITCODE)." }
    } finally {
        # Only the uniquely created task directory is removed.
        if ($temporary -and (Split-Path -Parent $temporary) -eq ([IO.Path]::GetTempPath()).TrimEnd('\')) {
            Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}
