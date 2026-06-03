package vm

import (
	// Note: AX-6 — text/tabwriter is structural for CLI table formatting; no core primitive.
	"text/tabwriter"

	core "dappco.re/go"
	"dappco.re/go/container"
)

// requireApple returns an available AppleProvider, or a Fail explaining that
// the macOS Containerisation runtime is required. Image commands are Apple-only
// — LinuxKit has no OCI image management.
//
// Usage:
//
//	r := requireApple(); if !r.OK { return r }; p := core.MustCast[*container.AppleProvider](r)
func requireApple() core.Result { // Value: *container.AppleProvider
	p := container.NewAppleProvider()
	if !p.Available() {
		return core.Fail(core.E("vm", "the apple container runtime is not available on this host (requires macOS 26+ and the `container` CLI; run `container system start`)", nil))
	}
	return core.Ok(p)
}

// addVMBuildCommand adds the 'build' command under vm.
func addVMBuildCommand(c *core.Core) {
	registerVMCommand(c, "vm/build", core.Command{
		Description: "Build an OCI image from a Containerfile (apple runtime)",
		Flags: core.NewOptions(
			core.Option{Key: "tag", Value: ""},
			core.Option{Key: "file", Value: ""},
		),
		Action: func(opts core.Options) core.Result {
			dir := "."
			if args := optionArgs(opts); len(args) > 0 {
				dir = args[0]
			}
			return buildImage(dir, opts.String("file"), opts.String("tag"))
		},
	})
}

func buildImage(dir, file, tag string) core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	source := dir
	if file != "" {
		source = file
	}
	br := p.Build(container.ContainerConfig{Source: source, Name: tag})
	if !br.OK {
		return br
	}
	img := core.MustCast[*container.Image](br)
	core.Print(nil, "%s %s", successStyle.Render("built"), img.Path)
	core.Println()
	return core.Ok(nil)
}

// addVMPullCommand adds the 'pull' command under vm.
func addVMPullCommand(c *core.Core) {
	registerVMCommand(c, "vm/pull", core.Command{
		Description: "Pull an image from a registry (apple runtime)",
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm pull", "image reference is required", nil))
			}
			return pullImage(args[0])
		},
	})
}

func pullImage(ref string) core.Result {
	if ref == "" {
		return core.Fail(core.E("vm pull", "image reference is required", nil))
	}
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	pr := p.Pull(ref)
	if !pr.OK {
		return pr
	}
	img := core.MustCast[*container.Image](pr)
	core.Print(nil, "%s %s %s", successStyle.Render("pulled"), img.Name, img.Digest)
	core.Println()
	return core.Ok(nil)
}

// addVMPushCommand adds the 'push' command under vm.
func addVMPushCommand(c *core.Core) {
	registerVMCommand(c, "vm/push", core.Command{
		Description: "Push a locally-tagged image to a registry (apple runtime)",
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm push", "image reference is required", nil))
			}
			return pushImage(args[0])
		},
	})
}

func pushImage(ref string) core.Result {
	if ref == "" {
		return core.Fail(core.E("vm push", "image reference is required", nil))
	}
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	pushRes := p.Push(&container.Image{Path: ref}, ref)
	if !pushRes.OK {
		return pushRes
	}
	core.Print(nil, "%s %s", successStyle.Render("pushed"), ref)
	core.Println()
	return core.Ok(nil)
}

// addVMImagesCommand adds the 'images' command under vm.
func addVMImagesCommand(c *core.Core) {
	registerVMCommand(c, "vm/images", core.Command{
		Description: "List images (apple runtime)",
		Action: func(opts core.Options) core.Result {
			return listImages()
		},
	})
}

func listImages() core.Result {
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	lr := p.ListImages()
	if !lr.OK {
		return lr
	}
	imgs := core.MustCast[[]*container.Image](lr)
	if len(imgs) == 0 {
		core.Println("no images")
		return core.Ok(nil)
	}
	core.Print(nil, "%s", formatImages(imgs))
	return core.Ok(nil)
}

// formatImages renders images as a REPOSITORY/DIGEST table (digest shortened).
func formatImages(imgs []*container.Image) string {
	var b core.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	core.Print(w, "%s", "REPOSITORY\tDIGEST")
	for _, img := range imgs {
		digest := img.Digest
		if len(digest) > 19 {
			digest = digest[:19]
		}
		core.Print(w, "%s\t%s", img.Name, digest)
	}
	_ = w.Flush()
	return b.String()
}

// addVMRmiCommand adds the 'rmi' command under vm.
func addVMRmiCommand(c *core.Core) {
	registerVMCommand(c, "vm/rmi", core.Command{
		Description: "Remove an image (apple runtime)",
		Action: func(opts core.Options) core.Result {
			args := optionArgs(opts)
			if len(args) == 0 {
				return core.Fail(core.E("vm rmi", "image reference is required", nil))
			}
			return removeImage(args[0])
		},
	})
}

func removeImage(ref string) core.Result {
	if ref == "" {
		return core.Fail(core.E("vm rmi", "image reference is required", nil))
	}
	r := requireApple()
	if !r.OK {
		return r
	}
	p := core.MustCast[*container.AppleProvider](r)
	rmRes := p.RemoveImage(ref)
	if !rmRes.OK {
		return rmRes
	}
	core.Print(nil, "%s %s", successStyle.Render("removed"), ref)
	core.Println()
	return core.Ok(nil)
}
