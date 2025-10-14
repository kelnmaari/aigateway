# 🔐 Authentication API Documentation

Complete authentication API reference for Ollama-OpenAI Proxy (v1.3.0+).

## 📑 Table of Contents

- [Overview](#overview)
- [Authentication Endpoints](#authentication-endpoints)
  - [Register](#post-apiauthregister)
  - [Login](#post-apiauthlogin)
  - [Refresh Token](#post-apiauthrefresh)
  - [Logout](#post-apiauthlogout)
  - [Get Current User](#get-apiauthme)
- [User Profile Endpoints](#user-profile-endpoints)
- [Error Responses](#error-responses)

---

## Overview

The authentication system uses **JWT (JSON Web Tokens)** with:
- **Access Token**: Short-lived token for API access (default: 15 minutes)
- **Refresh Token**: Long-lived token for refreshing access tokens (default: 7 days)
- **Optional "Remember Me"**: Extended refresh token lifetime (24 hours)

### Token Usage

Include the access token in the `Authorization` header:
```http
Authorization: Bearer <access_token>
```

---

## Authentication Endpoints

### POST /api/auth/register

Register a new user account.

#### Request

**Headers:**
```http
Content-Type: application/json
```

**Body:**
```json
{
  "username": "string (required)",
  "email": "string (required)",
  "password": "string (required, min 8 chars)",
  "display_name": "string (optional)"
}
```

**Validation Rules:**
- **Username**: 3-32 characters, alphanumeric and underscores only
- **Email**: Valid email format
- **Password**: Minimum 8 characters, must include uppercase, lowercase, number, and special character

#### Response

**Status:** `201 Created`

```json
{
  "user": {
    "id": "user_xxxxxxxx",
    "username": "johndoe",
    "email": "john@example.com",
    "full_name": "John Doe",
    "status": "active",
    "is_active": true,
    "is_admin": false,
    "verified": false,
    "created_at": "2025-10-13T10:00:00Z",
    "updated_at": "2025-10-13T10:00:00Z"
  },
  "token": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_at": "2025-10-13T10:15:00Z",
    "token_type": "Bearer"
  },
  "personal_tenant": {
    "id": "tenant_xxxxxxxx",
    "name": "John Doe's Workspace",
    "slug": "johndoe",
    "type": "personal",
    "owner_id": "user_xxxxxxxx"
  }
}
```

#### Error Responses

**400 Bad Request** - Validation error:
```json
{
  "error": "username already taken",
  "field": "username"
}
```

**500 Internal Server Error** - Server error

#### Example

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecureP@ss123",
    "display_name": "John Doe"
  }'
```

---

### POST /api/auth/login

Authenticate user and receive tokens.

#### Request

**Headers:**
```http
Content-Type: application/json
```

**Body:**
```json
{
  "username": "string (required, or use email)",
  "email": "string (alternative to username)",
  "password": "string (required)",
  "remember_me": "boolean (optional, default: false)"
}
```

**Note:** Provide either `username` OR `email`, not both.

#### Response

**Status:** `200 OK`

```json
{
  "user": {
    "id": "user_xxxxxxxx",
    "username": "johndoe",
    "email": "john@example.com",
    "full_name": "John Doe",
    "status": "active",
    "is_active": true,
    "is_admin": false,
    "verified": true,
    "last_login": "2025-10-13T10:30:00Z",
    "created_at": "2025-10-13T10:00:00Z",
    "updated_at": "2025-10-13T10:30:00Z"
  },
  "token": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc...",
    "expires_at": "2025-10-13T10:45:00Z",
    "token_type": "Bearer"
  },
  "tenants": [
    {
      "id": "tenant_xxxxxxxx",
      "name": "John Doe's Workspace",
      "slug": "johndoe",
      "type": "personal",
      "owner_id": "user_xxxxxxxx"
    }
  ]
}
```

#### Error Responses

**401 Unauthorized** - Invalid credentials:
```json
{
  "error": "incorrect password",
  "field": "password"
}
```

**401 Unauthorized** - User not found:
```json
{
  "error": "user not found",
  "field": "username"
}
```

**401 Unauthorized** - Account suspended:
```json
{
  "error": "account is suspended",
  "field": "username"
}
```

#### Example

```bash
# Login with username
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "SecureP@ss123"
  }'

# Login with email
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecureP@ss123",
    "remember_me": true
  }'
```

---

### POST /api/auth/refresh

Refresh access token using refresh token.

#### Request

**Headers:**
```http
Content-Type: application/json
```

**Body:**
```json
{
  "refresh_token": "string (required)"
}
```

#### Response

**Status:** `200 OK`

```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": "2025-10-13T11:00:00Z",
  "token_type": "Bearer"
}
```

#### Error Responses

**400 Bad Request** - Missing refresh token:
```json
{
  "error": "refresh_token is required"
}
```

**401 Unauthorized** - Invalid or expired:
```json
{
  "error": "Invalid or expired refresh token"
}
```

#### Example

```bash
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGc..."
  }'
```

---

### POST /api/auth/logout

Logout user and invalidate token.

**Requires Authentication:** Yes (Bearer token)

#### Request

**Headers:**
```http
Authorization: Bearer <access_token>
```

#### Response

**Status:** `200 OK`

```json
{
  "message": "Logged out successfully"
}
```

#### Error Responses

**401 Unauthorized** - Not authenticated:
```json
{
  "error": "Authentication required"
}
```

#### Example

```bash
curl -X POST http://localhost:8080/api/auth/logout \
  -H "Authorization: Bearer eyJhbGc..."
```

---

### GET /api/auth/me

Get current authenticated user information.

**Requires Authentication:** Yes (Bearer token)

#### Request

**Headers:**
```http
Authorization: Bearer <access_token>
```

#### Response

**Status:** `200 OK`

```json
{
  "user_id": "user_xxxxxxxx",
  "username": "johndoe",
  "email": "john@example.com",
  "tenant_ids": ["tenant_xxxxxxxx", "tenant_yyyyyyyy"]
}
```

#### Example

```bash
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer eyJhbGc..."
```

---

## User Profile Endpoints

### GET /api/users/me

Get detailed user profile.

**Requires Authentication:** Yes

#### Response

```json
{
  "id": "user_xxxxxxxx",
  "username": "johndoe",
  "email": "john@example.com",
  "full_name": "John Doe",
  "status": "active",
  "is_active": true,
  "is_admin": false,
  "verified": true,
  "last_login": "2025-10-13T10:30:00Z",
  "created_at": "2025-10-13T10:00:00Z",
  "updated_at": "2025-10-13T10:30:00Z"
}
```

### PUT /api/users/me

Update user profile.

**Requires Authentication:** Yes

#### Request Body

```json
{
  "email": "newemail@example.com",
  "full_name": "John Smith"
}
```

### POST /api/users/me/password

Change user password.

**Requires Authentication:** Yes

#### Request Body

```json
{
  "old_password": "OldP@ss123",
  "new_password": "NewP@ss456"
}
```

---

## Error Responses

### Standard Error Format

All errors follow this format:

```json
{
  "error": "Error message",
  "field": "field_name (optional)"
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success |
| `201` | Created (registration) |
| `400` | Bad Request (validation error) |
| `401` | Unauthorized (invalid credentials or token) |
| `500` | Internal Server Error |

---

## Authentication Flow Examples

### Complete Registration + Login Flow

```bash
# 1. Register new user
RESPONSE=$(curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "SecureP@ss123"
  }')

# Extract access token
ACCESS_TOKEN=$(echo $RESPONSE | jq -r '.token.access_token')

# 2. Use access token for API calls
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Token Refresh Flow

```bash
# When access token expires, use refresh token
REFRESH_RESPONSE=$(curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

# Get new access token
NEW_ACCESS_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.access_token')
```

---

*Last updated: v1.6.3*

