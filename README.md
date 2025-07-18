# html-to-file-api

A fast, secure Go API for converting HTML to PNG or PDF files using headless Chromium (via chromedp), with JWT authentication. Designed for cloud-native deployment (e.g., Google Cloud Run) and easy containerization.

## Features

- Convert HTML to PNG or PDF via simple API endpoints
- JWT-based authentication for secure access
- Built with [Gin](https://github.com/gin-gonic/gin) and [chromedp](https://github.com/chromedp/chromedp)
- Ready for Docker and Google Cloud Run

## Requirements

- Go 1.23+
- [Chromium](https://www.chromium.org/) (installed automatically in Docker)
- Set `JWT_SECRET_KEY` environment variable for authentication

## Getting Started

### 1. Clone and Build

```bash
git clone https://github.com/yourusername/html-to-file-api.git
cd html-to-file-api
go build -o html-to-file-api
```

### 2. Set Environment Variables

Create a `.env` file or export variables:

```env
JWT_SECRET_KEY=your_secret_key
PORT=8080 # optional, defaults to 8080
```

### 3. Run Locally

```bash
./html-to-file-api
```

## API Endpoints

### Health Check

```
GET /health
```

Response: `{ "message": "check" }`

### Convert HTML to PNG

```
POST /api/to-png
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "html": "<h1>Hello</h1>"
}
```

Response: PNG image (Content-Type: image/png)

### Convert HTML to PDF

```
POST /api/to-pdf
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "html": "<h1>Hello</h1>"
}
```

Response: PDF file (Content-Type: application/pdf)

## Authentication

All `/api/*` endpoints require a valid JWT in the `Authorization` header:

```
Authorization: Bearer <JWT_TOKEN>
```

Generate tokens using the built-in `GenerateJWT` function or your own tool, signed with your `JWT_SECRET_KEY`.

## Docker Usage

Build and run with Docker:

```bash
docker build -t html-to-file-api .
docker run -p 8080:8080 -e JWT_SECRET_KEY=your_secret_key html-to-file-api
```

## Google Cloud Run Deployment

This project includes a `cloudbuild.yaml` for CI/CD. Example deploy:

```bash
gcloud builds submit --config cloudbuild.yaml
```

## Environment Variables

- `JWT_SECRET_KEY` (required): Secret key for signing/verifying JWTs
- `PORT` (optional): Port to run the server (default: 8080)

## License

-
