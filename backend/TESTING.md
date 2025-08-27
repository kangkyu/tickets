# Testing Guide

This document explains how to run and write tests for the tickets-by-uma backend.

## Test Structure

```
backend/
├── integration_test.go           # Full HTTP integration tests
├── Makefile                      # Test automation commands
├── apphandlers/
│   └── event_handlers_test.go   # Handler unit tests
├── services/
│   └── uma_service_test.go      # Service layer tests
├── repositories/
│   └── repository_test.go       # Database layer tests
└── .github/workflows/test.yml   # CI/CD pipeline
```

## Prerequisites

1. **PostgreSQL**: Test database required
   ```bash
   # Install PostgreSQL (macOS)
   brew install postgresql
   brew services start postgresql
   
   # Create test database
   createdb tickets_uma_test
   ```

2. **Go Dependencies**: Ensure all modules are downloaded
   ```bash
   go mod download
   ```

## Running Tests

### Quick Commands (Using Makefile)

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests  
make test-integration

# Generate coverage report
make test-coverage

# Set up test database
make test-setup

# Clean up test artifacts
make clean
```

### Manual Commands

```bash
# Unit tests (no database required)
go test -v -short ./apphandlers/ ./services/

# Repository tests (requires test database)
go test -v ./repositories/

# Integration tests (requires test database)
go test -v -timeout 30s .

# All tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Test Categories

### 1. Unit Tests
- **Location**: `apphandlers/`, `services/`
- **Purpose**: Test individual functions and methods
- **Dependencies**: None (use mocks)
- **Run with**: `go test -short`

```go
// Example: Testing validation logic
func TestValidateCreateEventRequest(t *testing.T) {
    handler := &EventHandlers{}
    
    tests := []struct {
        name    string
        request models.CreateEventRequest
        wantErr bool
    }{
        // Test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := handler.validateCreateEventRequest(&tt.request)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 2. Repository Tests
- **Location**: `repositories/repository_test.go`
- **Purpose**: Test database operations
- **Dependencies**: Test PostgreSQL database
- **Features**: CRUD operations, concurrent access, edge cases

```go
func TestUserRepository(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    repo := NewUserRepository(db)
    
    // Test Create, Read, Update, Delete operations
    user := &models.User{
        Email: "test@example.com",
        Name:  "Test User",
    }
    
    err := repo.Create(user)
    // Assertions...
}
```

### 3. Service Layer Tests
- **Location**: `services/uma_service_test.go`
- **Purpose**: Test business logic and UMA service integration
- **Dependencies**: Mock external services
- **Coverage**: UMA validation, invoice creation, payment processing

```go
func TestUMAServiceValidation(t *testing.T) {
    service := NewLightsparkUMAService("", "", "", "", logger)
    
    tests := []struct {
        name    string
        address string
        wantErr bool
    }{
        {"valid UMA address", "$user@example.com", false},
        {"invalid format", "user@example.com", true},
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.ValidateUMAAddress(tt.address)
            // Assertions...
        })
    }
}
```

### 4. Integration Tests
- **Location**: `integration_test.go`
- **Purpose**: Test full HTTP request/response cycles
- **Dependencies**: Test database + mock services
- **Coverage**: All API endpoints, error handling, authentication

```go
func TestTicketPurchaseFlow(t *testing.T) {
    ts := setupTestServer(t)
    defer ts.teardown()
    
    // Create event
    event := models.CreateEventRequest{
        Title: "Test Event",
        // ... other fields
    }
    
    eventJSON, _ := json.Marshal(event)
    resp, err := http.Post(ts.httpServer.URL+"/api/admin/events", 
        "application/json", bytes.NewBuffer(eventJSON))
    
    // Assertions on response
    if resp.StatusCode != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", resp.StatusCode)
    }
}
```

## Test Environment Setup

### Database Configuration
- **Test Database**: `tickets_uma_test`
- **Connection**: `postgres://postgres:password@localhost:5432/tickets_uma_test?sslmode=disable`
- **Schema**: Automatically applied from `db/schema.sql`
- **Cleanup**: Tables truncated between tests

### Mock Services
```go
type MockUMAService struct {
    logger *slog.Logger
}

func (m *MockUMAService) CreateUMARequest(umaAddress string, amountSats int64, description string, isAdmin bool) (*models.Invoice, error) {
    return &models.Invoice{
        ID:          "test-invoice-123",
        PaymentHash: "test-payment-hash-456",
        Bolt11:      "lntb10000n1p3test...",
        AmountSats:  amountSats,
        Status:      "pending",
        ExpiresAt:   timePtr(time.Now().Add(time.Hour)),
    }, nil
}
```

## Writing New Tests

### 1. Unit Test Template
```go
func TestNewFeature(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"
    
    // Act  
    result := functionToTest(input)
    
    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### 2. Table-Driven Test Template
```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid case", "input1", "output1", false},
        {"error case", "bad input", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := functionToTest(tt.input)
            
            if tt.wantErr && err == nil {
                t.Error("Expected error but got none")
            }
            
            if !tt.wantErr && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            
            if result != tt.expected {
                t.Errorf("Expected %s, got %s", tt.expected, result)
            }
        })
    }
}
```

### 3. Integration Test Template
```go
func TestNewEndpoint(t *testing.T) {
    ts := setupTestServer(t)
    defer ts.teardown()
    
    // Prepare request
    payload := map[string]interface{}{
        "field": "value",
    }
    jsonPayload, _ := json.Marshal(payload)
    
    // Make request
    resp, err := http.Post(ts.httpServer.URL+"/api/endpoint", 
        "application/json", bytes.NewBuffer(jsonPayload))
    if err != nil {
        t.Fatal("Request failed:", err)
    }
    defer resp.Body.Close()
    
    // Assert response
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
    
    var response models.SuccessResponse
    json.NewDecoder(resp.Body).Decode(&response)
    
    // Assert response data
    if response.Message == "" {
        t.Error("Expected message in response")
    }
}
```

## Continuous Integration

GitHub Actions automatically run tests on:
- Push to `main` or `develop` branches
- Pull requests to `main`
- Changes in `backend/` directory

### Pipeline Steps
1. Set up PostgreSQL service
2. Install Go dependencies
3. Apply database schema
4. Run unit tests
5. Run integration tests
6. Generate coverage report
7. Upload to Codecov

## Test Data Management

### Database Cleanup
```go
func cleanDatabase(t *testing.T, db *sqlx.DB) {
    tables := []string{"payments", "tickets", "events", "users", "uma_request_invoices"}
    for _, table := range tables {
        _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
        if err != nil {
            t.Logf("Warning: Could not truncate table %s: %v", table, err)
        }
    }
}
```

### Test Fixtures
```go
func createTestUser(t *testing.T, repo repositories.UserRepository) *models.User {
    user := &models.User{
        Email: "test@example.com",
        Name:  "Test User",
    }
    
    err := repo.Create(user)
    if err != nil {
        t.Fatal("Failed to create test user:", err)
    }
    
    return user
}
```

## Coverage Goals

- **Unit Tests**: >90% line coverage
- **Integration Tests**: All API endpoints covered
- **Repository Tests**: All CRUD operations covered
- **Service Tests**: All business logic covered

## Best Practices

### 1. Test Naming
- Use descriptive test names: `TestCreateUser_WithValidEmail_ReturnsSuccess`
- Group related tests: `TestUserRepository`, `TestUserService`

### 2. Test Structure
- **Arrange**: Set up test data
- **Act**: Execute the function being tested  
- **Assert**: Verify the results

### 3. Test Independence
- Each test should be independent
- Use database cleanup between tests
- Don't rely on test execution order

### 4. Error Testing
- Test both success and failure cases
- Verify error messages and types
- Test edge cases and boundary conditions

### 5. Performance Testing
- Include benchmark tests for critical paths
- Test concurrent operations
- Monitor test execution time

## Debugging Tests

### Verbose Output
```bash
go test -v ./...
```

### Run Specific Test
```bash
go test -run TestUserRepository ./repositories/
```

### Test with Race Detection
```bash
go test -race ./...
```

### Debug with Delve
```bash
dlv test -- -test.run TestSpecificTest
```

## Troubleshooting

### Common Issues

1. **Database Connection Refused**
   - Ensure PostgreSQL is running: `brew services start postgresql`
   - Check database exists: `psql -l | grep tickets_uma_test`

2. **Schema Not Applied**
   - Run: `make test-setup`
   - Or manually: `psql -d tickets_uma_test -f db/schema.sql`

3. **Import Cycles**
   - Check package imports
   - Move shared test utilities to separate package

4. **Mock Service Issues**  
   - Verify mock implements correct interface
   - Check method signatures match

### Getting Help

- Check test logs for detailed error messages
- Use `go test -v` for verbose output
- Review database logs if connection issues persist
- Ensure all dependencies are installed with `go mod download`