package platform

import (
	"runtime"

	"github.com/platform-engineering-labs/orbital/platform/arch"
	"github.com/platform-engineering-labs/orbital/platform/os"
)

var SupportedPlatforms = []*Platform{
	{os.Linux, arch.X8664},
	{os.Linux, arch.All},
	{os.Linux, arch.Arm64},
	{os.Darwin, arch.All},
	{os.Darwin, arch.Arm64},
	{os.Darwin, arch.X8664},
	{os.All, arch.All},
}

type Platform struct {
	OS   os.OS
	Arch arch.Arch
}

func (p *Platform) Equal(pltfrm *Platform) bool {
	return p.OS == pltfrm.OS && p.Arch == pltfrm.Arch
}

func (p *Platform) String() string {
	return p.OS.String() + "-" + p.Arch.String()
}

func (p *Platform) Supported() bool {
	for _, supported := range SupportedPlatforms {
		if p.OS == supported.OS && p.Arch == supported.Arch {
			return true
		}
	}

	return false
}

func Current() *Platform {
	p := Platform{}

	switch runtime.GOOS {
	case "darwin":
		p.OS = os.Darwin
	case "linux":
		p.OS = os.Linux
	}

	switch runtime.GOARCH {
	case "amd64":
		p.Arch = arch.X8664
	case "arm64":
		p.Arch = arch.Arm64
	}

	return &p
}

func Expanded(current *Platform) []*Platform {
	var platforms []*Platform

	if current == nil {
		current = Current()
	}

	platforms = append(platforms, current)
	platforms = append(platforms, &Platform{os.All, arch.All})
	platforms = append(platforms, &Platform{current.OS, arch.All})
	platforms = append(platforms, &Platform{os.All, current.Arch})

	return platforms
}
