# Derperer

Derperer is mother of [derper](https://pkg.go.dev/tailscale.com/cmd/derper).

## Quick Start

First, you need to have a [fofa](https://fofa.info/) account

And fill your credentials in the `docker-compose.yml`

```yaml
- DERPERER_CONFIG_FOFACLIENT_EMAIL=xxx
- DERPERER_CONFIG_FOFACLIENT_KEY=xxx
```

Then start by

```bash
docker compose up -d
```

## Usage

### Speedtest

speedtest to one DERP node

```
Usage:
  derperer speedtest [flags]

Flags:
  -u, --derpMapUrl string     derp map url (default "https://controlplane.tailscale.com/derpmap/default")
  -r, --derpRegionId string   derp region id
  -d, --duration duration     duration (default 30s)
  -h, --help                  help for speedtest

Global Flags:
      --log-level string   log level (default "info")
```

Example:

```bash
derperer speedtest -r hkg
# 2024/07/04 11:51:31 derphttp.Client.Recv: connecting to derp-20 (hkg)
# 2024/07/04 11:51:33 derphttp.Client.Recv: connecting to derp-20 (hkg)
# 2024-07-04T11:52:05.290+0800    INFO    cmd/speedtest.go:64     bandwidth: 1.51Mbps, totalBytes: 5.69Mbytes, latency: 1.604274131s
```

### Server

Serve A derperer server, which auto collect DERP nodes and speedtest to them.

```
Usage:
  derperer server [flags]

Flags:
      --config.Address string                           address (default ":8080")
      --config.AdminToken string                        admin token
      --config.DERPMapPolicy.BaselineBandwidth float    bandwidth limit, unit: Mbps (default 2)
      --config.DERPMapPolicy.CheckDuration duration     check duration (default 5s)
      --config.DERPMapPolicy.RecheckInterval duration   update interval (default 1h0m0s)
      --config.DERPMapPolicy.TestConcurrency int        test concurrency (default 4)
      --config.DataPath string                          data path (default "./data")
      --config.FetchBatch int                           batch (default 100)
      --config.FetchInterval duration                   fetch interval (default 24h0m0s)
      --config.FofaClient.Email string                  fofa email
      --config.FofaClient.Key string                    fofa key
  -h, --help                                            help for server

Global Flags:
      --log-level string   log level (default "info")
```

There are three main routes:

- `/derp.json` return a DERP map like https://controlplane.tailscale.com/derpmap/default
- `/swagger/index.html` swagger ui
- `/admin` Admin page, available when `--config.AdminToken` is set

## Configuration

There are three way to pass configuration to derperer:

1. Command line flags, shown in `derperer -h`

2. Config file, the derperer will try to find `derperer.yaml` in `$PWD` or `$HOME/.derperer/`, you can fill your config like:

```yaml
config:
  FofaClient:
    Email: xxx
    Key: xxx
  AdminToken: xxx
  DataPath: /data
```

The key is the same as the flag name

3. Environment variables, the derperer will try to find `DERPERER_` prefix environment variables, corresponding to the flag name. for example`config.FofaClient.Key` will be `DERPERER_CONFIG_FOFACLIENT_KEY`

## How it works

The derperer will search DERP nodes from [fofa](https://fofa.info/) by fingerprint `"<h1>DERP</h1>"`.
Then it will speedtest to each DERP node using official [derp](https://pkg.go.dev/tailscale.com/derp) binary.

The speedtest result will be saved in `data` directory, and the DERP map will be served at `/derp.json`.

## What is derper

The derper binary is a simple DERP server.

You can learn more about DERP here:

- https://tailscale.com/blog/how-tailscale-works#encrypted-tcp-relays-derp
- https://tailscale.com/kb/1232/derp-servers