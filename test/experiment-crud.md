# Experiment Service API Tests

This document provides curl commands to test the Experiment Service CRUD operations.

## Prerequisites

1. Start the service:
   ```bash
   cd /testbed/experiment-service
   go run main.go
   ```

2. The service runs on port 8080

## Authentication

All endpoints (except health check) require Bearer token authentication:
```
Authorization: Bearer secret-token
```

---

## 1. Health Check

### Check service health (no auth required)
```bash
curl -X GET http://localhost:8080/health
```

**Expected Response:**
```json
{"status": "healthy", "service": "experiment-service"}
```

---

## 2. Create Experiment

### Create a new experiment with control and treatment variants (ramp-up-percentage type)
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "button-color-test",
    "description": "Testing the impact of button color on conversion rate",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {
        "name": "control",
        "description": "Original blue button",
        "traffic_percentage": 50
      },
      {
        "name": "treatment",
        "description": "New green button",
        "traffic_percentage": 50
      }
    ]
  }'
```

**Expected Response (201 Created):**
```json
{
  "id": 1,
  "name": "button-color-test",
  "description": "Testing the impact of button color on conversion rate",
  "experiment_type": "ramp-up-percentage",
  "start_date": "2026-03-01T00:00:00Z",
  "end_date": "2026-03-31T23:59:59Z",
  "variants": [
    {
      "id": 1,
      "experiment_id": 1,
      "name": "control",
      "description": "Original blue button",
      "traffic_percentage": 50,
      "created_at": "...",
      "updated_at": "..."
    },
    {
      "id": 2,
      "experiment_id": 1,
      "name": "treatment",
      "description": "New green button",
      "traffic_percentage": 50,
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "rules": [],
  "created_at": "...",
  "updated_at": "..."
}
```

### Create a rule-based assignment experiment
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "premium-feature-rollout",
    "description": "Roll out premium features based on user attributes",
    "experiment_type": "rule-based-assignment",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {
        "name": "control",
        "description": "Default experience"
      },
      {
        "name": "treatment",
        "description": "Premium features enabled"
      }
    ],
    "rules": [
      {
        "priority": 1,
        "condition": "entity.attributes.country == '\''US'\'' AND entity.attributes.tier == '\''premium'\''",
        "action": "assign_variant: '\''treatment'\''"
      },
      {
        "priority": 2,
        "condition": "entity.attributes.country == '\''US'\''",
        "action": "assign_variant: '\''control'\''"
      },
      {
        "priority": 3,
        "condition": "true",
        "action": "assign_variant: '\''control'\''"
      }
    ]
  }'
```

**Expected Response (201 Created):**
```json
{
  "id": 2,
  "name": "premium-feature-rollout",
  "description": "Roll out premium features based on user attributes",
  "experiment_type": "rule-based-assignment",
  "start_date": "2026-03-01T00:00:00Z",
  "end_date": "2026-03-31T23:59:59Z",
  "variants": [
    {
      "id": 3,
      "experiment_id": 2,
      "name": "control",
      "description": "Default experience",
      "traffic_percentage": 0,
      "created_at": "...",
      "updated_at": "..."
    },
    {
      "id": 4,
      "experiment_id": 2,
      "name": "treatment",
      "description": "Premium features enabled",
      "traffic_percentage": 0,
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "rules": [
    {
      "id": 1,
      "experiment_id": 2,
      "priority": 1,
      "condition": "entity.attributes.country == 'US' AND entity.attributes.tier == 'premium'",
      "action": "assign_variant: 'treatment'",
      "created_at": "...",
      "updated_at": "..."
    },
    {
      "id": 2,
      "experiment_id": 2,
      "priority": 2,
      "condition": "entity.attributes.country == 'US'",
      "action": "assign_variant: 'control'",
      "created_at": "...",
      "updated_at": "..."
    },
    {
      "id": 3,
      "experiment_id": 2,
      "priority": 3,
      "condition": "true",
      "action": "assign_variant: 'control'",
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "created_at": "...",
  "updated_at": "..."
}
```

### Create another experiment with multiple variants
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "homepage-layout-test",
    "description": "Testing different homepage layouts",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-15T00:00:00Z",
    "end_date": "2026-04-15T23:59:59Z",
    "variants": [
      {
        "name": "control",
        "description": "Current layout",
        "traffic_percentage": 34
      },
      {
        "name": "treatment-a",
        "description": "New layout with hero section",
        "traffic_percentage": 33
      },
      {
        "name": "treatment-b",
        "description": "New layout with grid layout",
        "traffic_percentage": 33
      }
    ]
  }'
```

### Ramp up traffic split for an experiment (update variants with experiment PUT)
```bash
curl -X PUT http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "variants": [
      {
        "id": 2,
        "traffic_percentage": 70
      },
      {
        "id": 1,
        "traffic_percentage": 30
      }
    ]
  }'
```

**Expected Response (200 OK):**
```json
{
  "message": "Experiment updated successfully"
}
```

---

## 3. List Experiments

### Get all experiments
```bash
curl -X GET http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token"
```

**Expected Response (200 OK):**
```json
[
  {
    "id": 1,
    "name": "button-color-test",
    "description": "Testing the impact of button color on conversion rate",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [...],
    "rules": [],
    "created_at": "...",
    "updated_at": "..."
  },
  {
    "id": 2,
    "name": "homepage-layout-test",
    "description": "Testing different homepage layouts",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-15T00:00:00Z",
    "end_date": "2026-04-15T23:59:59Z",
    "variants": [...],
    "rules": [],
    "created_at": "...",
    "updated_at": "..."
  }
]
```

---

## 4. Get Experiment by ID

### Get a specific experiment
```bash
curl -X GET http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token"
```

**Expected Response (200 OK):**
```json
{
  "id": 1,
  "name": "button-color-test",
  "description": "Testing the impact of button color on conversion rate",
  "experiment_type": "ramp-up-percentage",
  "start_date": "2026-03-01T00:00:00Z",
  "end_date": "2026-03-31T23:59:59Z",
  "variants": [
    {
      "id": 1,
      "experiment_id": 1,
      "name": "control",
      "description": "Original blue button",
      "traffic_percentage": 50,
      "created_at": "...",
      "updated_at": "..."
    },
    {
      "id": 2,
      "experiment_id": 1,
      "name": "treatment",
      "description": "New green button",
      "traffic_percentage": 50,
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "rules": [],
  "created_at": "...",
  "updated_at": "..."
}
```

### Get non-existent experiment
```bash
curl -X GET http://localhost:8080/experiments/999 \
  -H "Authorization: Bearer secret-token"
```

**Expected Response (404 Not Found):**
```json
{
  "Code": 404,
  "Message": "Experiment not found"
}
```

---

## 5. Update Experiment

### Update experiment fields
```bash
curl -X PUT http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated: Testing button color impact on CTR",
    "end_date": "2026-04-30T23:59:59Z"
  }'
```

**Expected Response (200 OK):**
```json
{
  "message": "Experiment updated successfully"
}
```

### Update with no fields (should return error)
```bash
curl -X PUT http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "No fields to update"
}
```

---

## 6. Delete Experiment

### Delete an experiment
```bash
curl -X DELETE http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token"
```

**Expected Response (200 OK):**
```json
{
  "message": "Experiment deleted successfully"
}
```

### Delete non-existent experiment
```bash
curl -X DELETE http://localhost:8080/experiments/999 \
  -H "Authorization: Bearer secret-token"
```

**Expected Response (404 Not Found):**
```json
{
  "Code": 404,
  "Message": "Experiment not found"
}
```

---

## 7. Error Cases

### Missing authentication
```bash
curl -X GET http://localhost:8080/experiments
```

**Expected Response (401 Unauthorized):**
```json
{
  "Code": 401,
  "Message": "Unauthorized"
}
```

### Invalid experiment ID
```bash
curl -X GET http://localhost:8080/experiments/abc \
  -H "Authorization: Bearer secret-token"
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "Invalid experiment ID"
}
```

### Create experiment with duplicate name
```bash
# First, create an experiment
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "duplicate-test",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {"name": "control", "traffic_percentage": 50},
      {"name": "treatment", "traffic_percentage": 50}
    ]
  }'

# Then, try to create another with the same name
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "duplicate-test",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-04-01T00:00:00Z",
    "end_date": "2026-04-30T23:59:59Z",
    "variants": [
      {"name": "control", "traffic_percentage": 50},
      {"name": "treatment", "traffic_percentage": 50}
    ]
  }'
```

**Expected Response (409 Conflict):**
```json
{
  "Code": 400,
  "Message": "Experiment with this name already exists"
}
```

### Create experiment with missing required fields
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-experiment"
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "Start date and end date are required"
}
```

### Create experiment with less than 2 variants
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-experiment",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {
        "name": "control",
        "traffic_percentage": 100
      }
    ]
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "At least 2 variants are required (e.g., control and treatment)"
}
```

### Create experiment with invalid traffic percentage sum
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-experiment",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {
        "name": "control",
        "traffic_percentage": 50
      },
      {
        "name": "treatment",
        "traffic_percentage": 30
      }
    ]
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "Traffic percentages must sum to 100"
}
```

### Create experiment with end date before start date
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-experiment",
    "experiment_type": "ramp-up-percentage",
    "start_date": "2026-03-31T00:00:00Z",
    "end_date": "2026-03-01T23:59:59Z",
    "variants": [
      {
        "name": "control",
        "traffic_percentage": 50
      },
      {
        "name": "treatment",
        "traffic_percentage": 50
      }
    ]
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "End date must be after start date"
}
```

### Create rule-based experiment without rules (should return error)
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-rule-experiment",
    "experiment_type": "rule-based-assignment",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {"name": "control"},
      {"name": "treatment"}
    ]
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "At least one rule is required for rule-based experiments"
}
```

### Create rule-based experiment with duplicate priorities (should return error)
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "invalid-rule-experiment",
    "experiment_type": "rule-based-assignment",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {"name": "control"},
      {"name": "treatment"}
    ],
    "rules": [
      {
        "priority": 1,
        "condition": "entity.attributes.country == '\''US'\''",
        "action": "assign_variant: '\''treatment'\''"
      },
      {
        "priority": 1,
        "condition": "true",
        "action": "assign_variant: '\''control'\''"
      }
    ]
  }'
```

**Expected Response (400 Bad Request):**
```json
{
  "Code": 400,
  "Message": "Rule priorities must be unique"
}
```

---

## 8. Evaluate Experiment

### Evaluate a ramp-up-percentage experiment for a new entity
```bash
curl -X POST http://localhost:8080/experiments/1/evaluate \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "user",
    "entity_id": "user-123"
  }'
```

**Expected Response (200 OK):**
```json
{
  "experiment_id": 1,
  "variant_name": "control",
  "entity_type": "user",
  "entity_id": "user-123"
}
```

### Re-evaluate the same entity (should return the same variant)
```bash
curl -X POST http://localhost:8080/experiments/1/evaluate \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "user",
    "entity_id": "user-123"
  }'
```

---

## 8.1 Evaluate Rule-Based Experiment

### Evaluate a rule-based experiment with entity attributes
```bash
curl -X POST http://localhost:8080/experiments/2/evaluate \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "user",
    "entity_id": "user-456",
    "attributes": {
      "country": "US",
      "tier": "premium",
      "age": 30
    }
  }'
```

**Expected Response (200 OK) - Rule matched with assign_variant action:**
```json
{
  "experiment_id": 2,
  "variant_name": "treatment",
  "entity_type": "user",
  "entity_id": "user-456",
  "matched_rule": {
    "priority": 1,
    "action": "{\"action\": \"assign_variant\", \"variant\": \"treatment\"}"
  }
}
```

### Evaluate with attributes that don't match any rule (fallback to default)
```bash
curl -X POST http://localhost:8080/experiments/2/evaluate \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "user",
    "entity_id": "user-789",
    "attributes": {
      "country": "FR",
      "tier": "basic"
    }
  }'
```

**Expected Response (200 OK) - No rule matched, fallback to control:**
```json
{
  "experiment_id": 2,
  "variant_name": "control",
  "entity_type": "user",
  "entity_id": "user-789"
}
```

### Create a rule-based experiment with enable_experiment action
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "feature-flag-with-fallback",
    "description": "Feature flag with hash-based fallback",
    "experiment_type": "rule-based-assignment",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {"name": "control", "traffic_percentage": 50},
      {"name": "treatment", "traffic_percentage": 50}
    ],
    "rules": [
      {
        "priority": 1,
        "condition": "country == '\''US'\'' AND tier == '\''premium'\''",
        "action": "{\"action\": \"enable_experiment\"}"
      },
      {
        "priority": 2,
        "condition": "true",
        "action": "{\"action\": \"assign_variant\", \"variant\": \"control\"}"
      }
    ]
  }'
```

### Create a rule-based experiment with set_payload action
```bash
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ui-config-experiment",
    "description": "UI configuration experiment with payload",
    "experiment_type": "rule-based-assignment",
    "start_date": "2026-03-01T00:00:00Z",
    "end_date": "2026-03-31T23:59:59Z",
    "variants": [
      {"name": "control"},
      {"name": "treatment"}
    ],
    "rules": [
      {
        "priority": 1,
        "condition": "country == '\''US'\''",
        "action": "{\"action\": \"set_payload\", \"payload\": {\"buttonColor\": \"blue\", \"timeout\": 5000, \"showBanner\": true}}"
      },
      {
        "priority": 2,
        "condition": "country == '\''EU'\''",
        "action": "{\"action\": \"set_payload\", \"payload\": {\"buttonColor\": \"green\", \"timeout\": 3000, \"showBanner\": false}}"
      },
      {
        "priority": 3,
        "condition": "true",
        "action": "{\"action\": \"assign_variant\", \"variant\": \"control\"}"
      }
    ]
  }'
```

### Evaluate experiment with set_payload action
```bash
curl -X POST http://localhost:8080/experiments/3/evaluate \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "user",
    "entity_id": "user-us-123",
    "attributes": {
      "country": "US"
    }
  }'
```

**Expected Response (200 OK) - Returns payload:**
```json
{
  "experiment_id": 3,
  "variant_name": "control",
  "entity_type": "user",
  "entity_id": "user-us-123",
  "payload": {
    "buttonColor": "blue",
    "timeout": 5000,
    "showBanner": true
  },
  "matched_rule": {
    "priority": 1,
    "action": "{\"action\": \"set_payload\", \"payload\": {\"buttonColor\": \"blue\", \"timeout\": 5000, \"showBanner\": true}}"
  }
}
```

---

## 9. Manual Evaluation Overrides

### Create a manual evaluation override (whitelist)
```bash
curl -X POST http://localhost:8080/evaluations \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "experiment_id": 1,
    "entity_type": "user",
    "entity_id": "user-456",
    "variant_name": "treatment"
  }'
```

**Expected Response (201 Created):**
```json
{
  "message": "Manual evaluation created/updated successfully"
}
```

### Evaluate the whitelisted entity
```bash
curl -X POST http://localhost:8080/experiments/1/evaluate \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_type": "user",
    "entity_id": "user-456"
  }'
```

**Expected Response (200 OK - returns whitelisted variant):**
```json
{
  "experiment_id": 1,
  "variant_name": "treatment",
  "entity_type": "user",
  "entity_id": "user-456"
}
```

---

## Test Sequence

Run these commands in order for a complete test flow:

```bash
# 1. Check health
curl http://localhost:8080/health

# 2. Create experiment
curl -X POST http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-exp","description":"Test","experiment_type":"ramp-up-percentage","start_date":"2026-03-01T00:00:00Z","end_date":"2026-03-31T23:59:59Z","variants":[{"name":"control","traffic_percentage":50},{"name":"treatment","traffic_percentage":50}]}'

# 3. List experiments
curl -X GET http://localhost:8080/experiments \
  -H "Authorization: Bearer secret-token"

# 4. Get experiment by ID
curl -X GET http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token"

# 5. Update experiment
curl -X PUT http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -d '{"description":"Updated description"}'

# 6. Delete experiment
curl -X DELETE http://localhost:8080/experiments/1 \
  -H "Authorization: Bearer secret-token"
```
