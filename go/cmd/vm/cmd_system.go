package vm

import (
	core "dappco.re/go"
	"dappco.re/go/container"
)

// addVMSystemCommand registers the `vm system` subgroup (start/status/stop) for
// the Apple container runtime, mirroring the `vm templates` subgroup.
func addVMSystemCommand(c *core.Core) {
	registerVMCommand(c, "vm/system", core.Command{
		Description: "Manage the Apple container system services",
	})
	registerVMCommand(c, "vm/system/start", core.Command{
		Description: "Start the Apple container system (apiserver + default kernel)",
		Flags:       core.NewOptions(core.Option{Key: "no-kernel-install", Value: false}),
		Action: func(opts core.Options) core.Result {
			return systemStart(!opts.Bool("no-kernel-install"))
		},
	})
	registerVMCommand(c, "vm/system/status", core.Command{
		Description: "Show Apple container system status",
		Action: func(opts core.Options) core.Result {
			return systemStatus()
		},
	})
	registerVMCommand(c, "vm/system/stop", core.Command{
		Description: "Stop the Apple container system services",
		Action: func(opts core.Options) core.Result {
			return systemStop()
		},
	})
}

func systemStart(installKernel bool) core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	sr := core.MustCast[*container.AppleProvider](r).SystemStart(installKernel)
	if !sr.OK {
		return sr
	}
	core.Print(nil, "%s", successStyle.Render("system started"))
	core.Println()
	return core.Ok(nil)
}

func systemStatus() core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	sr := core.MustCast[*container.AppleProvider](r).SystemStatus()
	if !sr.OK {
		return sr
	}
	core.Println(core.MustCast[string](sr))
	return core.Ok(nil)
}

func systemStop() core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	sr := core.MustCast[*container.AppleProvider](r).SystemStop()
	if !sr.OK {
		return sr
	}
	core.Print(nil, "%s", successStyle.Render("system stopped"))
	core.Println()
	return core.Ok(nil)
}
