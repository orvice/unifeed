package test

import (
	"testing"

	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/service"
)

func TestBlueskyService_TimelineToRSS(t *testing.T) {
	svc := service.NewBlueskyService()
	_, err := svc.TimelineToRSS(conf.Feed{})
	if err == nil {
		t.Error("expected error for empty config")
	}
	// TODO: 可用 httptest.Server mock Bluesky API 进一步测试
}
