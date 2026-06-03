# Sonar sweeps — core_go-container findings

300 findings across 8 rules. One rule per commit; fix every line listed under each rule.

## CRITICAL

### go:S1192 — String literals should not be duplicated (277×, code smell)

- `go/apple.go:115` — Define a constant instead of duplicating this literal "AppleProvider.Build" 5 times.
- `go/apple.go:125` — Define a constant instead of duplicating this literal "generate image id" 4 times.
- `go/apple.go:209` — Define a constant instead of duplicating this literal "AppleProvider.Run" 5 times.
- `go/apple.go:212` — Define a constant instead of duplicating this literal "image is required" 3 times.
- `go/apple.go:368` — Define a constant instead of duplicating this literal "AppleProvider.Encrypt" 8 times.
- `go/apple.go:425` — Define a constant instead of duplicating this literal "AppleProvider.Decrypt" 9 times.
- `go/apple.go:486` — Define a constant instead of duplicating this literal "container id is required" 6 times.
- `go/apple.go:576` — Define a constant instead of duplicating this literal "AppleProvider.Exec" 3 times.
- `go/apple.go:641` — Define a constant instead of duplicating this literal "AppleProvider.Pull" 3 times.
- `go/apple.go:672` — Define a constant instead of duplicating this literal "AppleProvider.Push" 3 times.
- `go/apple_test.go:26` — Define a constant instead of duplicating this literal "expected false" 3 times.
- `go/apple_test.go:39` — Define a constant instead of duplicating this literal "expected non-nil value" 3 times.
- `go/apple_test.go:42` — Define a constant instead of duplicating this literal "want %v, got %v" 4 times.
- `go/apple_test.go:67` — Define a constant instead of duplicating this literal "apple container runtime not available" 6 times.
- `go/apple_test.go:72` — Define a constant instead of duplicating this literal "expected error" 16 times.
- `go/apple_test.go:236` — Define a constant instead of duplicating this literal "expected symbol linked" 25 times.
- `go/apple_test.go:239` — Define a constant instead of duplicating this literal "expected callable symbol" 25 times.
- `go/apple_test.go:292` — Define a constant instead of duplicating this literal "AppleProvider Available" 3 times.
- `go/apple_test.go:340` — Define a constant instead of duplicating this literal "AppleProvider Build" 3 times.
- `go/apple_test.go:388` — Define a constant instead of duplicating this literal "AppleProvider Run" 3 times.
- `go/apple_test.go:436` — Define a constant instead of duplicating this literal "AppleProvider Tracked" 3 times.
- `go/apple_test.go:484` — Define a constant instead of duplicating this literal "AppleProvider Wait" 3 times.
- `go/apple_test.go:532` — Define a constant instead of duplicating this literal "AppleProvider Encrypt" 3 times.
- `go/apple_test.go:580` — Define a constant instead of duplicating this literal "AppleProvider Decrypt" 3 times.
- `go/cmd/vm/cmd_container.go:41` — Define a constant instead of duplicating this literal "ssh-port" 3 times.
- `go/cmd/vm/cmd_container.go:147` — Define a constant instead of duplicating this literal "container manager" 5 times.
- `go/cmd/vm/cmd_container.go:147` — Define a constant instead of duplicating this literal "i18n.fail.init" 5 times.
- `go/cmd/vm/cmd_templates.go:101` — Define a constant instead of duplicating this literal "common.label.template" 3 times.
- `go/cmd/vm/cmd_templates.go:157` — Define a constant instead of duplicating this literal "common.error.failed" 5 times.
- `go/datacube_test.go:17` — Define a constant instead of duplicating this literal "worker-01" 7 times.
- `go/datacube_test.go:17` — Define a constant instead of duplicating this literal "workspace-key" 4 times.
- `go/datacube_test.go:22` — Define a constant instead of duplicating this literal "want %v, got %v" 3 times.
- `go/datacube_test.go:37` — Define a constant instead of duplicating this literal "expected error" 3 times.
- `go/datacube_test.go:71` — Define a constant instead of duplicating this literal "port: 8080" 3 times.
- `go/datacube_test.go:71` — Define a constant instead of duplicating this literal "app/config.yml" 3 times.
- `go/datacube_test.go:203` — Define a constant instead of duplicating this literal "expected symbol linked" 56 times.
- `go/datacube_test.go:206` — Define a constant instead of duplicating this literal "expected callable symbol" 56 times.
- `go/datacube_test.go:227` — Define a constant instead of duplicating this literal "DataCube Read" 9 times.
- `go/datacube_test.go:275` — Define a constant instead of duplicating this literal "DataCube Write" 9 times.
- `go/datacube_test.go:323` — Define a constant instead of duplicating this literal "DataCube WriteMode" 9 times.
- `go/datacube_test.go:371` — Define a constant instead of duplicating this literal "DataCube EnsureDir" 9 times.
- `go/datacube_test.go:419` — Define a constant instead of duplicating this literal "DataCube IsFile" 9 times.
- `go/datacube_test.go:467` — Define a constant instead of duplicating this literal "DataCube Delete" 9 times.
- `go/datacube_test.go:515` — Define a constant instead of duplicating this literal "DataCube DeleteAll" 9 times.
- `go/datacube_test.go:563` — Define a constant instead of duplicating this literal "DataCube Rename" 9 times.
- `go/datacube_test.go:611` — Define a constant instead of duplicating this literal "DataCube List" 9 times.
- `go/datacube_test.go:659` — Define a constant instead of duplicating this literal "DataCube Stat" 9 times.
- `go/datacube_test.go:707` — Define a constant instead of duplicating this literal "DataCube Open" 9 times.
- `go/datacube_test.go:755` — Define a constant instead of duplicating this literal "DataCube Create" 9 times.
- `go/datacube_test.go:803` — Define a constant instead of duplicating this literal "DataCube Append" 9 times.
- `go/datacube_test.go:851` — Define a constant instead of duplicating this literal "DataCube ReadStream" 9 times.
- `go/datacube_test.go:899` — Define a constant instead of duplicating this literal "DataCube WriteStream" 9 times.
- `go/datacube_test.go:947` — Define a constant instead of duplicating this literal "DataCube Exists" 9 times.
- `go/datacube_test.go:995` — Define a constant instead of duplicating this literal "DataCube IsDir" 9 times.
- `go/datacube_test.go:1043` — Define a constant instead of duplicating this literal "DataCube Describe" 9 times.
- `go/datanode_test.go:51` — Define a constant instead of duplicating this literal "worker-01" 8 times.
- `go/datanode_test.go:53` — Define a constant instead of duplicating this literal "want %v, got %v" 7 times.
- `go/datanode_test.go:56` — Define a constant instead of duplicating this literal "want same instance" 3 times.
- `go/datanode_test.go:198` — Define a constant instead of duplicating this literal "expected symbol linked" 23 times.
- `go/datanode_test.go:201` — Define a constant instead of duplicating this literal "expected callable symbol" 23 times.
- `go/datanode_test.go:222` — Define a constant instead of duplicating this literal "DataNode WithSigil" 9 times.
- `go/datanode_test.go:270` — Define a constant instead of duplicating this literal "DataNode Build" 9 times.
- `go/datanode_test.go:318` — Define a constant instead of duplicating this literal "DataNode Start" 9 times.
- `go/datanode_test.go:366` — Define a constant instead of duplicating this literal "DataNode Stop" 9 times.
- `go/datanode_test.go:414` — Define a constant instead of duplicating this literal "DataNode Seal" 9 times.
- `go/datanode_test.go:462` — Define a constant instead of duplicating this literal "DataNode Info" 9 times.
- `go/datanode_test.go:510` — Define a constant instead of duplicating this literal "DataNode Uptime" 9 times.
- `go/devenv/claude_test.go:41` — Define a constant instead of duplicating this literal "want %v, got %v" 7 times.
- `go/devenv/claude_test.go:122` — Define a constant instead of duplicating this literal "DevOps Claude" 3 times.
- `go/devenv/claude_test.go:130` — Define a constant instead of duplicating this literal "expected symbol linked" 6 times.
- `go/devenv/claude_test.go:133` — Define a constant instead of duplicating this literal "expected callable symbol" 6 times.
- `go/devenv/claude_test.go:170` — Define a constant instead of duplicating this literal "DevOps CopyGHAuth" 3 times.
- `go/devenv/config_test.go:20` — Define a constant instead of duplicating this literal "want %v, got %v" 30 times.
- `go/devenv/config_test.go:25` — Define a constant instead of duplicating this literal "host-uk/core-images" 3 times.
- `go/devenv/config_test.go:82` — Define a constant instead of duplicating this literal "config.yaml" 5 times.
- `go/devenv/config_test.go:97` — Define a constant instead of duplicating this literal "https://cdn.example.com" 3 times.
- `go/devenv/config_test.go:142` — Define a constant instead of duplicating this literal "owner/repo" 4 times.
- `go/devenv/config_test.go:429` — Define a constant instead of duplicating this literal "expected symbol linked" 14 times.
- `go/devenv/config_test.go:432` — Define a constant instead of duplicating this literal "expected callable symbol" 14 times.
- `go/devenv/config_test.go:501` — Define a constant instead of duplicating this literal "configmedium Read" 3 times.
- `go/devenv/config_test.go:549` — Define a constant instead of duplicating this literal "configmedium Write" 3 times.
- `go/devenv/config_test.go:597` — Define a constant instead of duplicating this literal "configmedium EnsureDir" 3 times.
- `go/devenv/devops.go:41` — Define a constant instead of duplicating this literal "devops.New" 3 times.
- `go/devenv/devops.go:149` — Define a constant instead of duplicating this literal "core-dev" 4 times.
- `go/devenv/devops.go:158` — Define a constant instead of duplicating this literal "DevOps.Boot" 3 times.
- `go/devenv/devops_test.go:38` — Define a constant instead of duplicating this literal "expected %v to contain %v" 11 times.
- `go/devenv/devops_test.go:47` — Define a constant instead of duplicating this literal "expected true" 11 times.
- `go/devenv/devops_test.go:78` — Define a constant instead of duplicating this literal "want %v, got %v" 23 times.
- `go/devenv/devops_test.go:115` — Define a constant instead of duplicating this literal "core-dev" 10 times.
- `go/devenv/devops_test.go:119` — Define a constant instead of duplicating this literal "expected false" 8 times.
- `go/devenv/devops_test.go:190` — Define a constant instead of duplicating this literal "containers.json" 19 times.
- `go/devenv/devops_test.go:202` — Define a constant instead of duplicating this literal "test-id" 10 times.
- `go/devenv/devops_test.go:220` — Define a constant instead of duplicating this literal "expected non-nil value" 6 times.
- `go/devenv/devops_test.go:482` — Define a constant instead of duplicating this literal "my-container" 3 times.
- `go/devenv/devops_test.go:568` — Define a constant instead of duplicating this literal "expected error" 6 times.
- `go/devenv/devops_test.go:610` — Define a constant instead of duplicating this literal "v1.2.3" 4 times.
- `go/devenv/devops_test.go:1255` — Define a constant instead of duplicating this literal "expected symbol linked" 32 times.
- `go/devenv/devops_test.go:1258` — Define a constant instead of duplicating this literal "expected callable symbol" 32 times.
- `go/devenv/devops_test.go:1391` — Define a constant instead of duplicating this literal "DevOps IsInstalled" 9 times.
- `go/devenv/devops_test.go:1439` — Define a constant instead of duplicating this literal "DevOps Install" 9 times.
- `go/devenv/devops_test.go:1487` — Define a constant instead of duplicating this literal "DevOps CheckUpdate" 9 times.
- `go/devenv/devops_test.go:1567` — Define a constant instead of duplicating this literal "DevOps Boot" 9 times.
- `go/devenv/devops_test.go:1615` — Define a constant instead of duplicating this literal "DevOps Stop" 9 times.
- `go/devenv/devops_test.go:1663` — Define a constant instead of duplicating this literal "DevOps IsRunning" 9 times.
- `go/devenv/devops_test.go:1711` — Define a constant instead of duplicating this literal "DevOps Status" 9 times.
- `go/devenv/images_test.go:32` — Define a constant instead of duplicating this literal "expected false" 4 times.
- `go/devenv/images_test.go:44` — Define a constant instead of duplicating this literal "expected true" 8 times.
- `go/devenv/images_test.go:66` — Define a constant instead of duplicating this literal "expected non-nil value" 8 times.
- `go/devenv/images_test.go:69` — Define a constant instead of duplicating this literal "want len %v, got %v" 5 times.
- `go/devenv/images_test.go:72` — Define a constant instead of duplicating this literal "want %v, got %v" 22 times.
- `go/devenv/images_test.go:106` — Define a constant instead of duplicating this literal "manifest.json" 17 times.
- `go/devenv/images_test.go:114` — Define a constant instead of duplicating this literal "test.img" 9 times.
- `go/devenv/images_test.go:155` — Define a constant instead of duplicating this literal "expected error" 9 times.
- `go/devenv/images_test.go:181` — Define a constant instead of duplicating this literal "expected %v to contain %v" 6 times.
- `go/devenv/images_test.go:415` — Define a constant instead of duplicating this literal "no image source available" 4 times.
- `go/devenv/images_test.go:486` — Define a constant instead of duplicating this literal "v1.0.0" 14 times.
- `go/devenv/images_test.go:530` — Define a constant instead of duplicating this literal "test error" 3 times.
- `go/devenv/images_test.go:593` — Define a constant instead of duplicating this literal "v2.0.0" 5 times.
- `go/devenv/images_test.go:902` — Define a constant instead of duplicating this literal "expected symbol linked" 14 times.
- `go/devenv/images_test.go:905` — Define a constant instead of duplicating this literal "expected callable symbol" 14 times.
- `go/devenv/images_test.go:926` — Define a constant instead of duplicating this literal "ImageManager IsInstalled" 3 times.
- `go/devenv/images_test.go:974` — Define a constant instead of duplicating this literal "ImageManager Install" 3 times.
- `go/devenv/images_test.go:1022` — Define a constant instead of duplicating this literal "ImageManager CheckUpdate" 3 times.
- `go/devenv/images_test.go:1070` — Define a constant instead of duplicating this literal "Manifest Save" 3 times.
- `go/devenv/serve_test.go:24` — Define a constant instead of duplicating this literal "want %v, got %v" 13 times.
- `go/devenv/serve_test.go:264` — Define a constant instead of duplicating this literal "DevOps Serve" 3 times.
- `go/devenv/serve_test.go:272` — Define a constant instead of duplicating this literal "expected symbol linked" 6 times.
- `go/devenv/serve_test.go:275` — Define a constant instead of duplicating this literal "expected callable symbol" 6 times.
- `go/devenv/shell_test.go:16` — Define a constant instead of duplicating this literal "expected false" 3 times.
- `go/devenv/shell_test.go:98` — Define a constant instead of duplicating this literal "DevOps Shell" 3 times.
- `go/devenv/shell_test.go:106` — Define a constant instead of duplicating this literal "expected symbol linked" 3 times.
- `go/devenv/shell_test.go:109` — Define a constant instead of duplicating this literal "expected callable symbol" 3 times.
- `go/devenv/test.go:43` — Define a constant instead of duplicating this literal "DevOps.Test" 3 times.
- `go/devenv/test_test.go:17` — Define a constant instead of duplicating this literal "composer.json" 6 times.
- `go/devenv/test_test.go:32` — Define a constant instead of duplicating this literal "package.json" 6 times.
- `go/devenv/test_test.go:64` — Define a constant instead of duplicating this literal "test.yaml" 5 times.
- `go/devenv/test_test.go:112` — Define a constant instead of duplicating this literal "expected empty string, got %q" 3 times.
- `go/devenv/test_test.go:503` — Define a constant instead of duplicating this literal "DevOps Test" 3 times.
- `go/devenv/test_test.go:511` — Define a constant instead of duplicating this literal "expected symbol linked" 8 times.
- `go/devenv/test_test.go:514` — Define a constant instead of duplicating this literal "expected callable symbol" 8 times.
- `go/hypervisor.go:101` — Define a constant instead of duplicating this literal "-drive" 3 times.
- `go/hypervisor_test.go:26` — Define a constant instead of duplicating this literal "want type %v, got %v" 3 times.
- `go/hypervisor_test.go:42` — Define a constant instead of duplicating this literal "expected false" 5 times.
- `go/hypervisor_test.go:142` — Define a constant instead of duplicating this literal "expected %v to contain %v" 20 times.
- `go/hypervisor_test.go:146` — Define a constant instead of duplicating this literal "expected non-nil value" 11 times.
- `go/hypervisor_test.go:164` — Define a constant instead of duplicating this literal "not available" 4 times.
- `go/hypervisor_test.go:172` — Define a constant instead of duplicating this literal "want %v, got %v" 4 times.
- `go/hypervisor_test.go:211` — Define a constant instead of duplicating this literal "expected error" 4 times.
- `go/hypervisor_test.go:272` — Define a constant instead of duplicating this literal "/path/to/image.iso" 5 times.
- `go/hypervisor_test.go:322` — Define a constant instead of duplicating this literal "expected true" 3 times.
- `go/hypervisor_test.go:616` — Define a constant instead of duplicating this literal "expected symbol linked" 32 times.
- `go/hypervisor_test.go:619` — Define a constant instead of duplicating this literal "expected callable symbol" 32 times.
- `go/hypervisor_test.go:656` — Define a constant instead of duplicating this literal "QemuHypervisor Name" 3 times.
- `go/hypervisor_test.go:704` — Define a constant instead of duplicating this literal "QemuHypervisor Available" 3 times.
- `go/hypervisor_test.go:752` — Define a constant instead of duplicating this literal "QemuHypervisor BuildCommand" 3 times.
- `go/hypervisor_test.go:848` — Define a constant instead of duplicating this literal "HyperkitHypervisor Name" 3 times.
- `go/hypervisor_test.go:896` — Define a constant instead of duplicating this literal "HyperkitHypervisor Available" 3 times.
- `go/hypervisor_test.go:944` — Define a constant instead of duplicating this literal "HyperkitHypervisor BuildCommand" 3 times.
- `go/internal/proc/proc.go:184` — Define a constant instead of duplicating this literal "proc.LookPath" 3 times.
- `go/internal/proc/proc.go:217` — Define a constant instead of duplicating this literal "command already started" 3 times.
- `go/internal/proc/proc.go:338` — Define a constant instead of duplicating this literal "proc.Command.Wait" 3 times.
- `go/internal/proc/proc_test.go:6` — Define a constant instead of duplicating this literal "Process Kill" 6 times.
- `go/internal/proc/proc_test.go:45` — Define a constant instead of duplicating this literal "Process Signal" 6 times.
- `go/internal/proc/proc_test.go:240` — Define a constant instead of duplicating this literal "Command StdoutPipe" 6 times.
- `go/internal/proc/proc_test.go:279` — Define a constant instead of duplicating this literal "Command StderrPipe" 6 times.
- `go/internal/proc/proc_test.go:318` — Define a constant instead of duplicating this literal "Command Start" 6 times.
- `go/internal/proc/proc_test.go:357` — Define a constant instead of duplicating this literal "Command Run" 6 times.
- `go/internal/proc/proc_test.go:396` — Define a constant instead of duplicating this literal "Command Output" 6 times.
- `go/internal/proc/proc_test.go:435` — Define a constant instead of duplicating this literal "Command Wait" 6 times.
- `go/linuxkit.go:75` — Define a constant instead of duplicating this literal "LinuxKitManager.Run" 22 times.
- `go/linuxkit.go:163` — Define a constant instead of duplicating this literal "failed to close log file" 8 times.
- `go/linuxkit.go:252` — Define a constant instead of duplicating this literal "update container state" 3 times.
- `go/linuxkit.go:284` — Define a constant instead of duplicating this literal "container not found: " 3 times.
- `go/linuxkit.go:284` — Define a constant instead of duplicating this literal "LinuxKitManager.Stop" 3 times.
- `go/linuxkit.go:376` — Define a constant instead of duplicating this literal "LinuxKitManager.Logs" 3 times.
- `go/linuxkit_test.go:70` — Define a constant instead of duplicating this literal "containers.json" 5 times.
- `go/linuxkit_test.go:99` — Define a constant instead of duplicating this literal "want %v, got %v" 30 times.
- `go/linuxkit_test.go:115` — Define a constant instead of duplicating this literal "test.iso" 8 times.
- `go/linuxkit_test.go:116` — Define a constant instead of duplicating this literal "fake image" 9 times.
- `go/linuxkit_test.go:137` — Define a constant instead of duplicating this literal "expected non-empty value" 3 times.
- `go/linuxkit_test.go:227` — Define a constant instead of duplicating this literal "expected error" 15 times.
- `go/linuxkit_test.go:230` — Define a constant instead of duplicating this literal "expected %v to contain %v" 17 times.
- `go/linuxkit_test.go:288` — Define a constant instead of duplicating this literal "expected true" 3 times.
- `go/linuxkit_test.go:308` — Define a constant instead of duplicating this literal "container not found" 3 times.
- `go/linuxkit_test.go:722` — Define a constant instead of duplicating this literal "test.log" 4 times.
- `go/linuxkit_test.go:1219` — Define a constant instead of duplicating this literal "expected symbol linked" 32 times.
- `go/linuxkit_test.go:1222` — Define a constant instead of duplicating this literal "expected callable symbol" 32 times.
- `go/linuxkit_test.go:1291` — Define a constant instead of duplicating this literal "LinuxKitManager Run" 9 times.
- `go/linuxkit_test.go:1339` — Define a constant instead of duplicating this literal "LinuxKitManager Stop" 9 times.
- `go/linuxkit_test.go:1387` — Define a constant instead of duplicating this literal "LinuxKitManager List" 9 times.
- `go/linuxkit_test.go:1435` — Define a constant instead of duplicating this literal "LinuxKitManager Logs" 9 times.
- `go/linuxkit_test.go:1483` — Define a constant instead of duplicating this literal "followreader Read" 3 times.
- `go/linuxkit_test.go:1531` — Define a constant instead of duplicating this literal "followreader Close" 3 times.
- `go/linuxkit_test.go:1579` — Define a constant instead of duplicating this literal "LinuxKitManager Exec" 9 times.
- `go/linuxkit_test.go:1627` — Define a constant instead of duplicating this literal "LinuxKitManager State" 9 times.
- `go/linuxkit_test.go:1675` — Define a constant instead of duplicating this literal "LinuxKitManager Hypervisor" 9 times.
- `go/provider_test.go:23` — Define a constant instead of duplicating this literal "want %v, got %v" 9 times.
- `go/provider_test.go:126` — Define a constant instead of duplicating this literal "expected symbol linked" 20 times.
- `go/provider_test.go:129` — Define a constant instead of duplicating this literal "expected callable symbol" 20 times.
- `go/runtime.go:164` — Define a constant instead of duplicating this literal "--version" 3 times.
- `go/runtime_test.go:52` — Define a constant instead of duplicating this literal "expected true" 6 times.
- `go/runtime_test.go:82` — Define a constant instead of duplicating this literal "expected false" 7 times.
- `go/runtime_test.go:114` — Define a constant instead of duplicating this literal "expected error" 3 times.
- `go/runtime_test.go:163` — Define a constant instead of duplicating this literal "ContainerRuntime HasGPU" 3 times.
- `go/runtime_test.go:171` — Define a constant instead of duplicating this literal "expected symbol linked" 34 times.
- `go/runtime_test.go:174` — Define a constant instead of duplicating this literal "expected callable symbol" 34 times.
- `go/runtime_test.go:211` — Define a constant instead of duplicating this literal "ContainerRuntime HasNetworkIsolation" 3 times.
- `go/runtime_test.go:259` — Define a constant instead of duplicating this literal "ContainerRuntime HasVolumeMounts" 3 times.
- `go/runtime_test.go:307` — Define a constant instead of duplicating this literal "ContainerRuntime HasEncryption" 3 times.
- `go/runtime_test.go:355` — Define a constant instead of duplicating this literal "ContainerRuntime IsHardwareIsolated" 3 times.
- `go/runtime_test.go:403` — Define a constant instead of duplicating this literal "ContainerRuntime HasSubSecondStart" 3 times.
- `go/runtime_test.go:451` — Define a constant instead of duplicating this literal "ContainerRuntime Caps" 3 times.
- `go/runtime_test.go:659` — Define a constant instead of duplicating this literal "runtimeerror Error" 3 times.
- `go/sources/cdn.go:72` — Define a constant instead of duplicating this literal "cdn.Download" 7 times.
- `go/sources/cdn_test.go:25` — Define a constant instead of duplicating this literal "core-devops-darwin-arm64.qcow2" 3 times.
- `go/sources/cdn_test.go:28` — Define a constant instead of duplicating this literal "want %v, got %v" 15 times.
- `go/sources/cdn_test.go:31` — Define a constant instead of duplicating this literal "expected true" 3 times.
- `go/sources/cdn_test.go:67` — Define a constant instead of duplicating this literal "test.img" 10 times.
- `go/sources/cdn_test.go:146` — Define a constant instead of duplicating this literal "expected error" 3 times.
- `go/sources/cdn_test.go:346` — Define a constant instead of duplicating this literal "https://cdn.example.com" 6 times.
- `go/sources/cdn_test.go:346` — Define a constant instead of duplicating this literal "image.qcow2" 3 times.
- `go/sources/cdn_test.go:470` — Define a constant instead of duplicating this literal "expected symbol linked" 14 times.
- `go/sources/cdn_test.go:473` — Define a constant instead of duplicating this literal "expected callable symbol" 14 times.
- `go/sources/cdn_test.go:494` — Define a constant instead of duplicating this literal "CDNSource Name" 9 times.
- `go/sources/cdn_test.go:542` — Define a constant instead of duplicating this literal "CDNSource Available" 9 times.
- `go/sources/cdn_test.go:590` — Define a constant instead of duplicating this literal "CDNSource LatestVersion" 9 times.
- `go/sources/cdn_test.go:638` — Define a constant instead of duplicating this literal "CDNSource Download" 9 times.
- `go/sources/github_test.go:35` — Define a constant instead of duplicating this literal "want %v, got %v" 9 times.
- `go/sources/github_test.go:46` — Define a constant instead of duplicating this literal "owner/repo" 3 times.
- `go/sources/github_test.go:133` — Define a constant instead of duplicating this literal "expected symbol linked" 14 times.
- `go/sources/github_test.go:136` — Define a constant instead of duplicating this literal "expected callable symbol" 14 times.
- `go/sources/github_test.go:157` — Define a constant instead of duplicating this literal "GitHubSource Name" 9 times.
- `go/sources/github_test.go:205` — Define a constant instead of duplicating this literal "GitHubSource Available" 9 times.
- `go/sources/github_test.go:253` — Define a constant instead of duplicating this literal "GitHubSource LatestVersion" 9 times.
- `go/sources/github_test.go:301` — Define a constant instead of duplicating this literal "GitHubSource Download" 9 times.
- `go/sources/source_test.go:16` — Define a constant instead of duplicating this literal "expected empty value" 4 times.
- `go/sources/source_test.go:36` — Define a constant instead of duplicating this literal "owner/repo" 3 times.
- `go/sources/source_test.go:42` — Define a constant instead of duplicating this literal "want %v, got %v" 4 times.
- `go/state_test.go:18` — Define a constant instead of duplicating this literal "/tmp/test-state.json" 3 times.
- `go/state_test.go:20` — Define a constant instead of duplicating this literal "expected non-nil value" 3 times.
- `go/state_test.go:26` — Define a constant instead of duplicating this literal "want %v, got %v" 5 times.
- `go/state_test.go:38` — Define a constant instead of duplicating this literal "containers.json" 9 times.
- `go/state_test.go:84` — Define a constant instead of duplicating this literal "want len %v, got %v" 4 times.
- `go/state_test.go:89` — Define a constant instead of duplicating this literal "expected true" 6 times.
- `go/state_test.go:287` — Define a constant instead of duplicating this literal "expected %v to contain %v" 4 times.
- `go/state_test.go:394` — Define a constant instead of duplicating this literal "expected symbol linked" 36 times.
- `go/state_test.go:397` — Define a constant instead of duplicating this literal "expected callable symbol" 36 times.
- `go/state_test.go:562` — Define a constant instead of duplicating this literal "State SaveState" 3 times.
- `go/state_test.go:610` — Define a constant instead of duplicating this literal "State Add" 3 times.
- `go/state_test.go:658` — Define a constant instead of duplicating this literal "State Get" 3 times.
- `go/state_test.go:706` — Define a constant instead of duplicating this literal "State Update" 3 times.
- `go/state_test.go:754` — Define a constant instead of duplicating this literal "State Remove" 3 times.
- `go/state_test.go:802` — Define a constant instead of duplicating this literal "State All" 3 times.
- `go/state_test.go:850` — Define a constant instead of duplicating this literal "State FilePath" 3 times.
- `go/templates_test.go:29` — Define a constant instead of duplicating this literal "core-dev" 4 times.
- `go/templates_test.go:32` — Define a constant instead of duplicating this literal "expected non-empty value" 7 times.
- `go/templates_test.go:41` — Define a constant instead of duplicating this literal "expected true" 7 times.
- `go/templates_test.go:47` — Define a constant instead of duplicating this literal "server-php" 3 times.
- `go/templates_test.go:77` — Define a constant instead of duplicating this literal "expected %v to contain %v" 21 times.
- `go/templates_test.go:125` — Define a constant instead of duplicating this literal "expected error" 5 times.
- `go/templates_test.go:149` — Define a constant instead of duplicating this literal "want %v, got %v" 18 times.
- `go/templates_test.go:254` — Define a constant instead of duplicating this literal "missing required variables" 3 times.
- `go/templates_test.go:379` — Define a constant instead of duplicating this literal "want len %v, got %v" 9 times.
- `go/templates_test.go:405` — Define a constant instead of duplicating this literal "expected empty value" 7 times.
- `go/templates_test.go:544` — Define a constant instead of duplicating this literal "test.yml" 4 times.
- `go/templates_test.go:840` — Define a constant instead of duplicating this literal "expected symbol linked" 15 times.
- `go/templates_test.go:843` — Define a constant instead of duplicating this literal "expected callable symbol" 15 times.
- `go/tim.go:165` — Define a constant instead of duplicating this literal "bundle is required" 3 times.
- `go/tim.go:202` — Define a constant instead of duplicating this literal "workspace key is required" 4 times.
- `go/tim_test.go:17` — Define a constant instead of duplicating this literal "/var/tim/worker-01" 3 times.
- `go/tim_test.go:17` — Define a constant instead of duplicating this literal "worker-01" 14 times.
- `go/tim_test.go:19` — Define a constant instead of duplicating this literal "want %v, got %v" 9 times.
- `go/tim_test.go:59` — Define a constant instead of duplicating this literal "expected true" 4 times.
- `go/tim_test.go:71` — Define a constant instead of duplicating this literal "expected error" 7 times.
- `go/tim_test.go:98` — Define a constant instead of duplicating this literal "workspace-key-32-bytes-xxxxxxxxxx" 3 times.
- `go/tim_test.go:313` — Define a constant instead of duplicating this literal "expected symbol linked" 26 times.
- `go/tim_test.go:316` — Define a constant instead of duplicating this literal "expected callable symbol" 26 times.

### go:S3776 — Cognitive Complexity of functions should not be too high (13×, code smell)

- `go/apple.go:204` — Refactor this method to reduce its Cognitive Complexity from 18 to the 15 allowed.
- `go/cmd/vm/cmd_container.go:225` — Refactor this method to reduce its Cognitive Complexity from 17 to the 15 allowed.
- `go/devenv/claude.go:22` — Refactor this method to reduce its Cognitive Complexity from 27 to the 15 allowed.
- `go/devenv/config_test.go:45` — Refactor this method to reduce its Cognitive Complexity from 17 to the 15 allowed.
- `go/devenv/config_test.go:235` — Refactor this method to reduce its Cognitive Complexity from 23 to the 15 allowed.
- `go/devenv/images_test.go:48` — Refactor this method to reduce its Cognitive Complexity from 17 to the 15 allowed.
- `go/devenv/ssh_utils.go:16` — Refactor this method to reduce its Cognitive Complexity from 18 to the 15 allowed.
- `go/devenv/test.go:35` — Refactor this method to reduce its Cognitive Complexity from 16 to the 15 allowed.
- `go/hypervisor_test.go:200` — Refactor this method to reduce its Cognitive Complexity from 19 to the 15 allowed.
- `go/linuxkit.go:69` — Refactor this method to reduce its Cognitive Complexity from 45 to the 15 allowed.
- `go/sources/cdn.go:65` — Refactor this method to reduce its Cognitive Complexity from 18 to the 15 allowed.
- `go/templates.go:262` — Refactor this method to reduce its Cognitive Complexity from 16 to the 15 allowed.
- `go/templates_test.go:12` — Refactor this method to reduce its Cognitive Complexity from 22 to the 15 allowed.

## MAJOR

### yaml:DocumentStartCheck — For correct parsing especially in the case of multiple or embedded documents, documents should start with a document start marker (3×, code smell)

- `go/templates/core-dev.yml:11` — missing document start "---" (document-start)
- `go/templates/server-php.yml:13` — missing document start "---" (document-start)
- `tests/cli/container/Taskfile.yaml:1` — missing document start "---" (document-start)

### go:S107 — Functions should not have too many parameters (1×, code smell)

- `go/cmd/vm/cmd_container.go:116` — This function has 8 parameters, which is greater than the 7 authorized.

### go:S4144 — Functions should not have identical implementations (1×, code smell)

- `go/state.go:157` — Update this function so that its implementation is not identical to "Add" on line 131.

## MINOR

### go:S1940 — Boolean checks should not be inverted (2×, code smell)

- `go/devenv/devops_test.go:46` — Use the opposite operator ("!=") instead.
- `go/devenv/devops_test.go:1117` — Use the opposite operator ("!=") instead.

## INFO

### yaml:LineLengthCheck — For readability and maintenance lines should not exceed a certain length (2×, code smell)

- `go/templates/core-dev.yml:87` — line too long (81 > 80 characters) (line-length)
- `tests/cli/container/Taskfile.yaml:27` — line too long (102 > 80 characters) (line-length)

### go:S1135 — Track uses of "TODO" tags (1×, code smell)

- `go/datacube.go:149` — Complete the task associated to this TODO comment.

