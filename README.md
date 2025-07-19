# html-to-file-api

A fast, secure Go API for converting HTML to PNG or PDF files using headless Chromium (via chromedp), with JWT authentication. Designed for cloud-native deployment (e.g., Google Cloud Run) and easy containerization.

## Features

- Convert HTML to PNG or PDF via simple API endpoints
- JWT-based authentication for secure access
- Built with [Gin](https://github.com/gin-gonic/gin) and [chromedp](https://github.com/chromedp/chromedp)
- Ready for Docker and Google Cloud Run

## Live Demo

[Demo URL](https://html-to-file-api-1015515485383.asia-southeast1.run.app)

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

## Google Cloud Platform (GCP) CI/CD

This project is ready for automated deployment to Google Cloud Run using Google Cloud Build. The provided `cloudbuild.yaml` file defines the build and deployment steps.

### Prerequisites

- A Google Cloud project with billing enabled
- Cloud Build and Cloud Run APIs enabled
- Docker and gcloud CLI installed locally (for manual triggers)

### Setup

1. **Clone the repository and navigate to the project directory:**
   ```bash
   git clone https://github.com/yourusername/html-to-file-api.git
   cd html-to-file-api
   ```
2. **Set your GCP project:**
   ```bash
   gcloud config set project YOUR_PROJECT_ID
   ```
3. **(Optional) Authenticate Docker to GCR:**
   ```bash
   gcloud auth configure-docker
   ```

### Automated Build & Deploy

Cloud Build will:

- Build the Docker image
- Push it to Google Container Registry (GCR)
- Deploy to Cloud Run

To trigger a build and deploy (from the project root):

```bash
gcloud builds submit --config cloudbuild.yaml
# gcloud builds submit --config cloudbuild.yaml . --substitutions=_TAG="2025-07-19"
```

The `cloudbuild.yaml` is configured to:

- Build and tag the Docker image with the commit SHA
- Push the image to GCR
- Deploy to Cloud Run in the `asia-southeast1` region (edit as needed)
- Set memory, CPU, and timeout for large file conversions
- Allow unauthenticated access (can be changed for private APIs)

#### Environment Variables in Cloud Run

Set `JWT_SECRET_KEY` in the Cloud Run service configuration for production security.

For more details, see [Cloud Build documentation](https://cloud.google.com/build/docs) and [Cloud Run documentation](https://cloud.google.com/run/docs).

## Environment Variables

- `JWT_SECRET_KEY` (required): Secret key for signing/verifying JWTs
- `PORT` (optional): Port to run the server (default: 8080)

## License

-
