# Go REST Starter

A comprehensive Go REST API starter kit featuring authentication, database migrations, Docker support, and various utility packages. This project is designed to jumpstart your Go backend development with a solid foundation.

### By [Md Hasib](https://github.com/mdhasib01)

## Features

- **REST API**: Built with `gorilla/mux`.
- **Database**: PostgreSQL integration using `lib/pq`.
- **Migrations**: Database schema management with `golang-migrate`.
- **Authentication**:
  - JWT support.
  - OAuth2 integration via `goth` (Google, LinkedIn, Facebook).
- **Documentation**: Swagger/OpenAPI documentation.
- **Docker**: Full Docker and Docker Compose support.
- **Utilities**:
  - Email notifications (SMTP).
  - GeoIP integration.
  - WebSocket support.
  - Cron jobs.
  - Excel file handling.

## Prerequisites

- [Go 1.24.6+](https://go.dev/dl/)
- [PostgreSQL](https://www.postgresql.org/)
- [Docker](https://www.docker.com/) (optional, for containerized deployment)

## Configuration

The application uses a `config.json` file for configuration.

1.  Create a `config.json` file in the root directory. You can use `config.docker.json` (if available) or the example below as a template.
2.  Update the fields with your credentials (database, OAuth keys, SMTP settings, etc.).

## Installation & Running

### Local Development

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/mdhasib01/go-rest-starter.git
    cd Starter-Server
    ```

2.  **Install dependencies:**

    ```bash
    go mod download
    ```

3.  **Run the application:**
    You can use the `Makefile` command:
    ```bash
    make run
    ```
    Or build and run manually:
    ```bash
    go build -o ./api/bloodlink-server
    ./api/bloodlink-server
    ```

### Docker

1.  **Build and start services:**
    ```bash
    docker-compose up --build -d
    ```
    This will start the PostgreSQL database and the Go server.

## Database Migrations

The project uses `golang-migrate` for database migrations.

- **Run Migrations (Up):**
  ```bash
  make migrate-up
  ```
- **Rollback Migrations (Down):**
  ```bash
  make migrate-down
  ```

_Note: Check the `Makefile` to ensure the database connection string in the `migrate` command matches your local PostgreSQL setup._

## API Documentation

Swagger documentation is available. Once the server is running, you can typically access the API docs at:

```
http://localhost:8765/swagger/index.html
```

_(Port depends on your `config.json` setting)_

## Project Structure

- `api/`: API entry point/build artifacts.
- `config/`: Configuration loading logic.
- `controller/`: Request handlers and business logic.
- `dao/`: Data Access Object layer (Database interactions).
- `docs/`: Swagger documentation files.
- `model/`: Data structures and models.
- `pkg/`: Shared packages (logger, notifications, etc.).
- `rest/`: REST API routing and middleware.
- `utils/`: General utility functions.
