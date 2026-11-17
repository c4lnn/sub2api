//go:build integration

package repository

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type UpdateCacheSuite struct {
	IntegrationRedisSuite
	cache *updateCache
}

func (s *UpdateCacheSuite) SetupTest() {
	s.IntegrationRedisSuite.SetupTest()
	s.cache = NewUpdateCache(s.rdb).(*updateCache)
}

type failingUpdateReleaseClient struct {
	latestRepo string
}

func (c *failingUpdateReleaseClient) FetchLatestRelease(_ context.Context, repo string) (*service.GitHubRelease, error) {
	c.latestRepo = repo
	return nil, errors.New("release source unavailable")
}

func (*failingUpdateReleaseClient) FetchRecentReleases(context.Context, string, int) ([]*service.GitHubRelease, error) {
	return nil, errors.New("not used")
}

func (*failingUpdateReleaseClient) DownloadFile(context.Context, string, string, int64) error {
	return errors.New("not used")
}

func (*failingUpdateReleaseClient) FetchChecksumFile(context.Context, string) ([]byte, error) {
	return nil, errors.New("not used")
}

func (s *UpdateCacheSuite) TestCheckUpdateFailureDoesNotReuseLegacyCache() {
	legacyInfo := `{"latest":"99.0.0","release_info":{"html_url":"https://github.com/legacy/repo/releases/tag/v99.0.0"},"timestamp":` + strconv.FormatInt(time.Now().Unix(), 10) + `}`
	require.NoError(s.T(), s.rdb.Set(s.ctx, "update:latest", legacyInfo, 5*time.Minute).Err())
	githubClient := &failingUpdateReleaseClient{}
	updateService := service.NewUpdateService(s.cache, githubClient, "2026.11.17", "release")

	info, err := updateService.CheckUpdate(s.ctx, true)

	require.NoError(s.T(), err)
	require.Equal(s.T(), "c4lnn/sub2api", githubClient.latestRepo)
	require.Equal(s.T(), "2026.11.17", info.LatestVersion)
	require.False(s.T(), info.HasUpdate)
	require.Contains(s.T(), info.Warning, "release source unavailable")
	require.False(s.T(), info.Cached)
	storedLegacy, err := s.rdb.Get(s.ctx, "update:latest").Result()
	require.NoError(s.T(), err)
	require.Equal(s.T(), legacyInfo, storedLegacy)
}

func (s *UpdateCacheSuite) TestGetUpdateInfo_Missing() {
	_, err := s.cache.GetUpdateInfo(s.ctx)
	require.True(s.T(), errors.Is(err, redis.Nil), "expected redis.Nil for missing update info")
}

func (s *UpdateCacheSuite) TestGetUpdateInfo_IgnoresLegacyKey() {
	require.NoError(s.T(), s.rdb.Set(s.ctx, "update:latest", "legacy-release", 5*time.Minute).Err())

	_, err := s.cache.GetUpdateInfo(s.ctx)

	require.True(s.T(), errors.Is(err, redis.Nil), "legacy update cache must not be reused")
	legacy, legacyErr := s.rdb.Get(s.ctx, "update:latest").Result()
	require.NoError(s.T(), legacyErr)
	require.Equal(s.T(), "legacy-release", legacy)
}

func (s *UpdateCacheSuite) TestSetAndGetUpdateInfo() {
	updateTTL := 5 * time.Minute
	require.NoError(s.T(), s.cache.SetUpdateInfo(s.ctx, "v1.2.3", updateTTL), "SetUpdateInfo")

	info, err := s.cache.GetUpdateInfo(s.ctx)
	require.NoError(s.T(), err, "GetUpdateInfo")
	require.Equal(s.T(), "v1.2.3", info, "update info mismatch")
}

func (s *UpdateCacheSuite) TestSetUpdateInfo_TTL() {
	updateTTL := 5 * time.Minute
	require.NoError(s.T(), s.cache.SetUpdateInfo(s.ctx, "v1.2.3", updateTTL))

	ttl, err := s.rdb.TTL(s.ctx, updateCacheKey).Result()
	require.NoError(s.T(), err, "TTL updateCacheKey")
	s.AssertTTLWithin(ttl, 1*time.Second, updateTTL)
}

func (s *UpdateCacheSuite) TestSetUpdateInfo_Overwrite() {
	require.NoError(s.T(), s.cache.SetUpdateInfo(s.ctx, "v1.0.0", 5*time.Minute))
	require.NoError(s.T(), s.cache.SetUpdateInfo(s.ctx, "v2.0.0", 5*time.Minute))

	info, err := s.cache.GetUpdateInfo(s.ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "v2.0.0", info, "expected overwritten value")
}

func (s *UpdateCacheSuite) TestSetUpdateInfo_ZeroTTL() {
	// TTL=0 means persist forever (no expiry) in Redis SET command
	require.NoError(s.T(), s.cache.SetUpdateInfo(s.ctx, "v0.0.0", 0))

	info, err := s.cache.GetUpdateInfo(s.ctx)
	require.NoError(s.T(), err)
	require.Equal(s.T(), "v0.0.0", info)

	ttl, err := s.rdb.TTL(s.ctx, updateCacheKey).Result()
	require.NoError(s.T(), err)
	// TTL=-1 means no expiry, TTL=-2 means key doesn't exist
	require.Equal(s.T(), time.Duration(-1), ttl, "expected TTL=-1 for key with no expiry")
}

func TestUpdateCacheSuite(t *testing.T) {
	suite.Run(t, new(UpdateCacheSuite))
}
