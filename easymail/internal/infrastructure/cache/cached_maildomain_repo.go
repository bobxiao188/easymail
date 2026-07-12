package cache

import (
	"context"
	"encoding/json"
	"time"

	"easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/timeutil"

	"golang.org/x/sync/singleflight"
)

func domainCacheKey(scope, key string) string {
	return "domain:" + scope + ":" + key
}

// CachedMailDomainRepository wraps a management.MailDomainRepository with
// a pluggable CacheBackend. Business logic is unaware of caching.
type CachedMailDomainRepository struct {
	repo  management.MailDomainRepository
	cache CacheBackend
	ttl   time.Duration
	sf    singleflight.Group
}

// NewCachedMailDomainRepository creates a cache-decorated repository.
// When ttl <= 0, caching is disabled and all calls pass through to the wrapped repo.
func NewCachedMailDomainRepository(repo management.MailDomainRepository, cache CacheBackend, ttl time.Duration) *CachedMailDomainRepository {
	return &CachedMailDomainRepository{repo: repo, cache: cache, ttl: ttl}
}

func (r *CachedMailDomainRepository) isEnabled() bool {
	return r.ttl > 0 && r.cache != nil
}

// ---- Read-through cache (cache-aside) ----

func (r *CachedMailDomainRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.MailDomain, error) {
	if !r.isEnabled() {
		return r.repo.FindByID(ctx, id)
	}
	key := domainCacheKey("id", id.String())
	if d := r.getFromCache(ctx, key); d != nil {
		return d, nil
	}
	// Singleflight: coalesce concurrent cache misses for the same key.
	v, err, _ := r.sf.Do(key, func() (interface{}, error) {
		// Double-check cache after acquiring the singleflight lock.
		if d := r.getFromCache(ctx, key); d != nil {
			return d, nil
		}
		return r.repo.FindByID(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	d := v.(*management.MailDomain)
	r.setToCache(ctx, key, d)
	return d, nil
}

func (r *CachedMailDomainRepository) FindByName(ctx context.Context, name string) (*management.MailDomain, error) {
	if !r.isEnabled() {
		return r.repo.FindByName(ctx, name)
	}
	key := domainCacheKey("name", name)
	if d := r.getFromCache(ctx, key); d != nil {
		return d, nil
	}
	v, err, _ := r.sf.Do(key, func() (interface{}, error) {
		if d := r.getFromCache(ctx, key); d != nil {
			return d, nil
		}
		return r.repo.FindByName(ctx, name)
	})
	if err != nil {
		return nil, err
	}
	d := v.(*management.MailDomain)
	r.setToCache(ctx, key, d)
	return d, nil
}

func (r *CachedMailDomainRepository) FindValidatedByName(ctx context.Context, name string) (*management.MailDomain, error) {
	if !r.isEnabled() {
		return r.repo.FindValidatedByName(ctx, name)
	}
	key := domainCacheKey("validated:name", name)
	if d := r.getFromCache(ctx, key); d != nil {
		return d, nil
	}
	v, err, _ := r.sf.Do(key, func() (interface{}, error) {
		if d := r.getFromCache(ctx, key); d != nil {
			return d, nil
		}
		return r.repo.FindValidatedByName(ctx, name)
	})
	if err != nil {
		return nil, err
	}
	d := v.(*management.MailDomain)
	r.setToCache(ctx, key, d)
	return d, nil
}

// ---- Passthrough methods (not cached) ----

func (r *CachedMailDomainRepository) FindAllValidated(ctx context.Context) ([]management.MailDomain, error) {
	return r.repo.FindAllValidated(ctx)
}

func (r *CachedMailDomainRepository) Search(ctx context.Context, keyword string, page, pageSize int, includeDeleted bool) ([]management.MailDomain, int64, error) {
	return r.repo.Search(ctx, keyword, page, pageSize, includeDeleted)
}

// ---- Write-through + cache invalidation ----

func (r *CachedMailDomainRepository) Save(ctx context.Context, d *management.MailDomain) error {
	// Cache the old name before save for proper invalidation after rename.
	oldName, _ := r.loadOldName(ctx, d.ID)

	if err := r.repo.Save(ctx, d); err != nil {
		return err
	}
	r.invalidate(ctx, d.ID, oldName, d.Name)
	return nil
}

func (r *CachedMailDomainRepository) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	oldName, _ := r.loadOldName(ctx, id)

	if err := r.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	r.invalidate(ctx, id, oldName, "")
	return nil
}

func (r *CachedMailDomainRepository) HardDelete(ctx context.Context, id shared.GlobalID) error {
	oldName, _ := r.loadOldName(ctx, id)

	if err := r.repo.HardDelete(ctx, id); err != nil {
		return err
	}
	r.invalidate(ctx, id, oldName, "")
	return nil
}

func (r *CachedMailDomainRepository) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	oldName, _ := r.loadOldName(ctx, id)

	if err := r.repo.ToggleActive(ctx, id); err != nil {
		return err
	}
	r.invalidate(ctx, id, oldName, "")
	return nil
}

// loadOldName retrieves the current name from cache or DB for cache invalidation.
func (r *CachedMailDomainRepository) loadOldName(ctx context.Context, id shared.GlobalID) (string, bool) {
	if !r.isEnabled() {
		return "", false
	}
	// Prefer cache: it has the pre-rename name.
	key := domainCacheKey("id", id.String())
	if d := r.getFromCache(ctx, key); d != nil {
		return d.Name, true
	}
	// Fallback: read from DB without cache to get the old name.
	d, err := r.repo.FindByID(ctx, id)
	if err != nil {
		return "", false
	}
	return d.Name, true
}

// ---- Cache serialization ----

// jsonMailDomain is a JSON-friendly representation of MailDomain.
// All time fields use timeutil for consistent formatting.
type jsonMailDomain struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Active         bool   `json:"active"`
	IsDeleted      bool   `json:"isDeleted"`
	DKIMEnabled    bool   `json:"dkimEnabled"`
	DKIMSelector   string `json:"dkimSelector"`
	DKIMPrivateKey string `json:"dkimPrivateKey"`
	CreateTime     string `json:"createTime"`
	UpdateTime     string `json:"updateTime"`
	DeleteTime     string `json:"deleteTime"`
}

func toJSON(d *management.MailDomain) jsonMailDomain {
	return jsonMailDomain{
		ID:             d.ID.String(),
		Name:           d.Name,
		Description:    d.Description,
		Active:         d.Active,
		IsDeleted:      d.IsDeleted,
		DKIMEnabled:    d.DKIMEnabled,
		DKIMSelector:   d.DKIMSelector,
		DKIMPrivateKey: d.DKIMPrivateKey,
		CreateTime:     timeutil.FormatJSON(d.CreateTime),
		UpdateTime:     timeutil.FormatJSON(d.UpdateTime),
		DeleteTime:     timeutil.FormatJSON(d.DeleteTime),
	}
}

func fromJSON(j jsonMailDomain) *management.MailDomain {
	createTime, _ := timeutil.ParseJSON(j.CreateTime)
	updateTime, _ := timeutil.ParseJSON(j.UpdateTime)
	deleteTime, _ := timeutil.ParseJSON(j.DeleteTime)
	id, _ := shared.ParseGlobalID(j.ID)
	return &management.MailDomain{
		ID:             id,
		Name:           j.Name,
		Description:    j.Description,
		Active:         j.Active,
		IsDeleted:      j.IsDeleted,
		DKIMEnabled:    j.DKIMEnabled,
		DKIMSelector:   j.DKIMSelector,
		DKIMPrivateKey: j.DKIMPrivateKey,
		CreateTime:     createTime,
		UpdateTime:     updateTime,
		DeleteTime:     deleteTime,
	}
}

func (r *CachedMailDomainRepository) getFromCache(ctx context.Context, key string) *management.MailDomain {
	data, found, err := r.cache.Get(ctx, key)
	if err != nil || !found {
		return nil
	}
	var j jsonMailDomain
	if json.Unmarshal(data, &j) != nil {
		return nil
	}
	return fromJSON(j)
}

func (r *CachedMailDomainRepository) setToCache(ctx context.Context, key string, d *management.MailDomain) {
	data, err := json.Marshal(toJSON(d))
	if err != nil {
		return
	}
	_ = r.cache.Set(ctx, key, data, r.ttl)
}

func (r *CachedMailDomainRepository) invalidate(ctx context.Context, id shared.GlobalID, oldName, newName string) {
	keys := []string{domainCacheKey("id", id.String())}
	if oldName != "" {
		keys = append(keys, domainCacheKey("name", oldName))
		keys = append(keys, domainCacheKey("validated:name", oldName))
	}
	// If the name changed, also invalidate the new name (in case it was cached under a previous entity).
	if newName != "" && newName != oldName {
		keys = append(keys, domainCacheKey("name", newName))
		keys = append(keys, domainCacheKey("validated:name", newName))
	}
	_ = r.cache.Del(ctx, keys...)
}

// Compile-time check: CachedMailDomainRepository implements MailDomainRepository.
var _ management.MailDomainRepository = (*CachedMailDomainRepository)(nil)
