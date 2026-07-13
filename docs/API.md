# API Documentation

## Response Format

All API responses follow this structure:

| Field | Type | Description |
|-------|------|-------------|
| `success` | bool | Always present. `true` for success, `false` for error. |
| `data` | any | Response payload. `null` on errors. |
| `meta` | object or null | Pagination metadata. `null` for single-resource responses. |
| `error` | object or omitted | Error details. Present only on errors. |

### Pagination

Paginated list endpoints return `meta`:

| Field | Type | Description |
|-------|------|-------------|
| `page` | int | Current page number |
| `per_page` | int | Items per page |
| `total` | int | Total items across all pages |
| `total_pages` | int | Total number of pages |

### Error Codes

| HTTP | Code | Description |
|------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request body or parameters |
| 401 | `UNAUTHORIZED` | Missing, invalid, or expired token |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 403 | `ACCOUNT_LOCKED` | Account temporarily locked |
| 403 | `EMAIL_NOT_VERIFIED` | Email not verified |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Duplicate or state conflict |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Base URL

```
http://localhost:8080
```

## Swagger UI

Interactive API docs are available at:

```
http://localhost:8080/swagger/index.html
```

To regenerate swagger docs after modifying annotations:

```bash
make swagger
```

## Authentication

Most `/api/v1/` endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <token>
```

### Token Flow

1. **Register** or **Login** to get an access token and refresh token
2. Use the access token in the `Authorization` header for protected endpoints
3. When the access token expires, call `/auth/refresh` with the refresh token
4. Refresh tokens are single-use and stored in the database

### Token Lifetimes

| Token | Lifetime | Storage |
|-------|----------|---------|
| Access Token | 15 minutes | Client (localStorage/cookie) |
| Refresh Token | 7 days | Database (opaque, hashed) |

## Responses

### Success

```json
{
  "success": true,
  "data": { ... },
  "meta": null
}
```

### Error

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

## Endpoints

### Health Check

```
GET /health
```

Response:
```json
{"status": "ok"}
```

### Readiness Check

```
GET /ready
```

Response:
```json
{"status": "ready"}
```

---

### Authentication

#### Register

```
POST /api/v1/auth/register
```

Request:
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

Response (201):
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

#### Login

```
POST /api/v1/auth/login
```

Request:
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

Response (200):
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

#### Refresh Token

```
POST /api/v1/auth/refresh
```

Request:
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4"
}
```

Response (200):
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "bmV3IHJlZnJlc2ggdG9rZW4",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

#### Logout

```
POST /api/v1/auth/logout
```

Request:
```json
{
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4"
}
```

Response (200):
```json
{
  "success": true,
  "data": {
    "message": "logged out successfully"
  }
}
```

#### Get Current User

```
GET /api/v1/auth/sessions/me
```

Requires `Authorization: Bearer <token>`.

Response (200):
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "name": "John Doe",
    "is_active": true
  }
}
```

#### Logout All Sessions

```
POST /api/v1/auth/sessions/logout-all
```

Requires `Authorization: Bearer <token>`.

Revokes all refresh tokens for the authenticated user.

Response (200):
```json
{
  "success": true,
  "data": {
    "message": "all sessions terminated"
  }
}
```

---

### Todos

All todo endpoints require `Authorization: Bearer <token>`.

#### Create Todo

```
POST /api/v1/todos
```

Request:
```json
{
  "title": "Buy groceries",
  "description": "Milk, eggs, bread"
}
```

Response (201):
```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Buy groceries",
    "description": "Milk, eggs, bread",
    "completed": false,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### List Todos

```
GET /api/v1/todos?page=1&per_page=20
```

Response (200):
```json
{
  "success": true,
  "data": [...],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

#### Get Todo

```
GET /api/v1/todos/{id}
```

Response (200):
```json
{
  "success": true,
  "data": {
    "id": "...",
    "title": "...",
    "description": "...",
    "completed": false,
    "created_at": "...",
    "updated_at": "..."
  }
}
```

#### Update Todo

```
PUT /api/v1/todos/{id}
```

Request:
```json
{
  "title": "Updated title",
  "description": "Updated description"
}
```

#### Delete Todo

```
DELETE /api/v1/todos/{id}
```

Response (200):
```json
{"success": true}
```

#### Complete Todo

```
PATCH /api/v1/todos/{id}/complete
```

#### Search Todos

```
GET /api/v1/todos/search?q=groceries&page=1&per_page=20
```

---

### Authorization (RBAC)

All authorization endpoints require `Authorization: Bearer <token>`.

#### Roles

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/roles | Create a role |
| GET | /api/v1/auth/roles | List roles |
| GET | /api/v1/auth/roles/{id} | Get a role |
| PUT | /api/v1/auth/roles/{id} | Update a role |
| DELETE | /api/v1/auth/roles/{id} | Delete a role |

#### Permissions

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/permissions | Create a permission |
| GET | /api/v1/auth/permissions | List permissions |
| GET | /api/v1/auth/permissions/{id} | Get a permission |
| PUT | /api/v1/auth/permissions/{id} | Update a permission |
| DELETE | /api/v1/auth/permissions/{id} | Delete a permission |

#### User-Role Assignments

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/users/{userId}/roles | Assign role to user |
| DELETE | /api/v1/auth/users/{userId}/roles/{roleId} | Remove role from user |
| GET | /api/v1/auth/users/{userId}/roles | Get user roles |

#### Role-Permission Assignments

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/roles/{roleId}/permissions | Assign permission to role |
| DELETE | /api/v1/auth/roles/{roleId}/permissions/{permissionId} | Remove permission from role |
| GET | /api/v1/auth/roles/{roleId}/permissions | Get role permissions |

#### Permission Check

```
POST /api/v1/auth/check-permission
```

Request:
```json
{
  "resource": "todos",
  "action": "create"
}
```

Response (200):
```json
{
  "success": true,
  "data": {
    "allowed": true
  }
}
```
