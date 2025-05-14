package test

import (
	"testing"

	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/service"
)

func TestMastodonService_TimelineToRSS(t *testing.T) {
	svc := service.NewMastodonService()
	// 由于没有真实 API，这里只测试参数校验分支
	_, err := svc.TimelineToRSS(conf.Feed{})
	if err == nil {
		t.Error("expected error for empty config")
	}
	// TODO: 可用 httptest.Server mock Mastodon API 进一步测试
}
