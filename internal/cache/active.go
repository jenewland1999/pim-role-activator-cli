package cache

import (
	"encoding/json"
	"time"

	"github.com/jenewland1999/pim-role-activator-cli/internal/model"
)

const activeRolesPrefix = "active-roles"

// LoadActiveRoles reads the active-roles cache, prunes expired entries, and
// returns the remaining roles. The second return value is false when the cache
// is missing, expired, or corrupt.
func LoadActiveRoles(dir string, ttl time.Duration) ([]model.ActiveRole, bool) {
	c := New(dir, ttl, activeRolesPrefix)
	data, ok := c.Get()
	if !ok {
		return nil, false
	}

	var cached []model.CachedActiveRole
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	live := model.FromCachedRoles(cached)
	if len(live) == 0 {
		return nil, false
	}
	return live, true
}

// SaveActiveRoles writes active roles to the cache. The TTL is set to the
// minimum remaining expiry across all roles (so the cache auto-invalidates
// when the first role expires), capped at maxTTL.
func SaveActiveRoles(dir string, maxTTL time.Duration, roles []model.ActiveRole) error {
	cached := model.ToCachedRoles(roles)

	// Compute a dynamic TTL based on the soonest-expiring role.
	ttl := maxTTL
	now := time.Now()
	for _, c := range cached {
		remaining := c.ExpiresAt.Sub(now)
		if remaining > 0 && remaining < ttl {
			ttl = remaining
		}
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	c := New(dir, ttl, activeRolesPrefix)
	return c.Set(data)
}
