# Upgrade Report: dappco.re/go/core v0.8.0-alpha.1

## Scope

- Repository: `core/go-container`
- Branch: `agent/create-an-upgrade-plan-for-this-package`
- Requested target: `dappco.re/go/core v0.8.0-alpha.1`
- Consumers called out for break-risk review: `core`, `go-devops`

## Baseline Verification

- `go build ./...`: passed
- `go vet ./...`: passed
- `go test ./... -count=1 -timeout 120s`: passed
- `go test -cover ./...`: passed (`container` 81.7%, `cmd/vm` 0.0%, `devenv` 53.3%, `sources` 72.7%)
- `go mod tidy`: not run because this task is report-only and should not introduce dependency churn

## 1. go.mod Upgrade Plan

- Current core version: `dappco.re/go/core v0.5.0` at `go.mod:16`
- Required bump: `dappco.re/go/core v0.5.0` -> `dappco.re/go/core v0.8.0-alpha.1` at `go.mod:16`
- Direct `dappco.re/go/core/*` dependencies that should be compatibility-checked in the same upgrade pass:
  - `go.mod:6` `dappco.re/go/core/i18n v0.2.0`
  - `go.mod:7` `dappco.re/go/core/io v0.2.0`
  - `go.mod:8` `dappco.re/go/core/log v0.1.0`
- Legacy `forge.lthn.ai` modules still present in `go.mod`; these should be reviewed during the core bump because they may pin older transitive core APIs:
  - `go.mod:9` `forge.lthn.ai/core/cli v0.3.7`
  - `go.mod:10` `forge.lthn.ai/core/config v0.1.8`
  - `go.mod:17` `forge.lthn.ai/core/go v0.3.3`
  - `go.mod:18` `forge.lthn.ai/core/go-i18n v0.1.7`
  - `go.mod:19` `forge.lthn.ai/core/go-inference v0.1.6`
  - `go.mod:20` `forge.lthn.ai/core/go-io v0.1.7`
  - `go.mod:21` `forge.lthn.ai/core/go-log v0.0.4`

## 2. Banned Stdlib Imports

Each group lists every import site and the required Core replacement.

### `os`

- Replacement: Replace with core.Env/core.Fs
- `cmd/vm/cmd_container.go:7`
- `cmd/vm/cmd_templates.go:6`
- `devenv/claude.go:6`
- `devenv/config.go:4`
- `devenv/config_test.go:4`
- `devenv/devops.go:7`
- `devenv/devops_test.go:5`
- `devenv/images.go:7`
- `devenv/images_test.go:5`
- `devenv/serve.go:6`
- `devenv/serve_test.go:4`
- `devenv/shell.go:6`
- `devenv/ssh_utils.go:6`
- `devenv/test_test.go:4`
- `hypervisor.go:6`
- `linuxkit.go:8`
- `linuxkit_test.go:5`
- `sources/cdn.go:8`
- `sources/cdn_test.go:8`
- `sources/github.go:5`
- `state.go:5`
- `state_test.go:4`
- `templates.go:7`
- `templates_test.go:4`

### `os/exec`

- Replacement: No direct replacement was provided in the task; requires a manual audit for the v0.8.0-alpha.1 command-exec path
- `cmd/vm/cmd_templates.go:7`
- `devenv/claude.go:7`
- `devenv/devops_test.go:6`
- `devenv/serve.go:7`
- `devenv/shell.go:7`
- `devenv/ssh_utils.go:7`
- `hypervisor.go:7`
- `linuxkit.go:9`
- `linuxkit_test.go:6`
- `sources/github.go:6`

### `encoding/json`

- Replacement: Replace with core.JSONMarshalString/JSONUnmarshalString
- `devenv/images.go:5`
- `devenv/test.go:5`
- `state.go:4`

### `fmt`

- Replacement: Replace with core.Sprintf/core.Concat/core.E
- `cmd/vm/cmd_container.go:5`
- `cmd/vm/cmd_templates.go:5`
- `devenv/claude.go:5`
- `devenv/devops.go:6`
- `devenv/images.go:6`
- `devenv/serve.go:5`
- `devenv/shell.go:5`
- `devenv/ssh_utils.go:5`
- `hypervisor.go:5`
- `linuxkit.go:6`
- `sources/cdn.go:5`
- `sources/cdn_test.go:5`

### `errors`

- Replacement: Replace with core.E/core.Is
- No occurrences found

### `strings`

- Replacement: Replace with core.Contains/core.HasPrefix/core.Split/core.Trim/core.Replace
- `cmd/vm/cmd_container.go:8`
- `cmd/vm/cmd_templates.go:9`
- `devenv/claude.go:9`
- `devenv/ssh_utils.go:9`
- `devenv/test.go:7`
- `hypervisor.go:10`
- `sources/github.go:7`
- `templates.go:11`
- `templates_test.go:6`

### `path/filepath`

- Replacement: Replace with core.JoinPath/core.PathBase/core.PathDir
- `cmd/vm/cmd_templates.go:8`
- `devenv/claude.go:8`
- `devenv/config.go:5`
- `devenv/config_test.go:5`
- `devenv/devops.go:8`
- `devenv/devops_test.go:7`
- `devenv/images.go:8`
- `devenv/images_test.go:6`
- `devenv/serve.go:8`
- `devenv/serve_test.go:5`
- `devenv/ssh_utils.go:8`
- `devenv/test.go:6`
- `devenv/test_test.go:5`
- `hypervisor.go:8`
- `linuxkit_test.go:7`
- `sources/cdn.go:9`
- `sources/cdn_test.go:9`
- `state.go:6`
- `state_test.go:5`
- `templates.go:8`
- `templates_test.go:5`

## 3. Tests Not Matching `TestFile_Function_{Good,Bad,Ugly}`

- Total mismatches found: 236

- `devenv/claude_test.go:9` `TestClaudeOptions_Default`
- `devenv/claude_test.go:16` `TestClaudeOptions_Custom`
- `devenv/claude_test.go:27` `TestFormatAuthList_Good_NoAuth`
- `devenv/claude_test.go:33` `TestFormatAuthList_Good_Default`
- `devenv/claude_test.go:39` `TestFormatAuthList_Good_CustomAuth`
- `devenv/claude_test.go:47` `TestFormatAuthList_Good_MultipleAuth`
- `devenv/claude_test.go:55` `TestFormatAuthList_Good_EmptyAuth`
- `devenv/config_test.go:13` `TestDefaultConfig`
- `devenv/config_test.go:20` `TestConfigPath`
- `devenv/config_test.go:26` `TestLoadConfig_Good`
- `devenv/config_test.go:65` `TestLoadConfig_Bad`
- `devenv/config_test.go:82` `TestConfig_Struct`
- `devenv/config_test.go:105` `TestDefaultConfig_Complete`
- `devenv/config_test.go:114` `TestLoadConfig_Good_PartialConfig`
- `devenv/config_test.go:139` `TestLoadConfig_Good_AllSourceTypes`
- `devenv/config_test.go:208` `TestImagesConfig_Struct`
- `devenv/config_test.go:217` `TestGitHubConfig_Struct`
- `devenv/config_test.go:222` `TestRegistryConfig_Struct`
- `devenv/config_test.go:227` `TestCDNConfig_Struct`
- `devenv/config_test.go:232` `TestLoadConfig_Bad_UnreadableFile`
- `devenv/devops_test.go:18` `TestImageName`
- `devenv/devops_test.go:26` `TestImagesDir`
- `devenv/devops_test.go:48` `TestImagePath`
- `devenv/devops_test.go:58` `TestDefaultBootOptions`
- `devenv/devops_test.go:66` `TestIsInstalled_Bad`
- `devenv/devops_test.go:78` `TestIsInstalled_Good`
- `devenv/devops_test.go:142` `TestDevOps_Status_Good_NotInstalled`
- `devenv/devops_test.go:168` `TestDevOps_Status_Good_NoContainer`
- `devenv/devops_test.go:232` `TestDevOps_IsRunning_Bad_NotRunning`
- `devenv/devops_test.go:255` `TestDevOps_IsRunning_Bad_ContainerStopped`
- `devenv/devops_test.go:323` `TestDevOps_findContainer_Bad_NotFound`
- `devenv/devops_test.go:346` `TestDevOps_Stop_Bad_NotFound`
- `devenv/devops_test.go:369` `TestBootOptions_Custom`
- `devenv/devops_test.go:382` `TestDevStatus_Struct`
- `devenv/devops_test.go:403` `TestDevOps_Boot_Bad_NotInstalled`
- `devenv/devops_test.go:426` `TestDevOps_Boot_Bad_AlreadyRunning`
- `devenv/devops_test.go:465` `TestDevOps_Status_Good_WithImageVersion`
- `devenv/devops_test.go:501` `TestDevOps_findContainer_Good_MultipleContainers`
- `devenv/devops_test.go:546` `TestDevOps_Status_Good_ContainerWithUptime`
- `devenv/devops_test.go:583` `TestDevOps_IsRunning_Bad_DifferentContainerName`
- `devenv/devops_test.go:618` `TestDevOps_Boot_Good_FreshFlag`
- `devenv/devops_test.go:668` `TestDevOps_Stop_Bad_ContainerNotRunning`
- `devenv/devops_test.go:703` `TestDevOps_Boot_Good_FreshWithNoExisting`
- `devenv/devops_test.go:741` `TestImageName_Format`
- `devenv/devops_test.go:750` `TestDevOps_Install_Delegates`
- `devenv/devops_test.go:768` `TestDevOps_CheckUpdate_Delegates`
- `devenv/devops_test.go:786` `TestDevOps_Boot_Good_Success`
- `devenv/devops_test.go:818` `TestDevOps_Config`
- `devenv/images_test.go:16` `TestImageManager_Good_IsInstalled`
- `devenv/images_test.go:36` `TestNewImageManager_Good`
- `devenv/images_test.go:66` `TestManifest_Save`
- `devenv/images_test.go:94` `TestLoadManifest_Bad`
- `devenv/images_test.go:106` `TestCheckUpdate_Bad`
- `devenv/images_test.go:121` `TestNewImageManager_Good_AutoSource`
- `devenv/images_test.go:134` `TestNewImageManager_Good_UnknownSourceFallsToAuto`
- `devenv/images_test.go:147` `TestLoadManifest_Good_Empty`
- `devenv/images_test.go:159` `TestLoadManifest_Good_ExistingData`
- `devenv/images_test.go:174` `TestImageInfo_Struct`
- `devenv/images_test.go:187` `TestManifest_Save_Good_CreatesDirs`
- `devenv/images_test.go:207` `TestManifest_Save_Good_Overwrite`
- `devenv/images_test.go:239` `TestImageManager_Install_Bad_NoSourceAvailable`
- `devenv/images_test.go:256` `TestNewImageManager_Good_CreatesDir`
- `devenv/images_test.go:295` `TestImageManager_Install_Good_WithMockSource`
- `devenv/images_test.go:323` `TestImageManager_Install_Bad_DownloadError`
- `devenv/images_test.go:345` `TestImageManager_Install_Bad_VersionError`
- `devenv/images_test.go:367` `TestImageManager_Install_Good_SkipsUnavailableSource`
- `devenv/images_test.go:396` `TestImageManager_CheckUpdate_Good_WithMockSource`
- `devenv/images_test.go:426` `TestImageManager_CheckUpdate_Good_NoUpdate`
- `devenv/images_test.go:456` `TestImageManager_CheckUpdate_Bad_NoSource`
- `devenv/images_test.go:483` `TestImageManager_CheckUpdate_Bad_VersionError`
- `devenv/images_test.go:511` `TestImageManager_Install_Bad_EmptySources`
- `devenv/images_test.go:527` `TestImageManager_Install_Bad_AllUnavailable`
- `devenv/images_test.go:546` `TestImageManager_CheckUpdate_Good_FirstSourceUnavailable`
- `devenv/images_test.go:573` `TestManifest_Struct`
- `devenv/serve_test.go:12` `TestDetectServeCommand_Good_Laravel`
- `devenv/serve_test.go:21` `TestDetectServeCommand_Good_NodeDev`
- `devenv/serve_test.go:31` `TestDetectServeCommand_Good_NodeStart`
- `devenv/serve_test.go:41` `TestDetectServeCommand_Good_PHP`
- `devenv/serve_test.go:50` `TestDetectServeCommand_Good_GoMain`
- `devenv/serve_test.go:61` `TestDetectServeCommand_Good_GoWithoutMain`
- `devenv/serve_test.go:71` `TestDetectServeCommand_Good_Django`
- `devenv/serve_test.go:80` `TestDetectServeCommand_Good_Fallback`
- `devenv/serve_test.go:87` `TestDetectServeCommand_Good_Priority`
- `devenv/serve_test.go:99` `TestServeOptions_Default`
- `devenv/serve_test.go:105` `TestServeOptions_Custom`
- `devenv/serve_test.go:114` `TestHasFile_Good`
- `devenv/serve_test.go:123` `TestHasFile_Bad`
- `devenv/serve_test.go:129` `TestHasFile_Bad_Directory`
- `devenv/shell_test.go:9` `TestShellOptions_Default`
- `devenv/shell_test.go:15` `TestShellOptions_Console`
- `devenv/shell_test.go:23` `TestShellOptions_Command`
- `devenv/shell_test.go:31` `TestShellOptions_ConsoleWithCommand`
- `devenv/shell_test.go:40` `TestShellOptions_EmptyCommand`
- `devenv/test_test.go:11` `TestDetectTestCommand_Good_ComposerJSON`
- `devenv/test_test.go:21` `TestDetectTestCommand_Good_PackageJSON`
- `devenv/test_test.go:31` `TestDetectTestCommand_Good_GoMod`
- `devenv/test_test.go:41` `TestDetectTestCommand_Good_CoreTestYaml`
- `devenv/test_test.go:53` `TestDetectTestCommand_Good_Pytest`
- `devenv/test_test.go:63` `TestDetectTestCommand_Good_Taskfile`
- `devenv/test_test.go:73` `TestDetectTestCommand_Bad_NoFiles`
- `devenv/test_test.go:82` `TestDetectTestCommand_Good_Priority`
- `devenv/test_test.go:96` `TestLoadTestConfig_Good`
- `devenv/test_test.go:135` `TestLoadTestConfig_Bad_NotFound`
- `devenv/test_test.go:144` `TestHasPackageScript_Good`
- `devenv/test_test.go:156` `TestHasPackageScript_Bad_MissingScript`
- `devenv/test_test.go:165` `TestHasComposerScript_Good`
- `devenv/test_test.go:174` `TestHasComposerScript_Bad_MissingScript`
- `devenv/test_test.go:183` `TestTestConfig_Struct`
- `devenv/test_test.go:204` `TestTestCommand_Struct`
- `devenv/test_test.go:217` `TestTestOptions_Struct`
- `devenv/test_test.go:230` `TestDetectTestCommand_Good_TaskfileYml`
- `devenv/test_test.go:240` `TestDetectTestCommand_Good_Pyproject`
- `devenv/test_test.go:250` `TestHasPackageScript_Bad_NoFile`
- `devenv/test_test.go:258` `TestHasPackageScript_Bad_InvalidJSON`
- `devenv/test_test.go:267` `TestHasPackageScript_Bad_NoScripts`
- `devenv/test_test.go:276` `TestHasComposerScript_Bad_NoFile`
- `devenv/test_test.go:284` `TestHasComposerScript_Bad_InvalidJSON`
- `devenv/test_test.go:293` `TestHasComposerScript_Bad_NoScripts`
- `devenv/test_test.go:302` `TestLoadTestConfig_Bad_InvalidYAML`
- `devenv/test_test.go:314` `TestLoadTestConfig_Good_MinimalConfig`
- `devenv/test_test.go:332` `TestDetectTestCommand_Good_ComposerWithoutScript`
- `devenv/test_test.go:344` `TestDetectTestCommand_Good_PackageJSONWithoutScript`
- `hypervisor_test.go:23` `TestQemuHypervisor_Available_Bad_InvalidBinary`
- `hypervisor_test.go:47` `TestHyperkitHypervisor_Available_Bad_NotDarwin`
- `hypervisor_test.go:59` `TestHyperkitHypervisor_Available_Bad_InvalidBinary`
- `hypervisor_test.go:69` `TestIsKVMAvailable_Good`
- `hypervisor_test.go:83` `TestDetectHypervisor_Good`
- `hypervisor_test.go:98` `TestGetHypervisor_Good_Qemu`
- `hypervisor_test.go:110` `TestGetHypervisor_Good_QemuUppercase`
- `hypervisor_test.go:122` `TestGetHypervisor_Good_Hyperkit`
- `hypervisor_test.go:140` `TestGetHypervisor_Bad_Unknown`
- `hypervisor_test.go:147` `TestQemuHypervisor_BuildCommand_Good_WithPortsAndVolumes`
- `hypervisor_test.go:175` `TestQemuHypervisor_BuildCommand_Good_QCow2Format`
- `hypervisor_test.go:195` `TestQemuHypervisor_BuildCommand_Good_VMDKFormat`
- `hypervisor_test.go:215` `TestQemuHypervisor_BuildCommand_Good_RawFormat`
- `hypervisor_test.go:235` `TestHyperkitHypervisor_BuildCommand_Good_WithPorts`
- `hypervisor_test.go:258` `TestHyperkitHypervisor_BuildCommand_Good_QCow2Format`
- `hypervisor_test.go:269` `TestHyperkitHypervisor_BuildCommand_Good_RawFormat`
- `hypervisor_test.go:280` `TestHyperkitHypervisor_BuildCommand_Good_NoPorts`
- `hypervisor_test.go:296` `TestQemuHypervisor_BuildCommand_Good_NoSSHPort`
- `hypervisor_test.go:312` `TestQemuHypervisor_BuildCommand_Bad_UnknownFormat`
- `hypervisor_test.go:323` `TestHyperkitHypervisor_BuildCommand_Bad_UnknownFormat`
- `hypervisor_test.go:339` `TestHyperkitHypervisor_BuildCommand_Good_ISOFormat`
- `linuxkit_test.go:76` `TestNewLinuxKitManagerWithHypervisor_Good`
- `linuxkit_test.go:89` `TestLinuxKitManager_Run_Good_Detached`
- `linuxkit_test.go:128` `TestLinuxKitManager_Run_Good_DefaultValues`
- `linuxkit_test.go:153` `TestLinuxKitManager_Run_Bad_ImageNotFound`
- `linuxkit_test.go:164` `TestLinuxKitManager_Run_Bad_UnsupportedFormat`
- `linuxkit_test.go:204` `TestLinuxKitManager_Stop_Bad_NotFound`
- `linuxkit_test.go:214` `TestLinuxKitManager_Stop_Bad_NotRunning`
- `linuxkit_test.go:251` `TestLinuxKitManager_List_Good_VerifiesRunningStatus`
- `linuxkit_test.go:303` `TestLinuxKitManager_Logs_Bad_NotFound`
- `linuxkit_test.go:313` `TestLinuxKitManager_Logs_Bad_NoLogFile`
- `linuxkit_test.go:336` `TestLinuxKitManager_Exec_Bad_NotFound`
- `linuxkit_test.go:346` `TestLinuxKitManager_Exec_Bad_NotRunning`
- `linuxkit_test.go:359` `TestDetectImageFormat_Good`
- `linuxkit_test.go:381` `TestDetectImageFormat_Bad_Unknown`
- `linuxkit_test.go:429` `TestLinuxKitManager_Logs_Good_Follow`
- `linuxkit_test.go:467` `TestFollowReader_Read_Good_WithData`
- `linuxkit_test.go:500` `TestFollowReader_Read_Good_ContextCancel`
- `linuxkit_test.go:544` `TestNewFollowReader_Bad_FileNotFound`
- `linuxkit_test.go:551` `TestLinuxKitManager_Run_Bad_BuildCommandError`
- `linuxkit_test.go:570` `TestLinuxKitManager_Run_Good_Foreground`
- `linuxkit_test.go:598` `TestLinuxKitManager_Stop_Good_ContextCancelled`
- `linuxkit_test.go:635` `TestIsProcessRunning_Good_ExistingProcess`
- `linuxkit_test.go:641` `TestIsProcessRunning_Bad_NonexistentProcess`
- `linuxkit_test.go:647` `TestLinuxKitManager_Run_Good_WithPortsAndVolumes`
- `linuxkit_test.go:676` `TestFollowReader_Read_Bad_ReaderError`
- `linuxkit_test.go:697` `TestLinuxKitManager_Run_Bad_StartError`
- `linuxkit_test.go:718` `TestLinuxKitManager_Run_Bad_ForegroundStartError`
- `linuxkit_test.go:739` `TestLinuxKitManager_Run_Good_ForegroundWithError`
- `linuxkit_test.go:762` `TestLinuxKitManager_Stop_Good_ProcessExitedWhileRunning`
- `sources/cdn_test.go:16` `TestCDNSource_Good_Available`
- `sources/cdn_test.go:26` `TestCDNSource_Bad_NoURL`
- `sources/cdn_test.go:118` `TestCDNSource_LatestVersion_Bad_NoManifest`
- `sources/cdn_test.go:134` `TestCDNSource_LatestVersion_Bad_ServerError`
- `sources/cdn_test.go:150` `TestCDNSource_Download_Good_NoProgress`
- `sources/cdn_test.go:174` `TestCDNSource_Download_Good_LargeFile`
- `sources/cdn_test.go:206` `TestCDNSource_Download_Bad_HTTPErrorCodes`
- `sources/cdn_test.go:238` `TestCDNSource_InterfaceCompliance`
- `sources/cdn_test.go:243` `TestCDNSource_Config`
- `sources/cdn_test.go:254` `TestNewCDNSource_Good`
- `sources/cdn_test.go:268` `TestCDNSource_Download_Good_CreatesDestDir`
- `sources/cdn_test.go:294` `TestSourceConfig_Struct`
- `sources/github_test.go:9` `TestGitHubSource_Good_Available`
- `sources/github_test.go:23` `TestGitHubSource_Name`
- `sources/github_test.go:28` `TestGitHubSource_Config`
- `sources/github_test.go:40` `TestGitHubSource_Good_Multiple`
- `sources/github_test.go:51` `TestNewGitHubSource_Good`
- `sources/github_test.go:65` `TestGitHubSource_InterfaceCompliance`
- `sources/source_test.go:9` `TestSourceConfig_Empty`
- `sources/source_test.go:17` `TestSourceConfig_Complete`
- `sources/source_test.go:31` `TestImageSource_Interface`
- `state_test.go:13` `TestNewState_Good`
- `state_test.go:21` `TestLoadState_Good_NewFile`
- `state_test.go:33` `TestLoadState_Good_ExistingFile`
- `state_test.go:64` `TestLoadState_Bad_InvalidJSON`
- `state_test.go:142` `TestState_Get_Bad_NotFound`
- `state_test.go:162` `TestState_SaveState_Good_CreatesDirectory`
- `state_test.go:177` `TestDefaultStateDir_Good`
- `state_test.go:183` `TestDefaultStatePath_Good`
- `state_test.go:189` `TestDefaultLogsDir_Good`
- `state_test.go:195` `TestLogPath_Good`
- `state_test.go:201` `TestEnsureLogsDir_Good`
- `state_test.go:211` `TestGenerateID_Good`
- `templates_test.go:13` `TestListTemplates_Good`
- `templates_test.go:44` `TestGetTemplate_Good_CoreDev`
- `templates_test.go:55` `TestGetTemplate_Good_ServerPhp`
- `templates_test.go:66` `TestGetTemplate_Bad_NotFound`
- `templates_test.go:73` `TestApplyVariables_Good_SimpleSubstitution`
- `templates_test.go:86` `TestApplyVariables_Good_WithDefaults`
- `templates_test.go:99` `TestApplyVariables_Good_AllDefaults`
- `templates_test.go:109` `TestApplyVariables_Good_MixedSyntax`
- `templates_test.go:128` `TestApplyVariables_Good_EmptyDefault`
- `templates_test.go:138` `TestApplyVariables_Bad_MissingRequired`
- `templates_test.go:149` `TestApplyVariables_Bad_MultipleMissing`
- `templates_test.go:164` `TestApplyTemplate_Good`
- `templates_test.go:178` `TestApplyTemplate_Bad_TemplateNotFound`
- `templates_test.go:189` `TestApplyTemplate_Bad_MissingVariable`
- `templates_test.go:199` `TestExtractVariables_Good`
- `templates_test.go:221` `TestExtractVariables_Good_NoVariables`
- `templates_test.go:230` `TestExtractVariables_Good_OnlyDefaults`
- `templates_test.go:241` `TestScanUserTemplates_Good`
- `templates_test.go:265` `TestScanUserTemplates_Good_MultipleTemplates`
- `templates_test.go:287` `TestScanUserTemplates_Good_EmptyDirectory`
- `templates_test.go:295` `TestScanUserTemplates_Bad_NonexistentDirectory`
- `templates_test.go:301` `TestExtractTemplateDescription_Good`
- `templates_test.go:318` `TestExtractTemplateDescription_Good_NoComments`
- `templates_test.go:333` `TestExtractTemplateDescription_Bad_FileNotFound`
- `templates_test.go:339` `TestVariablePatternEdgeCases_Good`
- `templates_test.go:387` `TestScanUserTemplates_Good_SkipsBuiltinNames`
- `templates_test.go:405` `TestScanUserTemplates_Good_SkipsDirectories`
- `templates_test.go:422` `TestScanUserTemplates_Good_YamlExtension`
- `templates_test.go:443` `TestExtractTemplateDescription_Good_EmptyComment`
- `templates_test.go:461` `TestExtractTemplateDescription_Good_MultipleEmptyComments`
- `templates_test.go:481` `TestScanUserTemplates_Good_DefaultDescription`

## 4. Exported Functions Missing Usage-Example Doc Comments

- Total exported functions missing a usage example marker: 38
- Note: every function listed below has a doc comment, but none of the comments include an obvious usage example marker such as `Usage:` or `Example:`.

- `cmd/vm/cmd_templates.go:150` `RunFromTemplate` (missing usage example)
- `cmd/vm/cmd_templates.go:296` `ParseVarFlags` (missing usage example)
- `cmd/vm/cmd_vm.go:28` `AddVMCommands` (missing usage example)
- `container.go:84` `GenerateID` (missing usage example)
- `devenv/config.go:41` `DefaultConfig` (missing usage example)
- `devenv/config.go:57` `ConfigPath` (missing usage example)
- `devenv/config.go:67` `LoadConfig` (missing usage example)
- `devenv/devops.go:31` `New` (missing usage example)
- `devenv/devops.go:56` `ImageName` (missing usage example)
- `devenv/devops.go:61` `ImagesDir` (missing usage example)
- `devenv/devops.go:73` `ImagePath` (missing usage example)
- `devenv/devops.go:109` `DefaultBootOptions` (missing usage example)
- `devenv/images.go:40` `NewImageManager` (missing usage example)
- `devenv/serve.go:75` `DetectServeCommand` (missing usage example)
- `devenv/test.go:75` `DetectTestCommand` (missing usage example)
- `devenv/test.go:115` `LoadTestConfig` (missing usage example)
- `hypervisor.go:50` `NewQemuHypervisor` (missing usage example)
- `hypervisor.go:155` `NewHyperkitHypervisor` (missing usage example)
- `hypervisor.go:222` `DetectImageFormat` (missing usage example)
- `hypervisor.go:239` `DetectHypervisor` (missing usage example)
- `hypervisor.go:258` `GetHypervisor` (missing usage example)
- `linuxkit.go:25` `NewLinuxKitManager` (missing usage example)
- `linuxkit.go:49` `NewLinuxKitManagerWithHypervisor` (missing usage example)
- `sources/cdn.go:24` `NewCDNSource` (missing usage example)
- `sources/github.go:22` `NewGitHubSource` (missing usage example)
- `state.go:22` `DefaultStateDir` (missing usage example)
- `state.go:31` `DefaultStatePath` (missing usage example)
- `state.go:40` `DefaultLogsDir` (missing usage example)
- `state.go:49` `NewState` (missing usage example)
- `state.go:58` `LoadState` (missing usage example)
- `state.go:157` `LogPath` (missing usage example)
- `state.go:166` `EnsureLogsDir` (missing usage example)
- `templates.go:47` `ListTemplates` (missing usage example)
- `templates.go:52` `ListTemplatesIter` (missing usage example)
- `templates.go:75` `GetTemplate` (missing usage example)
- `templates.go:107` `ApplyTemplate` (missing usage example)
- `templates.go:120` `ApplyVariables` (missing usage example)
- `templates.go:169` `ExtractVariables` (missing usage example)

## Risk Notes

- Breaking-change surface is moderate because this repo is consumed by two modules: `core` and `go-devops`.
- The highest-effort part of the upgrade is not the version bump itself; it is the repo-wide removal of banned stdlib imports, especially the current `os/exec` usage across runtime code and tests.
- The doc-comment and test-renaming work is mechanically simple, but it touches many files and will create broad diff surface for downstream review.
