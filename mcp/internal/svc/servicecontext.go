package svc

import (
	"ai-gozero-agent/mcp/internal/config"
	"fmt"
	"github.com/unidoc/unipdf/v3/common/license"
)

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	err := license.SetMeteredKey(c.UniPDFLicense)
	if err != nil {
		fmt.Printf("license metered key error: %v\n", err)
	}
	return &ServiceContext{
		Config: c,
	}
}
