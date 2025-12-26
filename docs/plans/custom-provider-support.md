# Custom Provider Support Implementation Plan

## Overview

This document outlines the implementation plan for adding comprehensive custom provider support to CDD (Context-Driven Development). The goal is to allow users to define and manage their own AI model providers alongside the built-in catwalk providers.

## Problem Statement

Currently, CDD only supports providers available through the catwalk service (`https://catwalk.charm.sh`). This limits users to:
- Public providers registered with catwalk
- No way to add internal/enterprise LLMs
- No support for self-hosted model servers
- Limited customization of provider settings

## Solution Goals

1. **Multi-Source Provider Support**: Combine catwalk providers with user-defined custom providers
2. **Flexible Provider Definition**: Support multiple formats (URL, file, manual)
3. **Enterprise Ready**: Enable internal LLM integration for organizations
4. **User-Friendly UX**: Seamless integration with existing wizard workflow
5. **Backward Compatibility**: Maintain all existing functionality

## Architecture Overview

### File Structure Changes
```
~/.config/cdd/
├── cdd.json              # Main config (user API keys, model selections)
├── providers.json        # Catwalk providers cache (auto-managed)
└── custom-providers.json # User-defined providers (user-managed)
```

### Component Responsibilities
- **Provider Manager**: Load, validate, and merge provider sources
- **Custom Provider Storage**: Persist user-defined providers separately
- **Wizard Integration**: Add custom provider steps to setup flow
- **CLI Commands**: Provide management interface for custom providers
- **Validation Layer**: Ensure provider definitions are valid and safe

## Implementation Phases

### Phase 1: Backend Infrastructure (Week 1-2)

#### 1.1 Custom Provider Data Structures
**File**: `internal/config/custom_provider.go`

```go
// CustomProvider represents a user-defined provider
type CustomProvider struct {
    Name                string            `json:"name"`
    ID                  string            `json:"id"`
    Type                catwalk.Type      `json:"type"`
    APIEndpoint         string            `json:"api_endpoint"`
    DefaultHeaders      map[string]string `json:"default_headers,omitempty"`
    DefaultLargeModelID string            `json:"default_large_model_id,omitempty"`
    DefaultSmallModelID string            `json:"default_small_model_id,omitempty"`
    Models              []catwalk.Model   `json:"models"`
    CreatedAt           time.Time         `json:"created_at"`
    UpdatedAt           time.Time         `json:"updated_at"`
}

// CustomProviderManager handles custom provider lifecycle
type CustomProviderManager struct {
    filePath string
}
```

**Key Features**:
- Separate storage from main config
- Timestamp tracking for audit
- Validation of provider definitions
- Safe merging with catwalk providers

#### 1.2 Provider Loading and Merging
**File**: `internal/config/provider_loader.go`

```go
// ProviderLoader combines multiple provider sources
type ProviderLoader struct {
    catwalkURL     string
    customManager  *CustomProviderManager
}

// LoadAllProviders returns merged provider list
func (pl *ProviderLoader) LoadAllProviders() ([]catwalk.Provider, error)
```

**Merge Strategy**:
1. Load catwalk providers (cached or fresh)
2. Load custom providers from storage
3. Merge by provider ID (custom providers override catwalk)
4. Validate all providers
5. Return combined list to wizard

#### 1.3 Provider Validation
**File**: `internal/config/provider_validator.go`

```go
// Validation checks for custom providers
type ValidationResult struct {
    IsValid bool
    Errors  []ValidationError
    Warnings []ValidationWarning
}

func ValidateCustomProvider(p *CustomProvider) *ValidationResult
```

**Validation Rules**:
- Unique provider ID (not conflicting with catwalk)
- Valid provider type (supported by fantasy library)
- Valid API endpoint URL format
- At least one model defined
- Model IDs are unique within provider
- Supported model capabilities

### Phase 2: Wizard Integration (Week 3-4)

#### 2.1 Enhanced Provider Selection
**File**: `internal/tui/components/wizard/custom_provider.go`

**New Wizard Steps**:
1. **Provider Selection**: Add "Add Custom Provider" option
2. **Custom Provider Method**: Choose import source (URL/File/Manual)
3. **Import Configuration**: For URL/File imports
4. **Manual Definition**: Step-by-step provider creation
5. **Model Configuration**: Define models for custom provider

**UI Flow**:
```
Select Provider
├── 🌟 Recommended Providers (from catwalk)
│   ├── Anthropic (OAuth/API Key)
│   ├── OpenAI (API Key)
│   └── ...
├── 🔧 Custom Providers (configured)
│   ├── My Internal LLM
│   └── Company GPT
└── ➕ Add Custom Provider
    ├── Import from URL
    ├── Import from File
    └── Manual Definition
```

#### 2.2 Custom Provider Import Wizard
**File**: `internal/tui/components/wizard/import_provider.go`

**Import Methods**:

1. **Import from URL**
   ```
   ┌─ Import from URL ──────────────────────────┐
   │ Enter provider catalog URL:                │
   │ https://company.com/providers.json  [____] │
   │                                             │
   │ [✓] Validate URL    [Import]   [← Back]    │
   └─────────────────────────────────────────────┘
   ```

2. **Import from File**
   ```
   ┌─ Import from File ─────────────────────────┐
   │ Select JSON file:                          │
   │ ./my-providers.json                 [Browse]│
   │                                             │
   │ [✓] Validate File    [Import]   [← Back]    │
   └─────────────────────────────────────────────┘
   ```

3. **Manual Definition**
   ```
   ┌─ Define Custom Provider ───────────────────┐
   │ Provider Name: Company LLM           [____]│
   │ Provider ID:   company-llm           [____]│
   │ Provider Type: OpenAI Compatible     [▼]   │
   │                                             │
   │ API Endpoint:                              │
   │ https://llm.company.com/v1          [____] │
   │                                             │
   │ Authentication:                             │
   │ Header Name: Authorization            [____]│
   │ Header Value: Bearer $COMPANY_KEY     [____]│
   │                                             │
   │ [+ Add Model]  [✓ Continue]  [← Back]      │
   └─────────────────────────────────────────────┘
   ```

#### 2.3 Model Definition for Custom Providers
**File**: `internal/tui/components/wizard/custom_model.go`

**Model Configuration UI**:
```
┌─ Define Models for Company LLM ──────────────┐
│                                               │
│ Model 1:                                      │
│ Name: Company GPT-4                     [____]│
│ ID:   company-gpt-4                   [____]│
│ Context Window: 128000                 [____]│
│ Cost per 1M input: 0.01               [____]│
│ Cost per 1M output: 0.03              [____]│
│                                               │
│ [+ Add Model]  [✓ Finish]  [← Back]         │
└───────────────────────────────────────────────┘
```

### Phase 3: CLI Commands (Week 5)

#### 3.1 Provider Management Commands
**File**: `cmd/providers.go`

**New Commands**:
```bash
# List all providers (catwalk + custom)
cdd providers list

# Add custom provider from URL
cdd providers add-url https://company.com/providers.json

# Add custom provider from file
cdd providers add-file ./my-providers.json

# Add custom provider manually (interactive)
cdd providers add

# Remove custom provider
cdd providers remove <provider-id>

# Update custom provider
cdd providers update <provider-id>

# Export custom providers
cdd providers export ./my-providers.json

# Validate custom providers
cdd providers validate
```

#### 3.2 Provider Detail Commands
```bash
# Show provider details
cdd providers show <provider-id>

# List models for provider
cdd providers models <provider-id>

# Test provider connection
cdd providers test <provider-id>
```

### Phase 4: Advanced Features (Week 6-7)

#### 4.1 Provider Templates
**File**: `internal/config/provider_templates.go`

**Pre-built Templates**:
- OpenAI Compatible API
- Anthropic Compatible API  
- Azure OpenAI Service
- Google Vertex AI
- AWS Bedrock
- Local Ollama Server

**Template Usage**:
```bash
cdd providers add-template ollama --base-url http://localhost:11434
```

#### 4.2 Provider Sharing
**File**: `internal/config/provider_sharing.go`

**Features**:
- Export provider definitions as shareable JSON
- Import provider definitions from others
- Provider marketplace integration (future)
- Organization-wide provider sharing

#### 4.3 Advanced Configuration
**File**: `internal/config/advanced_provider.go`

**Advanced Settings**:
- Custom rate limiting
- Retry policies
- Timeout configurations
- Fallback provider chains
- Load balancing between providers

### Phase 5: Testing and Documentation (Week 8)

#### 5.1 Comprehensive Testing
**Test Files**:
- `internal/config/custom_provider_test.go`
- `internal/config/provider_loader_test.go`
- `internal/tui/components/wizard/custom_provider_test.go`
- `cmd/providers_test.go`

**Test Scenarios**:
- Valid custom provider definitions
- Invalid provider handling
- Merge behavior with catwalk providers
- Wizard flow with custom providers
- CLI command functionality
- Error handling and edge cases

#### 5.2 Documentation
**Files to Update/Create**:
- `docs/custom-providers.md` - User guide
- `docs/providers-api.md` - API documentation
- `README.md` - Update with new features
- Provider template examples in `examples/providers/`

## Detailed Implementation Details

### Data Model Changes

#### Custom Provider Schema
```json
{
  "version": "1.0",
  "providers": [
    {
      "name": "Company Internal LLM",
      "id": "company-llm",
      "type": "openai_compat",
      "api_endpoint": "https://llm.company.com/v1",
      "default_headers": {
        "Authorization": "Bearer $COMPANY_API_KEY",
        "X-Company-ID": "my-company"
      },
      "default_large_model_id": "company-gpt-4",
      "default_small_model_id": "company-gpt-3.5",
      "models": [
        {
          "id": "company-gpt-4",
          "name": "Company GPT-4",
          "context_window": 128000,
          "cost_per_1m_in": 0.01,
          "cost_per_1m_out": 0.03,
          "options": {
            "temperature": 0.7,
            "top_p": 0.9
          }
        }
      ],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Modified Main Config
```json
{
  "models": {
    "large": {
      "model": "company-gpt-4",
      "provider": "company-llm"
    },
    "small": {
      "model": "company-gpt-3.5", 
      "provider": "company-llm"
    }
  },
  "providers": {
    "company-llm": {
      "api_key": "$COMPANY_API_KEY",
      "custom": true
    }
  }
}
```

### Error Handling Strategy

#### Validation Errors
- **Provider ID Conflict**: Custom provider ID matches catwalk provider
- **Invalid Model Definition**: Model missing required fields
- **Unsupported Provider Type**: Provider type not supported by fantasy
- **Invalid API Endpoint**: Malformed URL or unreachable endpoint

#### Runtime Errors
- **Provider Unavailable**: Custom provider API endpoint unreachable
- **Authentication Failure**: Invalid API key or credentials
- **Model Not Found**: Requested model not available in provider
- **Rate Limiting**: Provider API rate limits exceeded

### Security Considerations

#### Input Validation
- Sanitize all user-provided provider definitions
- Validate URL endpoints to prevent SSRF attacks
- Limit custom provider count to prevent resource exhaustion
- Validate JSON structure before parsing

#### API Key Security
- Never log or expose custom API keys
- Use environment variable substitution securely
- Provide guidance on secure credential storage
- Support secret management integration (future)

#### Network Security
- Validate HTTPS for production providers
- Support custom CA certificates for enterprise
- Implement connection timeouts and retries
- Log network errors without sensitive data

### Performance Optimization

#### Provider Caching
- Cache custom provider definitions locally
- Implement TTL-based cache invalidation
- Lazy load provider details on first use
- Background refresh of provider metadata

#### Wizard Performance
- Async provider loading in wizard
- Progress indicators for slow operations
- Cancel long-running operations
- Minimize wizard step count

## Migration Strategy

### Existing Users
1. **Automatic Migration**: No action required for existing users
2. **Gradual Adoption**: Custom provider features opt-in
3. **Backward Compatibility**: All existing functionality preserved
4. **Clear Migration Path**: Documentation for advanced features

### Configuration Migration
```bash
# Existing config continues to work
~/.config/cdd/cdd.json

# Custom providers stored separately  
~/.config/cdd/custom-providers.json

# No manual migration needed
```

## Success Metrics

### User Experience
- **Setup Time**: Custom provider setup < 5 minutes
- **Error Rate**: < 5% of custom provider configurations fail
- **User Satisfaction**: > 4.5/5 rating for custom provider features
- **Adoption Rate**: > 20% of users try custom providers within 30 days

### Technical Performance
- **Startup Time**: No impact on CDD startup performance
- **Memory Usage**: < 10MB additional memory for custom provider support
- **Wizard Performance**: < 2 second delay for provider loading
- **CLI Response**: All provider commands < 1 second response time

### Business Impact
- **Enterprise Adoption**: Enable 3+ enterprise customers to use internal LLMs
- **Feature Requests**: Address 80% of custom provider related feature requests
- **Competitive Advantage**: Unique provider flexibility vs competitors
- **Community Engagement**: Increase contributor interest in provider ecosystem

## Risk Assessment

### Technical Risks
- **Provider Compatibility**: Some custom providers may not work perfectly
- **Wizard Complexity**: Adding custom provider steps may confuse users
- **Performance Impact**: Additional provider loading may slow startup
- **Maintenance Burden**: Supporting diverse provider types increases complexity

### Mitigation Strategies
- **Comprehensive Testing**: Test with multiple provider types
- **Progressive Disclosure**: Hide advanced features unless needed
- **Performance Monitoring**: Track startup time and memory usage
- **Community Support**: Leverage community for provider testing and support

### User Experience Risks
- **Overwhelming UI**: Too many options may confuse users
- **Setup Complexity**: Custom provider setup may be too complex
- **Error Recovery**: Poor error messages may frustrate users
- **Feature Discovery**: Users may not know about custom provider features

### Mitigation Strategies
- **User Testing**: Test wizard flow with representative users
- **Simplified Defaults**: Provide sensible defaults for all options
- **Clear Error Messages**: Descriptive error messages with recovery suggestions
- **Feature Documentation**: Comprehensive guides and examples

## Future Enhancements

### Phase 6: Advanced Provider Features
- **Provider Chains**: Fallback between multiple providers
- **Load Balancing**: Distribute requests across providers
- **Cost Optimization**: Automatic provider selection based on cost
- **Quality Routing**: Route requests to best provider for task type

### Phase 7: Ecosystem Integration
- **Provider Marketplace**: Community-driven provider sharing
- **Enterprise SSO**: Integration with enterprise identity systems
- **Secret Management**: Integration with HashiCorp Vault, AWS Secrets Manager
- **Monitoring Integration**: Provider usage analytics and monitoring

### Phase 8: AI-Powered Features
- **Provider Recommendations**: AI-suggested providers based on use case
- **Automatic Optimization**: AI-driven provider selection and configuration
- **Performance Prediction**: Predict provider performance for specific tasks
- **Cost Estimation**: AI-powered cost prediction and optimization

## Conclusion

This comprehensive plan for custom provider support will significantly enhance CDD's flexibility and enterprise readiness. The phased approach ensures manageable implementation while maintaining backward compatibility and focusing on user experience.

The combination of wizard integration, CLI commands, and advanced features will make CDD the most flexible AI coding assistant available, capable of working with any OpenAI-compatible API or custom enterprise LLM deployment.

## Next Steps

1. **Review and Approve Plan**: Stakeholder review of this comprehensive plan
2. **Phase 1 Implementation**: Start with backend infrastructure
3. **User Testing**: Test wizard flow with sample users
4. **Iterative Development**: Regular checkpoints and adjustments
5. **Documentation**: Parallel documentation development
6. **Beta Release**: Limited beta with early adopters
7. **General Availability**: Full release with marketing support

---

**Document Version**: 1.0  
**Last Updated**: 2024-12-19  
**Author**: CDD Development Team  
**Reviewers**: TBD