# GoLiveBypass para Vesktop

Fork de [bezumiya/GoLiveBypass](https://github.com/bezumiya/GoLiveBypass), restrito ao plugin e à instalação no **Vesktop**. Windows x64 e Linux x64.

## Outros usos e macOS

Para qualquer uso do GoLiveBypass que não seja o plugin para Vesktop, acesse o [repositório original](https://github.com/bezumiya/GoLiveBypass) e siga as orientações de lá.

**macOS:** baixe o [GoLiveBypass macOS 0.8.0 (DMG universal) no projeto original](https://github.com/bezumiya/GoLiveBypass/releases/tag/macos-v0.8.0). O aplicativo para macOS e suas orientações são mantidos no repositório original.

## Instalar

Baixe o [Vesktop no site oficial](https://vesktop.dev/), caso ainda não esteja instalado, e conecte-se à sua conta Discord. Execute o comando como seu usuário normal. O instalador fecha o Vesktop, instala o build pronto, preserva suas configurações e abre o app com o GoLiveBypass ativado. Depois, conclua o assistente Proton/WireGuard dentro do Vesktop.

**Windows — cole no PowerShell:**

```powershell
irm https://raw.githubusercontent.com/yagoraimon/GoLiveBypass-Vesktop/main/install.ps1 | iex
```

**Linux — cole no terminal:**

```bash
curl -fsSL https://raw.githubusercontent.com/yagoraimon/GoLiveBypass-Vesktop/main/install.sh | bash
```

Não precisa instalar Git, Node, pnpm ou Go. O download é conferido por SHA-256 antes de executar. Não execute com `sudo` nem como administrador: a autorização para configurar a VPN é solicitada pelo plugin depois da instalação.

## O que muda

- O instalador inclui um build do Vencord com o GoLiveBypass. O arquivo `app.asar` do Vesktop permanece intacto.
- O caminho do build é salvo em **`state.json`**, que é o arquivo lido pelo Vesktop. Não procura nem gera `vesktop.asar`.
- As configurações existentes são mescladas. O GoLiveBypass é ativado; login e rota continuam para o assistente do plugin.
- Build e backups ficam em `%LOCALAPPDATA%\GoLiveBypass-Vesktop` (Windows) ou `${XDG_DATA_HOME:-~/.local/share}/golivebypass-vesktop` (Linux).
- Para atualizar, execute novamente o mesmo comando. O updater que recompila fontes do projeto original fica desativado neste fork.

O build anterior e suas configurações permanecem salvos. Outros plugins personalizados que você compilou por conta própria não são incluídos neste build; os plugins padrão do Vencord continuam disponíveis.

## Instalações atendidas

Windows: instalador padrão do Vesktop, instalação por máquina e portable em execução. Linux: pacote nativo ou AppImage em execução/instalado em `~/Applications`. O Vesktop e o perfil são detectados automaticamente. Instalações personalizadas podem usar `--exe` e `--data-dir` no executável da release.

Flatpak e Snap não são atendidos: o motor VPN do plugin precisa executar ferramentas do host fora do sandbox. Use Vesktop nativo/AppImage. A instalação do plugin não elimina a necessidade de uma conta Proton ou de uma configuração WireGuard válida, nem garante disponibilidade de servidores ou conectividade da rede.

## Desenvolvimento e distribuição

O plugin deriva do commit de origem registrado em `upstream.json`. O Vencord é fixado em um commit, e as dependências usam o lockfile desse commit. O workflow executa testes em Windows e Linux, gera o helper Proton e compila os instaladores com o payload embutido. As máquinas dos usuários só executam os instaladores prontos.

```bash
cd installer
go test ./...
```

Na raiz, com Node, pnpm e Go disponíveis: `node scripts/build.mjs`. Os executáveis são gerados em `release/`. Publicações usam tags `vesktop-v1.x.x` e o workflow `release.yml`.

## Créditos e licença

GoLiveBypass: bezumiya e colaboradores. Vencord e Vesktop: Vendicated e colaboradores. As atribuições e licenças originais são preservadas. Este fork é independente e não representa os mantenedores originais. GPL-3.0-or-later; consulte `LICENSE` e as licenças das dependências.
