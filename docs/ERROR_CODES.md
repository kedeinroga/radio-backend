# Error Codes Reference

This document lists all standardized error codes returned by the API. The frontend should use these codes to display localized error messages.

## Error Response Format

All errors follow this structure:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "English fallback message",
    "field": "fieldName"  // Optional, for validation errors
  }
}
```

## Authentication Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `USER_NOT_FOUND` | 404 | User does not exist |
| `USER_ALREADY_EXISTS` | 409 | Email already registered |
| `INVALID_CREDENTIALS` | 401 | Invalid email or password |
| `INVALID_TOKEN` | 401 | JWT token is invalid or expired |
| `UNAUTHORIZED` | 401 | Authentication required |
| `INVALID_USER_TYPE` | 400 | User type is invalid |

## Validation Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Generic validation error |
| `EMAIL_REQUIRED` | 400 | Email field is required |
| `EMAIL_INVALID` | 400 | Email format is invalid |
| `PASSWORD_REQUIRED` | 400 | Password field is required |
| `PASSWORD_TOO_SHORT` | 400 | Password must be at least 8 characters |
| `PASSWORD_WEAK` | 400 | Password must contain uppercase, lowercase, and digit |
| `QUERY_REQUIRED` | 400 | Search query is required |
| `QUERY_TOO_SHORT` | 400 | Search query must be at least 2 characters |

## Station Errors

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `STATION_NOT_FOUND` | 404 | Station does not exist |
| `INVALID_QUERY` | 400 | Search query is invalid |

## Frontend Implementation Example

### React/TypeScript

```typescript
const errorMessages = {
  en: {
    USER_NOT_FOUND: "User not found",
    INVALID_CREDENTIALS: "Invalid email or password",
    EMAIL_REQUIRED: "Email is required",
    // ... more translations
  },
  es: {
    USER_NOT_FOUND: "Usuario no encontrado",
    INVALID_CREDENTIALS: "Email o contraseña inválidos",
    EMAIL_REQUIRED: "El email es requerido",
    // ... more translations
  }
};

function getErrorMessage(errorCode: string, language: string): string {
  return errorMessages[language]?.[errorCode] || errorCode;
}

// Usage
try {
  await api.login(email, password);
} catch (error) {
  const message = getErrorMessage(error.code, currentLanguage);
  showError(message);
}
```

### Flutter/Dart

```dart
class ErrorMessages {
  static final Map<String, Map<String, String>> messages = {
    'en': {
      'USER_NOT_FOUND': 'User not found',
      'INVALID_CREDENTIALS': 'Invalid email or password',
      'EMAIL_REQUIRED': 'Email is required',
    },
    'es': {
      'USER_NOT_FOUND': 'Usuario no encontrado',
      'INVALID_CREDENTIALS': 'Email o contraseña inválidos',
      'EMAIL_REQUIRED': 'El email es requerido',
    },
  };

  static String get(String code, String language) {
    return messages[language]?[code] ?? code;
  }
}

// Usage
try {
  await api.login(email, password);
} catch (error) {
  final message = ErrorMessages.get(error.code, currentLanguage);
  showError(message);
}
```

## Adding New Error Codes

When adding new error codes:

1. Define the error in `internal/domain/errors.go`:
```go
var ErrNewError = &DomainError{
    Code:    "NEW_ERROR_CODE",
    Message: "English fallback message",
}
```

2. Update this documentation with the new code

3. Notify frontend team to add translations

## Best Practices

- **Always use error codes** - Never hardcode error messages in the backend
- **Provide context** - Use the `field` property for validation errors
- **Fallback messages** - The `message` field provides English fallback
- **Consistency** - Use UPPER_SNAKE_CASE for all error codes
- **Documentation** - Keep this document updated with all error codes
