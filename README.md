# unifeed

unifeed is a Go-based RSS aggregation service. It allows you to configure Mastodon and Bluesky user timelines via a YAML config file and exposes them as unified RSS feeds through a simple HTTP API.

## Features
- Multi-source support (Mastodon/Bluesky)
- Unified endpoint: `/feeds/{name}` returns the configured user's timeline as RSS
- Easy to extend and customize

## Example Config File (config.yaml)
```yaml
Feeds:
  - Name: mastodon-demo
    Mastodon:
      Host: https://mastodon.social
      Token: your-mastodon-token
  - Name: bluesky-demo
    Bluesky:
      Host: https://bsky.social
      Handle: user.bsky.social
      AppKey: your-app-key
      AppSecret: your-app-secret
```

## Usage
1. Prepare your config file (e.g., `config.yaml`) and fill in the required tokens/secrets.
2. Start the service with the config file path specified by the butterfly framework flag:
   ```sh
   ./unifeed -config.file.path=config.yaml
   ```
3. Access the feeds:
   - Mastodon: `GET /feeds/mastodon-demo`
   - Bluesky: `GET /feeds/bluesky-demo`

## Testing
```sh
go test ./test/...
```

## Contribution & Extension
- To support more platforms, refer to the `internal/service` directory for extension patterns.
- Issues and PRs are welcome!
