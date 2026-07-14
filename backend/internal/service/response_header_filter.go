package service

import (
	"github.com/CERNss/sub2api-lite/internal/config"
	"github.com/CERNss/sub2api-lite/internal/util/responseheaders"
)

func compileResponseHeaderFilter(cfg *config.Config) *responseheaders.CompiledHeaderFilter {
	if cfg == nil {
		return nil
	}
	return responseheaders.CompileHeaderFilter(cfg.Security.ResponseHeaders)
}
