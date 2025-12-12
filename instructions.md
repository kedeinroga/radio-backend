# Backend Development Specification: Radio Stream Proxy
> **Role:** Backend Engineer (Go)
> **Goal:** Build a robust Proxy API using Clean Architecture that serves radio stations to a Flutter app.
> **Output:** A REST JSON API running on port 8080.

## 1. Technical Stack & Standards
* **Language:** Go (Golang) 1.21+
* **Framework:** Gin Gonic (`github.com/gin-gonic/gin`)
* **HTTP Client:** Resty (`github.com/go-resty/resty/v2`)
* **Architecture:** Hexagonal (Ports & Adapters) / Standard Go Project Layout.
* **Logging:** `log/slog` (Structured JSON logging).
* **External Source:** Radio Browser API (`https://de1.api.radio-browser.info`).

## 2. The API Contract
You must expose the following endpoints returning the **exact** JSON structure defined below.

### 2.1 Endpoint: Get Popular Stations
* **Method:** `GET`
* **Path:** `/api/v1/stations/popular`
* **Query Params:** `limit` (int, default: 20), `country` (string).

### 2.2 Endpoint: Search Stations (NEW)
* **Method:** `GET`
* **Path:** `/api/v1/stations/search`
* **Query Params:**
    * `q` (string, required): The search term (name, tag, or country).
    * `limit` (int, default: 20).
* **Behavior:** Delegate the search to Radio Browser API (`/json/stations/search?name={q}`).

### 2.3 Response Schema (JSON)
Both endpoints return this structure.

```json
{
  "data": [
    {
      "id": "96181607-b3e3-4d6f-bd13-1a067576f332",
      "name": "Radio Paradise Main Mix",
      "stream_url": "[http://stream.radioparadise.com/mp3-192](http://stream.radioparadise.com/mp3-192)",
      "image_url": "[https://www.radioparadise.com/graphics/logo_flat.png](https://www.radioparadise.com/graphics/logo_flat.png)",
      "tags": ["eclectic", "rock", "pop"],
      "country": "US",
      "votes": 15400,
      "is_premium_only": false
    }
  ],
  "meta": {
    "count": 1,
    "user_type": "guest"
  }
}
```

## 3. Implementation Logic
### 3.1 Architecture Layers
Transport (internal/handlers):

GetPopular: Parses limit/country.

Search: Parses q param. Returns 400 if q is empty.

Service (internal/services):

ListPopular(): Calls Repository GetTopStations.

Search(term): Calls Repository SearchStations. Validates term length (min 3 chars recommended).

Repository (internal/repositories/radiobrowser):

Implement SearchStations(query string) calling the external API.

Mapping: Convert External DTO -> Domain Entity.

### 3.2 Authentication Strategy (Middleware)
Non-blocking: Both Popular and Search endpoints are public but must process the Authorization header to identify if the user is Guest or Premium (for future logic).

## 4. Deliverables Checklist
[ ] Server runs on port defined in .env.

[ ] Endpoint /api/v1/stations/popular works.

[ ] Endpoint /api/v1/stations/search?q=jazz works and returns filtered results.

[ ] Logs are structured (JSON).

[ ] CI/CD Pipeline (GitHub Actions) configured.
