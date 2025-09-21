package preview

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// InMemoryURLManager implements URLManager interface with in-memory storage
type InMemoryURLManager struct {
	config      *PreviewConfig
	reservations map[string]*SubdomainReservation
	mutex       sync.RWMutex
}

// SubdomainReservation represents a subdomain reservation
type SubdomainReservation struct {
	Subdomain string    `json:"subdomain"`
	PreviewID string    `json:"preview_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewInMemoryURLManager creates a new in-memory URL manager
func NewInMemoryURLManager(config *PreviewConfig) *InMemoryURLManager {
	return &InMemoryURLManager{
		config:       config,
		reservations: make(map[string]*SubdomainReservation),
	}
}

// GenerateSubdomain generates a unique subdomain
func (um *InMemoryURLManager) GenerateSubdomain(appName string) (string, error) {
	um.mutex.Lock()
	defer um.mutex.Unlock()

	// Clean app name for subdomain use
	cleanAppName := um.cleanAppName(appName)

	// Generate hash for uniqueness
	hash, err := um.generateHash(6)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash: %w", err)
	}

	// Apply subdomain pattern
	subdomain := strings.ReplaceAll(um.config.SubdomainPattern, "{app}", cleanAppName)
	subdomain = strings.ReplaceAll(subdomain, "{hash}", hash)

	// Ensure uniqueness
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		if _, exists := um.reservations[subdomain]; !exists {
			return subdomain, nil
		}

		// Generate new hash and try again
		hash, err = um.generateHash(6)
		if err != nil {
			return "", fmt.Errorf("failed to generate hash: %w", err)
		}
		subdomain = strings.ReplaceAll(um.config.SubdomainPattern, "{app}", cleanAppName)
		subdomain = strings.ReplaceAll(subdomain, "{hash}", hash)
	}

	return "", fmt.Errorf("failed to generate unique subdomain after %d attempts", maxAttempts)
}

// ReserveSubdomain reserves a subdomain
func (um *InMemoryURLManager) ReserveSubdomain(ctx context.Context, subdomain string, previewID string, ttl time.Duration) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()

	// Validate subdomain
	if err := um.ValidateSubdomain(subdomain); err != nil {
		return fmt.Errorf("invalid subdomain: %w", err)
	}

	// Check if already reserved
	if existing, exists := um.reservations[subdomain]; exists {
		if existing.PreviewID != previewID {
			return fmt.Errorf("subdomain %s is already reserved", subdomain)
		}
		// Update existing reservation
		existing.ExpiresAt = time.Now().Add(ttl)
		return nil
	}

	// Create new reservation
	reservation := &SubdomainReservation{
		Subdomain: subdomain,
		PreviewID: previewID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	um.reservations[subdomain] = reservation
	return nil
}

// ReleaseSubdomain releases a subdomain
func (um *InMemoryURLManager) ReleaseSubdomain(ctx context.Context, subdomain string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()

	delete(um.reservations, subdomain)
	return nil
}

// GetURL constructs the full URL for a subdomain
func (um *InMemoryURLManager) GetURL(subdomain string, tls bool) string {
	scheme := "http"
	if tls {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s.%s", scheme, subdomain, um.config.BaseDomain)
}

// ValidateSubdomain validates subdomain format
func (um *InMemoryURLManager) ValidateSubdomain(subdomain string) error {
	// RFC 1035 compliant subdomain validation
	if len(subdomain) == 0 || len(subdomain) > 63 {
		return fmt.Errorf("subdomain length must be 1-63 characters")
	}

	// Must start and end with alphanumeric character
	if !regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`).MatchString(subdomain) {
		return fmt.Errorf("subdomain must contain only lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen")
	}

	// Check for reserved subdomains
	reserved := []string{
		"www", "api", "admin", "app", "mail", "ftp", "ssh", "vpn",
		"staging", "prod", "production", "dev", "development",
		"test", "testing", "demo", "docs", "help", "support",
		"blog", "news", "about", "contact", "login", "auth",
		"preview", "quantumlayer", "qlf",
	}

	for _, r := range reserved {
		if subdomain == r {
			return fmt.Errorf("subdomain %s is reserved", subdomain)
		}
	}

	return nil
}

// IsSubdomainAvailable checks if subdomain is available
func (um *InMemoryURLManager) IsSubdomainAvailable(ctx context.Context, subdomain string) (bool, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()

	// Check validation first
	if err := um.ValidateSubdomain(subdomain); err != nil {
		return false, err
	}

	// Check if reserved
	if reservation, exists := um.reservations[subdomain]; exists {
		// Check if reservation has expired
		if time.Now().After(reservation.ExpiresAt) {
			// Clean up expired reservation
			delete(um.reservations, subdomain)
			return true, nil
		}
		return false, nil
	}

	return true, nil
}

// cleanAppName cleans app name for use in subdomain
func (um *InMemoryURLManager) cleanAppName(appName string) string {
	// Convert to lowercase
	clean := strings.ToLower(appName)

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	clean = reg.ReplaceAllString(clean, "-")

	// Remove leading/trailing hyphens
	clean = strings.Trim(clean, "-")

	// Limit length
	if len(clean) > 20 {
		clean = clean[:20]
	}

	// Ensure it's valid
	if clean == "" {
		clean = "app"
	}

	return clean
}

// generateHash generates a random hex hash
func (um *InMemoryURLManager) generateHash(length int) (string, error) {
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// CleanupExpiredReservations removes expired subdomain reservations
func (um *InMemoryURLManager) CleanupExpiredReservations(ctx context.Context) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()

	now := time.Now()
	for subdomain, reservation := range um.reservations {
		if now.After(reservation.ExpiresAt) {
			delete(um.reservations, subdomain)
		}
	}

	return nil
}

// GetReservations returns all current reservations
func (um *InMemoryURLManager) GetReservations(ctx context.Context) (map[string]*SubdomainReservation, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()

	// Return a copy to avoid race conditions
	reservations := make(map[string]*SubdomainReservation)
	for k, v := range um.reservations {
		reservations[k] = &SubdomainReservation{
			Subdomain: v.Subdomain,
			PreviewID: v.PreviewID,
			CreatedAt: v.CreatedAt,
			ExpiresAt: v.ExpiresAt,
		}
	}

	return reservations, nil
}

// StartCleanupScheduler starts periodic cleanup of expired reservations
func (um *InMemoryURLManager) StartCleanupScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := um.CleanupExpiredReservations(ctx); err != nil {
					// Log error (in production, use proper logging)
					fmt.Printf("URL manager cleanup error: %v\n", err)
				}
			}
		}
	}()
}

// RedisURLManager implements URLManager interface with Redis storage
type RedisURLManager struct {
	config *PreviewConfig
	// Redis client would go here in a real implementation
	// client redis.Client
}

// NewRedisURLManager creates a new Redis-backed URL manager
func NewRedisURLManager(config *PreviewConfig) *RedisURLManager {
	return &RedisURLManager{
		config: config,
	}
}

// Implement URLManager interface methods for Redis
// (These would use Redis commands for persistence)

func (rum *RedisURLManager) GenerateSubdomain(appName string) (string, error) {
	// Implementation using Redis for persistence
	return "", fmt.Errorf("Redis URL manager not implemented")
}

func (rum *RedisURLManager) ReserveSubdomain(ctx context.Context, subdomain string, previewID string, ttl time.Duration) error {
	// Implementation using Redis SETEX or similar
	return fmt.Errorf("Redis URL manager not implemented")
}

func (rum *RedisURLManager) ReleaseSubdomain(ctx context.Context, subdomain string) error {
	// Implementation using Redis DEL
	return fmt.Errorf("Redis URL manager not implemented")
}

func (rum *RedisURLManager) GetURL(subdomain string, tls bool) string {
	scheme := "http"
	if tls {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s.%s", scheme, subdomain, rum.config.BaseDomain)
}

func (rum *RedisURLManager) ValidateSubdomain(subdomain string) error {
	// Same validation logic as InMemoryURLManager
	if len(subdomain) == 0 || len(subdomain) > 63 {
		return fmt.Errorf("subdomain length must be 1-63 characters")
	}

	if !regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`).MatchString(subdomain) {
		return fmt.Errorf("subdomain must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

func (rum *RedisURLManager) IsSubdomainAvailable(ctx context.Context, subdomain string) (bool, error) {
	// Implementation using Redis EXISTS
	return false, fmt.Errorf("Redis URL manager not implemented")
}