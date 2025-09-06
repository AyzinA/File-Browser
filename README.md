# Secure File Browser

A lightweight, self-hosted file browser with TLS support, written in Go.  
It allows users to browse and download files from a specified directory through a simple web interface.

---

## Features

- 📂 Browse directories and files from a given root.
- 🔍 Search files by name.
- ⬇️ Download files directly via browser.
- 🏷️ Sorting (by name, size, modified date).
- 🍞 Breadcrumb navigation.
- 🛡️ TLS support with self-signed certificate generation.
- 🎨 Minimal responsive UI with dark theme.

---

## Project Structure

```
.
├── Dockerfile              # Multi-stage build (Go → distroless image)
├── docker-compose.yml      # Container orchestration
├── go.mod                  # Go module definition
├── main.go                 # File browser implementation
├── generate_cert.go        # Self-signed certificate generator
├── templates/              # HTML templates
│   └── index.html
├── static/                 # Static assets (CSS)
│   └── style.css
├── .env                    # Default environment variables
└── certs/                  # TLS certificates (generated at runtime)
```

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/yourname/file-browser.git
cd file-browser
```

### 2. Generate certificates (optional if TLS is enabled)

```bash
go run generate_cert.go -hosts "localhost,127.0.0.1" -out certs
```

This will create:

- `certs/cert.pem`
- `certs/key.pem`

### 3. Build & Run with Docker

```bash
docker-compose up --build
```

The file browser will be available at:  
👉 [http://localhost:8080](http://localhost:8080)

---

## Environment Variables

| Variable       | Default Value             | Description                                |
|----------------|---------------------------|--------------------------------------------|
| `ROOT_DIR`     | `/data`                   | Root directory to serve files from          |
| `HOST`         | `0.0.0.0`                 | Server bind address                        |
| `PORT`         | `8080`                    | Port for HTTP(S)                           |
| `USE_TLS`      | `false`                   | Enable TLS (`true`/`false`)                |
| `CERT_FILE`    | `certs/cert.pem`          | TLS certificate file                       |
| `KEY_FILE`     | `certs/key.pem`           | TLS private key file                       |
| `UPLOAD_DIR`   | `../sfd/data/uploads`     | Directory mounted inside container          |

---

## Usage

- Navigate the directory tree.
- Use the search bar to filter by file names.
- Click **Download** next to any file to save it.
- Click 📁 folder names to enter directories.
- Use breadcrumbs or 🏠 **Root** button to navigate back.

---

## Development

Run locally without Docker:

```bash
go run main.go
```

By default, it serves files from `./data` at `http://localhost:8080`.

---

## Security Notes

- Always restrict `ROOT_DIR` to a safe directory.
- For production, enable TLS:
  ```bash
  USE_TLS=true
  ```
- Certificates can be generated with `generate_cert.go` or provided externally.

---

## License

MIT — feel free to use and adapt.
