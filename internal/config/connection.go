// Package config provides configuration management for CDD CLI.
package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

// Connection represents a named API connection to a provider.
// Users can have multiple connections to the same provider type
// (e.g., "Work Claude" and "Personal Claude" both using Anthropic).
type Connection struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	ProviderID   string            `json:"provider_id"`
	APIKey       string            `json:"api_key,omitempty"`
	OAuthToken   *oauth.Token      `json:"oauth,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// IsConfigured returns true if the connection has authentication configured.
func (c *Connection) IsConfigured() bool {
	return c.APIKey != "" || c.OAuthToken != nil
}

// IsOAuth returns true if the connection uses OAuth authentication.
func (c *Connection) IsOAuth() bool {
	return c.OAuthToken != nil
}

// ConnectionManager provides CRUD operations for connections.
type ConnectionManager struct {
	cfg *Config
}

// NewConnectionManager creates a new ConnectionManager.
func NewConnectionManager(cfg *Config) *ConnectionManager {
	return &ConnectionManager{cfg: cfg}
}

// List returns all connections.
func (m *ConnectionManager) List() []Connection {
	if m.cfg.Connections == nil {
		return []Connection{}
	}
	return m.cfg.Connections
}

// Get returns a connection by ID.
func (m *ConnectionManager) Get(id string) *Connection {
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].ID == id {
			return &m.cfg.Connections[i]
		}
	}
	return nil
}

// GetByName returns a connection by name.
func (m *ConnectionManager) GetByName(name string) *Connection {
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].Name == name {
			return &m.cfg.Connections[i]
		}
	}
	return nil
}

// GetByProvider returns all connections for a given provider ID.
func (m *ConnectionManager) GetByProvider(providerID string) []Connection {
	var result []Connection
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].ProviderID == providerID {
			result = append(result, m.cfg.Connections[i])
		}
	}
	return result
}

// Add creates a new connection and saves to config.
func (m *ConnectionManager) Add(conn Connection) error {
	// Generate ID if not provided.
	if conn.ID == "" {
		conn.ID = uuid.New().String()
	}

	// Validate required fields.
	if conn.Name == "" {
		return fmt.Errorf("connection name is required")
	}
	if conn.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}

	// Check for duplicate name.
	if existing := m.GetByName(conn.Name); existing != nil {
		return fmt.Errorf("connection with name %q already exists", conn.Name)
	}

	// Set timestamps.
	now := time.Now()
	conn.CreatedAt = now
	conn.UpdatedAt = now

	// Add to config.
	m.cfg.Connections = append(m.cfg.Connections, conn)

	// Persist to disk.
	return Save(m.cfg)
}

// Update modifies an existing connection.
func (m *ConnectionManager) Update(conn Connection) error {
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].ID != conn.ID {
			continue
		}
		// Preserve original CreatedAt.
		conn.CreatedAt = m.cfg.Connections[i].CreatedAt
		conn.UpdatedAt = time.Now()

		// Check for duplicate name (excluding self).
		if existing := m.GetByName(conn.Name); existing != nil && existing.ID != conn.ID {
			return fmt.Errorf("connection with name %q already exists", conn.Name)
		}

		m.cfg.Connections[i] = conn
		return Save(m.cfg)
	}
	return fmt.Errorf("connection %q not found", conn.ID)
}

// Delete removes a connection by ID.
func (m *ConnectionManager) Delete(id string) error {
	for i := range m.cfg.Connections {
		if m.cfg.Connections[i].ID == id {
			// Remove from slice.
			m.cfg.Connections = append(m.cfg.Connections[:i], m.cfg.Connections[i+1:]...)
			return Save(m.cfg)
		}
	}
	return fmt.Errorf("connection %q not found", id)
}

// GetActiveConnection returns the connection for the given model tier.
// It first tries to use ConnectionID, then falls back to Provider for backward compatibility.
func (m *ConnectionManager) GetActiveConnection(tier SelectedModelType) *Connection {
	model, ok := m.cfg.Models[tier]
	if !ok {
		return nil
	}

	// Try ConnectionID first (new behavior).
	if model.ConnectionID != "" {
		return m.Get(model.ConnectionID)
	}

	// Fall back to Provider (backward compatibility).
	if model.Provider != "" {
		connections := m.GetByProvider(model.Provider)
		if len(connections) > 0 {
			return &connections[0]
		}
	}

	return nil
}

// SetActiveModel sets the active model for a tier.
func (m *ConnectionManager) SetActiveModel(tier SelectedModelType, connectionID, modelID string) error {
	// Verify connection exists.
	conn := m.Get(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	// Update the model selection.
	model := m.cfg.Models[tier]
	model.ConnectionID = connectionID
	model.Model = modelID
	model.Provider = conn.ProviderID // Keep for backward compatibility
	m.cfg.Models[tier] = model

	return Save(m.cfg)
}
