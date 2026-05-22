# Backend Assessment — HTTP Service

A production-quality Go HTTP service implementing two parts:

1. **Rate-Limited API** — per-user rate limiting with fixed 1-minute windows
2. **Product Catalog** — in-memory product/media store with split data model

Built with Go standard library (`net/http`) — no frameworks.

---

## 1. How to Run

```bash
# From the project root:

# Build and run
go build -o server ./cmd/server
./server

# Or run directly
go run ./cmd/server
```

The server starts on **port 8080** by default.

---

## 2. Example curl Commands

### POST /request — Submit a request for a user

```bash
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-001", "payload": {"action": "apply", "job_id": "job-123"}}'
```

### GET /stats — View per-user rate limit stats

```bash
curl http://localhost:8080/stats
```

### POST /products — Create a product with media

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Wireless Headphones",
    "sku": "WH-1000XM5",
    "image_urls": [
      "https://cdn.example.com/products/wh-1000xm5/img-1.jpg",
      "https://cdn.example.com/products/wh-1000xm5/img-2.jpg"
    ],
    "video_urls": [
      "https://cdn.example.com/products/wh-1000xm5/vid-1.mp4"
    ]
  }'
```

### GET /products — List products (paginated)

```bash
# Default pagination (limit=20, offset=0)
curl http://localhost:8080/products

# Custom pagination
curl "http://localhost:8080/products?limit=5&offset=10"
```

### GET /products/:id — Get a single product with full media details

```bash
curl http://localhost:8080/products/<product-id>
```

### POST /products/:id/media — Append media to an existing product

```bash
curl -X POST http://localhost:8080/products/<product-id>/media \
  -H "Content-Type: application/json" \
  -d '{
    "image_urls": [
      "https://cdn.example.com/products/wh-1000xm5/img-3.jpg"
    ],
    "video_urls": []
  }'
```

### Duplicate SKU — returns 409 Conflict

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name": "Duplicate", "sku": "WH-1000XM5", "image_urls": [], "video_urls": []}'
```

---

## 3. POST /request Response (201 Created)

A successful request returns HTTP **201 Created** rather than 200 OK. The 201 status code explicitly communicates that the request was accepted and processed (a resource/state change occurred within the rate limiter). This follows REST semantics for successful creation-like operations.

```json
{
  "message": "request accepted",
  "user_id": "user-001",
  "timestamp": "2026-05-22T10:30:00Z"
}
```

---

## 4. Rate Limiting Approach

**Algorithm:** Fixed 1-minute window

**Why fixed window over sliding window:**
- Simpler to reason about and implement correctly
- Easier to compute and document `retry_after_seconds` — the caller knows exactly how long until the window resets
- Lower overhead: no need to track per-request timestamps in a sliding log
- Predictable behavior: limits reset at fixed boundaries

**Mechanism:**

```
UserBucket {
  mu:          sync.Mutex     // per-user lock
  Count:       int            // accepted count in current window
  WindowStart: time.Time      // start of current window
  Rejected:    int            // cumulative rejected count (all time)
}
```

- Each POST /request locks the user's individual `UserBucket.mutex`
- The top-level `map[string]*UserBucket` is protected by `sync.RWMutex`
- Two goroutines hitting the same user simultaneously cannot exceed 5 accepts (checked under the per-user lock)

---

## 5. Stats Schema

```json
{
  "users": {
    "<user_id>": {
      "accepted_current_window": 3,
      "rejected_cumulative": 7
    }
  }
}
```

- `accepted_current_window`: number of requests accepted in the current 1-minute window (resets when the window expires)
- `rejected_cumulative`: total rejected requests *across all time* — this counter never resets, not even when the window expires

The cumulative rejected count allows operators to detect persistent abuse patterns even if the user has stopped sending requests recently.

---

## 6. Duplicate SKU (409 Conflict)

A duplicate SKU returns HTTP **409 Conflict** rather than 400 Bad Request.

- **400 Bad Request** is for malformed input (missing fields, invalid types, bad URLs)
- **409 Conflict** is appropriate here because the request is well-formed but conflicts with existing state (the SKU uniqueness constraint)

This follows HTTP semantics more precisely and makes error handling cleaner for API clients.

---

## 7. GET /products List Schema

```json
{
  "products": [
    {
      "id": "abc123def456",
      "name": "Wireless Headphones",
      "sku": "WH-1000XM5",
      "image_count": 2,
      "video_count": 1,
      "thumbnail_url": "https://cdn.example.com/products/wh-1000xm5/img-1.jpg",
      "created_at": "2026-05-22T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

**What is excluded and why:**

- `image_urls` and `video_urls` arrays are **never serialized** in list responses
- Rationale: list queries must be fast and avoid touching the media map
- The list response only reads from `map[string]*Product` (flat struct without URLs)
- Detail queries (`GET /products/:id`) read both maps and include the full URL arrays

This split-data-model design ensures list performance degrades only with product count, not with total media volume.

---

## 8. URL Validation Rules

- Must start with `http://` or `https://`
- Maximum length of **2048 characters**
- Must not be empty
- No duplicate URLs within the same request (checked via a `map[string]bool`)

---

## 9. Maximum URLs Per Request

- Maximum **20 image URLs** per request
- Maximum **20 video URLs** per request

These limits apply to both `POST /products` and `POST /products/:id/media`.

---

## 10. Data Model Explanation

### In-Memory Storage

Products and media are stored in **separate maps**:

```
products: map[string]*Product     // id -> product (no URLs)
media:    map[string]*MediaStore  // id -> { image_urls, video_urls }
skuIndex: map[string]string       // sku -> id
```

- `Product` is a flat struct with count fields (`ImageCount`, `VideoCount`) and a `ThumbnailURL` — it **never contains URL arrays**
- `MediaStore` holds the actual URL arrays, keyed by the same product ID
- **List query** (`GET /products`): reads only the `products` map — O(n) where n = number of products, never touches media
- **Detail query** (`GET /products/:id`): reads both maps — 2 map lookups total

### With PostgreSQL

In a production database, the equivalent design would be:

```sql
CREATE TABLE products (
    id           UUID PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    sku          VARCHAR(255) UNIQUE NOT NULL,
    image_count  INT DEFAULT 0,
    video_count  INT DEFAULT 0,
    thumbnail_url TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE product_media (
    id       UUID PRIMARY KEY,
    product_id UUID REFERENCES products(id),
    type     VARCHAR(5) CHECK (type IN ('image', 'video')),
    url      TEXT NOT NULL
);
```

- **List query**: `SELECT id, name, sku, image_count, video_count, thumbnail_url, created_at FROM products LIMIT $1 OFFSET $2` — no JOIN on media
- **Detail query**: `SELECT * FROM products WHERE id = $1` + `SELECT url, type FROM product_media WHERE product_id = $1`
- CDN URLs stored as `TEXT` columns — indexed by product_id for fast media lookups

---

## 11. Production Limitations (Part 1 — Rate Limiter)

- **Single instance only** — rate limiter state lives in process memory (`map[string]*UserBucket`)
- **Restart loses all state** — counters reset to zero on restart
- **Multi-instance deployment breaks rate limiting** — each instance has independent state, so a user could send 5 requests to instance A and 5 to instance B, exceeding the intended 5/min limit
- **Fix**: Use **Redis** with `INCR` + `EXPIREAT` for distributed rate limiting:
  ```go
  key := "ratelimit:" + userID + ":" + currentWindowMinute()
  count := redis.Incr(key)
  if count == 1 {
      redis.Expireat(key, endOfCurrentMinute())
  }
  if count > 5 {
      // reject
  }
  ```
  This approach is atomic, survives restarts, and works across instances.

---

## 12. AI Tools Disclosure

- **Claude (Anthropic)**: Used for scaffolding the project structure, writing boilerplate for handler routing, and generating initial README drafts
- **What I wrote/reviewed myself**:
  - Rate limiter concurrency logic (double-checked locking pattern, per-user mutex design)
  - Product store atomicity guarantees (RW mutex granularity)
  - All validation rules and error handling
  - Data model separation (products vs media maps)
  - Thread-safety verification for concurrent access patterns

---

## Project Structure

```
cmd/
  server/
    main.go              # Entry point — wires dependencies, starts server
internal/
  ratelimiter/
    limiter.go           # RateLimiter with per-user mutex buckets
  catalog/
    store.go             # ProductStore with split product/media maps
    models.go            # Product, MediaStore, request/response types
  handlers/
    request.go           # POST /request, GET /stats
    products.go          # All /products/* handlers
    helpers.go           # Shared writeJSON utility
  validation/
    urls.go              # URL and string field validation
README.md
```
