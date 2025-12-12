# Contributing to Radio Backend

## Code Standards

### Clean Architecture Principles

This project strictly follows Clean Architecture (Hexagonal Architecture). Please ensure:

1. **Dependency Rule**: Dependencies only point inward
   - Domain layer has NO external dependencies
   - Services depend only on domain interfaces
   - Handlers depend on services
   - Infrastructure implements domain interfaces

2. **SOLID Principles**:
   - **Single Responsibility**: Each struct/function has one reason to change
   - **Open/Closed**: Extend via new implementations, not modifications
   - **Liskov Substitution**: All interface implementations are interchangeable
   - **Interface Segregation**: Small, focused interfaces
   - **Dependency Inversion**: Depend on abstractions, not concretions

### Code Style

- **Formatting**: Use `gofmt` and `goimports` (run `make fmt`)
- **Naming**:
  - Interfaces: `Repository`, `Service`, `Handler` (no "I" prefix)
  - Implementations: `PostgresRepository`, `StationService`
  - Private functions: `parseLimit`, `validateEmail`
  - Public functions: `NewStationService`, `ListPopular`

- **Function Size**: Max 60 lines, ideally 20-30 lines
- **Complexity**: Cyclomatic complexity < 15
- **Comments**: Only for "why", not "what" (code should be self-documenting)

### Error Handling

```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to fetch stations: %w", err)
}

// Use domain errors for business logic
if user == nil {
    return domain.ErrUserNotFound
}
```

### Testing

- Write unit tests for all services and domain logic
- Write integration tests for handlers and repositories
- Use table-driven tests for multiple scenarios
- Minimum 80% code coverage
- Run tests with: `make test`

Example:

```go
func TestStationService_ListPopular(t *testing.T) {
    tests := []struct {
        name     string
        limit    int
        country  string
        userType domain.UserType
        want     int
        wantErr  bool
    }{
        {
            name:     "guest user",
            limit:    10,
            country:  "US",
            userType: domain.UserTypeGuest,
            want:     10,
            wantErr:  false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Development Workflow

1. **Fork and Clone**
```bash
git clone <your-fork>
cd radio-backend
```

2. **Create Feature Branch**
```bash
git checkout -b feature/your-feature-name
```

3. **Make Changes**
   - Follow code standards above
   - Write tests
   - Update documentation if needed

4. **Run Quality Checks**
```bash
make fmt      # Format code
make lint     # Run linters
make test     # Run tests
make security # Security scan
```

5. **Commit Changes**
```bash
git add .
git commit -m "feat: add new feature"
```

Use conventional commits:
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

6. **Push and Create PR**
```bash
git push origin feature/your-feature-name
```

## Pull Request Guidelines

- Provide clear description of changes
- Reference related issues
- Ensure all CI checks pass
- Request review from maintainers
- Address review feedback promptly

## Questions?

Open an issue for questions or discussions.
