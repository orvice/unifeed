# Unifeed

Unifeed is an RSS Feed aggregation service that supports fetching timelines from Mastodon and Bluesky and converting them to RSS Feeds.

## Features

- Support for Mastodon and Bluesky timeline fetching
- Timeline to RSS Feed conversion
- AI-powered content summarization
- S3 storage support
- Scheduled updates
- Prometheus metrics monitoring

## Quick Start

### Configuration

Create a configuration file `config.yaml`:

```yaml
feeds:
  - name: mastodon-feed
    mastodon:
      host: https://mastodon.example.com
      token: your-access-token
  - name: bluesky-feed
    bluesky:
      host: https://bsky.social
      handle: your.bsky.handle
      app_key: your-app-key
      app_secret: your-app-secret
  - name: rss-feed
    rss_feed: https://example.com/feed.xml

s3:
  endpoint: s3.example.com
  access_key_id: your-access-key
  secret_access_key: your-secret-key
  use_ssl: true
  bucket_name: unifeed

ai:
  api_key: your-openai-api-key
  model: gpt-3.5-turbo
  max_tokens: 500
  temperature: 0.7

scheduler:
  update_interval: 5m
  max_retries: 3
  retry_delay: 5s
```

### Build

```bash
go build -o unifeed cmd/main.go
```

### Run

```bash
./unifeed -config config.yaml
```

## API Endpoints

### Get Feed

```
GET /feeds/{name}
```

Response example:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Feed Title</title>
    <link>https://example.com</link>
    <description>Feed Description</description>
    <item>
      <title>Item Title</title>
      <link>https://example.com/item</link>
      <description>Item Description</description>
      <pubDate>Wed, 21 Oct 2023 07:28:00 GMT</pubDate>
      <summary>AI generated summary of the item content...</summary>
    </item>
  </channel>
</rss>
```

### Manually Update Feed

```
POST /feeds/{name}/update
```

### Get Feed Status

```
GET /feeds/{name}/status
```

Response example:

```json
{
  "name": "feed-name",
  "last_update": "2023-10-21T07:28:00Z",
  "next_update": "2023-10-21T07:33:00Z",
  "status": "running",
  "error": null
}
```

## Monitoring Metrics

The service exposes the following Prometheus metrics:

- `feed_update_total`: Total number of feed updates
- `feed_update_duration_seconds`: Duration of feed updates
- `feed_items_total`: Total number of items in each feed
- `feed_cache_hits_total`: Total number of cache hits
- `feed_cache_misses_total`: Total number of cache misses
- `feed_cache_hit_ratio`: Cache hit ratio
- `feed_errors_total`: Total number of errors
- `ai_summary_total`: Total number of AI summaries generated
- `ai_summary_duration_seconds`: Duration of AI summary generation
- `s3_operation_total`: Total number of S3 operations
- `s3_operation_duration_seconds`: Duration of S3 operations

## Development

### Dependency Management

```bash
go mod tidy
```

### Testing

```bash
go test ./...
```

### Build Docker Image

```bash
docker build -t unifeed .
```

## License

MIT
