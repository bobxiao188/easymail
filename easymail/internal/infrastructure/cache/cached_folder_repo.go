package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"easymail/internal/domain/mailbox"
	"easymail/internal/domain/shared"
	"easymail/pkg/timeutil"

	"golang.org/x/sync/singleflight"
)

func folderCacheKey(scope, key string) string {
	return "folder:" + scope + ":" + key
}

// CachedFolderRepository wraps a mailbox.FolderRepository with caching.
type CachedFolderRepository struct {
	repo  mailbox.FolderRepository
	cache CacheBackend
	ttl   time.Duration
	sf    singleflight.Group
}

func NewCachedFolderRepository(repo mailbox.FolderRepository, cache CacheBackend, ttl time.Duration) *CachedFolderRepository {
	return &CachedFolderRepository{repo: repo, cache: cache, ttl: ttl}
}

func (r *CachedFolderRepository) isEnabled() bool {
	return r.ttl > 0 && r.cache != nil
}

// ---- Read-through cache ----

func (r *CachedFolderRepository) FindByID(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) (*mailbox.Folder, error) {
	if !r.isEnabled() {
		return r.repo.FindByID(ctx, mailUserID, id)
	}
	key := folderCacheKey("id", id.String())
	if d := r.getFromCache(ctx, key); d != nil {
		return d, nil
	}
	v, err, _ := r.sf.Do(key, func() (interface{}, error) {
		if d := r.getFromCache(ctx, key); d != nil {
			return d, nil
		}
		return r.repo.FindByID(ctx, mailUserID, id)
	})
	if err != nil {
		return nil, err
	}
	d := v.(*mailbox.Folder)
	r.setToCache(ctx, key, d)
	return d, nil
}

func (r *CachedFolderRepository) FindByMailUserAndKind(ctx context.Context, mailUserID shared.GlobalID, kind mailbox.FolderKind) (*mailbox.Folder, error) {
	if !r.isEnabled() {
		return r.repo.FindByMailUserAndKind(ctx, mailUserID, kind)
	}
	key := folderCacheKey("kind", mailUserID.String()+":"+fmt.Sprintf("%d", kind))
	if d := r.getFromCache(ctx, key); d != nil {
		return d, nil
	}
	v, err, _ := r.sf.Do(key, func() (interface{}, error) {
		if d := r.getFromCache(ctx, key); d != nil {
			return d, nil
		}
		return r.repo.FindByMailUserAndKind(ctx, mailUserID, kind)
	})
	if err != nil {
		return nil, err
	}
	d := v.(*mailbox.Folder)
	r.setToCache(ctx, key, d)
	return d, nil
}

// ---- Passthrough ----

func (r *CachedFolderRepository) ListByMailUser(ctx context.Context, mailUserID shared.GlobalID) ([]*mailbox.Folder, error) {
	return r.repo.ListByMailUser(ctx, mailUserID)
}

// ---- Write-through + invalidation ----

func (r *CachedFolderRepository) Save(ctx context.Context, f *mailbox.Folder) error {
	oldID := f.ID
	if err := r.repo.Save(ctx, f); err != nil {
		return err
	}
	r.invalidate(ctx, oldID, f.MailUserID)
	return nil
}

func (r *CachedFolderRepository) Update(ctx context.Context, f *mailbox.Folder) error {
	if err := r.repo.Update(ctx, f); err != nil {
		return err
	}
	r.invalidate(ctx, f.ID, f.MailUserID)
	return nil
}

func (r *CachedFolderRepository) UpdateName(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID, name string) error {
	if err := r.repo.UpdateName(ctx, mailUserID, id, name); err != nil {
		return err
	}
	r.invalidate(ctx, id, mailUserID)
	return nil
}

func (r *CachedFolderRepository) SoftDelete(ctx context.Context, mailUserID shared.GlobalID, id shared.GlobalID) error {
	if err := r.repo.SoftDelete(ctx, mailUserID, id); err != nil {
		return err
	}
	r.invalidate(ctx, id, mailUserID)
	return nil
}

// ---- Cache serialization ----

type jsonFolder struct {
	ID         string `json:"id"`
	MailUserID  string `json:"mailboxId"`
	FolderName string `json:"folderName"`
	FolderKind int    `json:"folderKind"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

func toJSONFolder(f *mailbox.Folder) jsonFolder {
	return jsonFolder{
		ID:         f.ID.String(),
		MailUserID:  f.MailUserID.String(),
		FolderName: f.FolderName,
		FolderKind: int(f.FolderKind),
		CreateTime: timeutil.FormatJSON(f.CreateTime),
		UpdateTime: timeutil.FormatJSON(f.UpdateTime),
	}
}

func fromJSONFolder(j jsonFolder) *mailbox.Folder {
	createTime, _ := timeutil.ParseJSON(j.CreateTime)
	updateTime, _ := timeutil.ParseJSON(j.UpdateTime)
	id, _ := shared.ParseGlobalID(j.ID)
	mailUserID, _ := shared.ParseGlobalID(j.MailUserID)
	return &mailbox.Folder{
		ID:         id,
		MailUserID:  mailUserID,
		FolderName: j.FolderName,
		FolderKind: mailbox.FolderKind(j.FolderKind),
		CreateTime: createTime,
		UpdateTime: updateTime,
	}
}

func (r *CachedFolderRepository) getFromCache(ctx context.Context, key string) *mailbox.Folder {
	data, found, err := r.cache.Get(ctx, key)
	if err != nil || !found {
		return nil
	}
	var j jsonFolder
	if json.Unmarshal(data, &j) != nil {
		return nil
	}
	return fromJSONFolder(j)
}

func (r *CachedFolderRepository) setToCache(ctx context.Context, key string, f *mailbox.Folder) {
	data, err := json.Marshal(toJSONFolder(f))
	if err != nil {
		return
	}
	_ = r.cache.Set(ctx, key, data, r.ttl)
}

func (r *CachedFolderRepository) invalidate(ctx context.Context, id shared.GlobalID, mailUserID shared.GlobalID) {
	keys := []string{folderCacheKey("id", id.String())}
	if mailUserID != "" {
		keys = append(keys, folderCacheKey("kind", mailUserID.String()+":"))
	}
	_ = r.cache.Del(ctx, keys...)
}

var _ mailbox.FolderRepository = (*CachedFolderRepository)(nil)
