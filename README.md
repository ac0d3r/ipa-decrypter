# ipa-decrypter

Yet another auto-ipa-decrypter tool (Powered by [ipatool](https://github.com/majd/ipatool) & [fairplay-decrypt](https://github.com/EthanArbuckle/fairplay-decrypt), with [unfaird](https://github.com/lbr77/unfaird) as a device-based fallback).

Search apps by name or Bundle ID in the browser; the server downloads the encrypted IPA from the App Store via ipatool, then decrypts it locally using the account's FairPlay credential (`credential.bin`), or on a jailbroken iOS device running unfaird when no credential is available. It ships with a download queue, live progress, a local cache, user accounts (invite-code registration) and multiple App Store storefront accounts. UI inspired by [decrypt.34306.lol](https://decrypt.34306.lol/).

## Requirements

- Docker (recommended), or Go 1.27 + Bun to build from source (plus `make build-transform` for local decryption — the Docker image already ships the transform helper)
- A FairPlay `credential.bin` for each Apple ID (extracted from a jailbroken device, see [fairplay-decrypt](https://github.com/EthanArbuckle/fairplay-decrypt)), or alternatively a jailbroken iOS device running [unfaird](https://github.com/lbr77/unfaird), on the same LAN as the server (or otherwise reachable)
- At least one Apple ID with access to the App Store

> Multiple accounts / storefronts are supported (the matching logged-in account is picked automatically per storefront), but only a single-account setup has been tested so far.

## Installation

```bash
# Option 1 — use the published image (a new one is pushed on every v* tag):
docker pull ghcr.io/ac0d3r/ipa-decrypter:latest
docker tag ghcr.io/ac0d3r/ipa-decrypter:latest ipa-decrypter:latest
docker compose up -d

# Option 2 — build the image locally, then run:
docker compose build
docker compose up -d
```

Once running:

- LAN: `http://<ip>:8080`
- Admin console: `http://<ip>:8080/admin` (default credentials `admin / admin`; change the password under "Users" after first login)

### Jailbroken device (optional)

A jailbroken device is needed only to extract the FairPlay credential for local decryption, or as a decryption fallback when no credential is available.

```bash
make build-unfaird                               # build the unfaird deb (requires Theos)
make install-unfaird DEVICE=root@<ip>:<port>      # install to the device over SSH (port optional)
```

Then add the device in the admin console under "Devices" (host + unfaird port, default `6347`). Once the health check passes, you're all set. For local decryption instead, extract `credential.bin` per [fairplay-decrypt](https://github.com/EthanArbuckle/fairplay-decrypt) and upload it in the admin console under "Accounts".

## Configuration

For Docker deployments, settings live in `.env` (see [.env.example](.env.example)):

| Variable | Default | Purpose |
|---|---|---|
| `HOST_PORT` | `8080` | Port exposed on the host |
| `DATA_DIR` | `./data` | Data directory (database, accounts, IPA files) — back this up and you're covered |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / `admin` | Admin account bootstrapped on first run |
| `KEYCHAIN_PASSPHRASE` | `ipatool-ipa-decrypter` | Passphrase for the ipatool keyring |

## Development

```bash
make deps        # install dependencies
make build-all   # build backend + frontend
make build-transform  # build the FairPlay transform helper into bin/ (local decryption)
make dev-web     # frontend dev server (proxies /api -> :8080)
make ci          # gofmt + go vet + go test
```

## Legal note

Use this project for research only, and only with apps you are licensed to. Use of the software and of your Apple ID remains your responsibility. The project is provided as-is, without warranty.

## License

[MIT](LICENSE)

## Credits

- [ipatool](https://github.com/majd/ipatool) — App Store authentication & IPA downloads
- [unfaird](https://github.com/lbr77/unfaird) — decryption daemon on the jailbroken device
- [fairplay-decrypt](https://github.com/EthanArbuckle/fairplay-decrypt) — Offline FairPlay App Store IPA decryption with independent key derivation
- UI inspired by [decrypt.34306.lol](https://decrypt.34306.lol/)
