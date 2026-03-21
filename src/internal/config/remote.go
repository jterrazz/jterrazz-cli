package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// RemoteMode controls how the local tailscale client is run.
type RemoteMode string

const (
	RemoteModeAuto      RemoteMode = "auto"
	RemoteModeUserspace RemoteMode = "userspace"
)

// RemoteAuthMethod controls how `tailscale up` authenticates the node.
type RemoteAuthMethod string

const (
	RemoteAuthOAuth   RemoteAuthMethod = "oauth"
	RemoteAuthAuthKey RemoteAuthMethod = "authkey"
)

const (
	remoteModeSystemLegacy  RemoteMode       = "system"
	remoteAuthNoneLegacy    RemoteAuthMethod = "none"
	defaultRemoteMode       RemoteMode       = RemoteModeUserspace
	defaultRemoteAuthMethod RemoteAuthMethod = RemoteAuthOAuth
)

// RemoteSettings is the persisted remote access config.
type RemoteSettings struct {
	Mode       RemoteMode       `json:"mode"`
	AuthMethod RemoteAuthMethod `json:"auth_method"`
	Secret     string           `json:"secret,omitempty"`
	Hostname   string           `json:"hostname,omitempty"`
}

// JRCConfig is the user runtime config persisted in ~/.jterrazz/config.json.
type JRCConfig struct {
	Remote    RemoteSettings `json:"remote"`
	UserEmail string         `json:"user_email,omitempty"`
	UserName  string         `json:"user_name,omitempty"`
}

// RemoteStatus summarizes current remote connectivity.
type RemoteStatus struct {
	Mode         RemoteMode
	BackendState string
	Hostname     string
	IP           string
	Connected    bool
	KeepAwake    bool
}

type tailscalePeer struct {
	HostName     string   `json:"HostName"`
	DNSName      string   `json:"DNSName"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	Online       bool     `json:"Online"`
	ExitNode     bool     `json:"ExitNode"`
	OS           string   `json:"OS"`
}

type tailscaleStatus struct {
	BackendState string         `json:"BackendState"`
	Self         *tailscalePeer `json:"Self"`
	Peer         map[string]*tailscalePeer `json:"Peer"`
}

// TailscalePeerInfo describes a peer on the tailnet.
type TailscalePeerInfo struct {
	Name   string
	IP     string
	Online bool
}

// TailscaleFullStatus is the enriched status returned by GetTailscaleFullStatus.
type TailscaleFullStatus struct {
	Connected    bool
	BackendState string
	Mode         RemoteMode
	IP           string
	ExitNode     string // hostname of active exit node, or ""
	Peers        []TailscalePeerInfo
}

func defaultRemoteSettings() RemoteSettings {
	return RemoteSettings{
		Mode:       defaultRemoteMode,
		AuthMethod: defaultRemoteAuthMethod,
	}
}

func normalizeRemoteSettings(s RemoteSettings) RemoteSettings {
	if s.Mode == "" || s.Mode == remoteModeSystemLegacy {
		s.Mode = defaultRemoteMode
	}
	if s.AuthMethod == "" || s.AuthMethod == remoteAuthNoneLegacy {
		s.AuthMethod = defaultRemoteAuthMethod
	}
	return s
}

// jterrazDir returns the root of the unified user data directory.
func jterrazDir() string {
	return filepath.Join(os.Getenv("HOME"), ".jterrazz")
}

func jrcPath() string {
	return filepath.Join(jterrazDir(), "config.json")
}

func userspaceDir() string {
	return filepath.Join(jterrazDir(), "tailscale")
}

func userspaceSocketPath() string {
	return filepath.Join(userspaceDir(), "tailscaled.sock")
}

func userspaceStatePath() string {
	return filepath.Join(userspaceDir(), "tailscaled.state")
}

func userspaceLogPath() string {
	return filepath.Join(userspaceDir(), "tailscaled.log")
}

func userspacePIDPath() string {
	return filepath.Join(userspaceDir(), "tailscaled.pid")
}

func keepAwakePIDPath() string {
	return filepath.Join(userspaceDir(), "caffeinate.pid")
}

// migrateOldConfig moves data from ~/.config/jterrazz/ to ~/.jterrazz/ if needed.
func migrateOldConfig() {
	oldDir := filepath.Join(os.Getenv("HOME"), ".config", "jterrazz")
	newDir := jterrazDir()

	// Migrate config.json
	oldJRC := filepath.Join(oldDir, "jrc.json")
	newJRC := jrcPath()
	if _, err := os.Stat(oldJRC); err == nil {
		if _, err := os.Stat(newJRC); os.IsNotExist(err) {
			_ = os.MkdirAll(newDir, 0700)
			if data, readErr := os.ReadFile(oldJRC); readErr == nil {
				_ = os.WriteFile(newJRC, data, 0600)
			}
		}
	}

	// Migrate tailscale state
	oldTS := filepath.Join(oldDir, "tailscale")
	newTS := userspaceDir()
	if info, err := os.Stat(oldTS); err == nil && info.IsDir() {
		if _, err := os.Stat(newTS); os.IsNotExist(err) {
			_ = os.MkdirAll(newDir, 0700)
			exec.Command("cp", "-r", oldTS, newTS).Run()
		}
	}
}

// LoadJRC loads ~/.jterrazz/config.json. Missing file returns defaults.
func LoadJRC() (JRCConfig, error) {
	migrateOldConfig()

	cfg := JRCConfig{Remote: defaultRemoteSettings()}

	data, err := os.ReadFile(jrcPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("failed to read config.json: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config.json: %w", err)
	}
	cfg.Remote = normalizeRemoteSettings(cfg.Remote)
	return cfg, nil
}

// SaveJRC writes ~/.jterrazz/config.json with strict file permissions.
func SaveJRC(cfg JRCConfig) error {
	cfg.Remote = normalizeRemoteSettings(cfg.Remote)
	if err := ValidateRemoteSettings(cfg.Remote); err != nil {
		return err
	}

	dir := filepath.Dir(jrcPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode config.json: %w", err)
	}
	out = append(out, '\n')

	tmpPath := jrcPath() + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0600); err != nil {
		return fmt.Errorf("failed to write temp config.json: %w", err)
	}
	if err := os.Rename(tmpPath, jrcPath()); err != nil {
		return fmt.Errorf("failed to save config.json: %w", err)
	}
	return nil
}

// LoadRemoteSettings loads current remote settings from config.json.
func LoadRemoteSettings() (RemoteSettings, error) {
	cfg, err := LoadJRC()
	if err != nil {
		return defaultRemoteSettings(), err
	}
	return cfg.Remote, nil
}

// SaveRemoteSettings saves remote settings into config.json.
func SaveRemoteSettings(s RemoteSettings) error {
	cfg, err := LoadJRC()
	if err != nil {
		return err
	}
	cfg.Remote = s
	return SaveJRC(cfg)
}

// HasRemoteSettings returns true when config.json exists and contains valid remote config.
func HasRemoteSettings() bool {
	if _, err := os.Stat(jrcPath()); err != nil {
		return false
	}
	s, err := LoadRemoteSettings()
	if err != nil {
		return false
	}
	return ValidateRemoteSettings(s) == nil
}

// ValidateRemoteSettings validates remote settings semantics.
func ValidateRemoteSettings(s RemoteSettings) error {
	s = normalizeRemoteSettings(s)

	switch s.Mode {
	case RemoteModeAuto, RemoteModeUserspace:
	default:
		return fmt.Errorf("invalid remote mode: %s", s.Mode)
	}

	switch s.AuthMethod {
	case RemoteAuthOAuth, RemoteAuthAuthKey:
	default:
		return fmt.Errorf("invalid auth_method: %s", s.AuthMethod)
	}

	if s.AuthMethod == RemoteAuthAuthKey && strings.TrimSpace(s.Secret) == "" {
		return fmt.Errorf("secret is required when auth_method is %s", s.AuthMethod)
	}

	return nil
}

// ConfigureRemoteInteractive prompts for remote settings and persists them in config.json.
func ConfigureRemoteInteractive() error {
	current, err := LoadRemoteSettings()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Remote access setup (writes ~/.jterrazz/config.json)")
	fmt.Println()

	current = normalizeRemoteSettings(current)
	fmt.Printf("Mode [auto/userspace] (%s): ", current.Mode)
	modeInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read mode: %w", err)
	}
	modeInput = strings.TrimSpace(strings.ToLower(modeInput))
	if modeInput != "" {
		current.Mode = RemoteMode(modeInput)
	}

	fmt.Printf("Auth method [oauth/authkey] (%s): ", current.AuthMethod)
	authInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read auth method: %w", err)
	}
	authInput = strings.TrimSpace(strings.ToLower(authInput))
	if authInput != "" {
		current.AuthMethod = RemoteAuthMethod(authInput)
	}

	if current.AuthMethod == RemoteAuthAuthKey {
		if current.Secret == "" {
			fmt.Print("Auth key: ")
		} else {
			fmt.Print("Auth key (leave empty to keep current): ")
		}
		secretInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read secret: %w", err)
		}
		secretInput = strings.TrimSpace(secretInput)
		if secretInput != "" {
			current.Secret = secretInput
		}
	} else {
		current.Secret = ""
	}

	fmt.Printf("Hostname (optional, current: %s): ", current.Hostname)
	hostInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read hostname: %w", err)
	}
	hostInput = strings.TrimSpace(hostInput)
	if hostInput != "" {
		current.Hostname = hostInput
	}

	if err := ValidateRemoteSettings(current); err != nil {
		return err
	}

	if err := SaveRemoteSettings(current); err != nil {
		return err
	}
	return nil
}

func tailscaleArgsForMode(mode RemoteMode, args ...string) []string {
	if mode == RemoteModeUserspace {
		return append([]string{"--socket", userspaceSocketPath()}, args...)
	}
	return args
}

func runTailscale(mode RemoteMode, args ...string) (string, error) {
	allArgs := tailscaleArgsForMode(mode, args...)
	cmd := exec.Command("tailscale", allArgs...)
	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	return output.String(), err
}

func formatCommandError(err error, output string) error {
	if err == nil {
		return nil
	}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Error:") {
			return fmt.Errorf("%s", strings.TrimSpace(strings.TrimPrefix(trimmed, "Error:")))
		}
	}
	return err
}

var nonDefaultFlagsError = regexp.MustCompile(`requires mentioning all\s+non-default flags`)

func shouldRetryWithSuggestedFlags(output string) bool {
	return nonDefaultFlagsError.MatchString(strings.ToLower(output))
}

func parseSuggestedUpFlags(output string) []string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "tailscale up ") {
			fields := strings.Fields(trimmed)
			if len(fields) > 2 {
				return fields[2:]
			}
		}
	}
	return nil
}

type cliFlag struct {
	Key    string
	Tokens []string
}

func parseCLIFLags(tokens []string) []cliFlag {
	var flags []cliFlag
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if !strings.HasPrefix(t, "--") {
			continue
		}
		if eq := strings.Index(t, "="); eq > 0 {
			flags = append(flags, cliFlag{
				Key:    t[:eq],
				Tokens: []string{t},
			})
			continue
		}
		if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "--") {
			flags = append(flags, cliFlag{
				Key:    t,
				Tokens: []string{t, tokens[i+1]},
			})
			i++
			continue
		}
		flags = append(flags, cliFlag{
			Key:    t,
			Tokens: []string{t},
		})
	}
	return flags
}

func mergeUpArgsWithSuggestedFlags(desiredUpArgs []string, suggestedFlags []string) []string {
	desiredFlags := desiredUpArgs
	if len(desiredFlags) > 0 && desiredFlags[0] == "up" {
		desiredFlags = desiredFlags[1:]
	}

	desired := parseCLIFLags(desiredFlags)
	suggested := parseCLIFLags(suggestedFlags)

	desiredByKey := make(map[string]cliFlag, len(desired))
	for _, f := range desired {
		desiredByKey[f.Key] = f
	}

	suggestedKeys := make(map[string]bool, len(suggested))
	usedDesired := make(map[string]bool, len(desired))

	var merged []string
	for _, f := range suggested {
		suggestedKeys[f.Key] = true
		if d, ok := desiredByKey[f.Key]; ok {
			merged = append(merged, d.Tokens...)
			usedDesired[d.Key] = true
			continue
		}
		merged = append(merged, f.Tokens...)
	}

	for _, f := range desired {
		if usedDesired[f.Key] {
			continue
		}
		if suggestedKeys[f.Key] {
			continue
		}
		merged = append(merged, f.Tokens...)
	}

	return append([]string{"up"}, merged...)
}

func getTailscaleStatus(mode RemoteMode) (tailscaleStatus, error) {
	var st tailscaleStatus
	cmd := exec.Command("tailscale", tailscaleArgsForMode(mode, "status", "--json")...)
	out, err := cmd.Output()
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(out, &st); err != nil {
		return st, fmt.Errorf("failed to parse tailscale status output: %w", err)
	}
	return st, nil
}

func ensureUserspaceDaemon() error {
	if _, err := getTailscaleStatus(RemoteModeUserspace); err == nil {
		return nil
	}

	if !CommandExists("tailscaled") {
		return fmt.Errorf("tailscaled is required for userspace mode")
	}

	if err := os.MkdirAll(userspaceDir(), 0700); err != nil {
		return fmt.Errorf("failed to create userspace directory: %w", err)
	}

	logFile, err := os.OpenFile(userspaceLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open tailscaled log file: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(
		"tailscaled",
		"--tun=userspace-networking",
		"--state="+userspaceStatePath(),
		"--socket="+userspaceSocketPath(),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start userspace tailscaled: %w", err)
	}

	_ = os.WriteFile(userspacePIDPath(), []byte(strconv.Itoa(cmd.Process.Pid)), 0600)
	_ = cmd.Process.Release()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := getTailscaleStatus(RemoteModeUserspace); err == nil {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("userspace tailscaled did not become ready (check %s)", userspaceLogPath())
}

func stopUserspaceDaemon() {
	data, err := os.ReadFile(userspacePIDPath())
	if err != nil {
		return
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Signal(syscall.SIGTERM)
	_ = os.Remove(userspacePIDPath())
}

func pidFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid in %s", path)
	}
	return pid, nil
}

func processRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func isKeepAwakeRunning() bool {
	pid, err := pidFromFile(keepAwakePIDPath())
	if err != nil {
		return false
	}
	if processRunning(pid) {
		return true
	}
	_ = os.Remove(keepAwakePIDPath())
	return false
}

func ensureKeepAwake() error {
	if !CommandExists("caffeinate") {
		return nil
	}
	if isKeepAwakeRunning() {
		return nil
	}
	if err := os.MkdirAll(userspaceDir(), 0700); err != nil {
		return fmt.Errorf("failed to create userspace directory: %w", err)
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", os.DevNull, err)
	}
	defer devNull.Close()

	cmd := exec.Command("caffeinate", "-i")
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start caffeinate: %w", err)
	}

	if err := os.WriteFile(keepAwakePIDPath(), []byte(strconv.Itoa(cmd.Process.Pid)), 0600); err != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("failed to persist caffeinate pid: %w", err)
	}

	_ = cmd.Process.Release()
	return nil
}

func stopKeepAwake() {
	pid, err := pidFromFile(keepAwakePIDPath())
	if err == nil && pid > 0 {
		if process, findErr := os.FindProcess(pid); findErr == nil {
			_ = process.Signal(syscall.SIGTERM)
		}
	}
	_ = os.Remove(keepAwakePIDPath())
}

func buildUpArgs(settings RemoteSettings) []string {
	args := []string{"up", "--ssh"}
	if settings.Hostname != "" {
		args = append(args, "--hostname", settings.Hostname)
	}
	if settings.AuthMethod == RemoteAuthAuthKey {
		args = append(args, "--auth-key", settings.Secret)
	}
	return args
}

func remoteUpWithMode(mode RemoteMode, settings RemoteSettings) error {
	if !CommandExists("tailscale") {
		return fmt.Errorf("tailscale CLI not found")
	}

	if mode == RemoteModeUserspace {
		if err := ensureUserspaceDaemon(); err != nil {
			return err
		}
	}

	upArgs := buildUpArgs(settings)
	output, err := runTailscale(mode, upArgs...)
	if err == nil {
		if mode == RemoteModeUserspace {
			_ = ensureKeepAwake()
		}
		return nil
	}

	// Keep existing non-default preferences by retrying with the flags suggested by tailscale.
	if shouldRetryWithSuggestedFlags(output) {
		if suggested := parseSuggestedUpFlags(output); len(suggested) > 0 {
			retryArgs := mergeUpArgsWithSuggestedFlags(upArgs, suggested)
			retryOutput, retryErr := runTailscale(mode, retryArgs...)
			if retryErr == nil {
				if mode == RemoteModeUserspace {
					_ = ensureKeepAwake()
				}
				return nil
			}
			return formatCommandError(retryErr, retryOutput)
		}
	}

	return formatCommandError(err, output)
}

func detectActiveMode() RemoteMode {
	if _, err := getTailscaleStatus(RemoteModeUserspace); err == nil {
		return RemoteModeUserspace
	}
	return RemoteModeAuto
}

// RemoteUp connects remote access using configured settings.
// Returns the mode that was actually used.
func RemoteUp(settings RemoteSettings) (RemoteMode, error) {
	settings = normalizeRemoteSettings(settings)
	if err := ValidateRemoteSettings(settings); err != nil {
		return "", err
	}

	switch settings.Mode {
	case RemoteModeUserspace:
		if err := remoteUpWithMode(RemoteModeUserspace, settings); err != nil {
			return "", err
		}
		return RemoteModeUserspace, nil
	case RemoteModeAuto:
		if err := remoteUpWithMode(RemoteModeUserspace, settings); err != nil {
			return "", err
		}
		return RemoteModeUserspace, nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", settings.Mode)
	}
}

// RemoteDown disconnects remote access.
// Returns the mode that was actually used.
func RemoteDown(settings RemoteSettings) (RemoteMode, error) {
	settings = normalizeRemoteSettings(settings)
	mode := settings.Mode
	if mode == RemoteModeAuto {
		mode = detectActiveMode()
	}

	if mode == RemoteModeUserspace {
		downOutput, downErr := runTailscale(RemoteModeUserspace, "down")
		stopKeepAwake()
		stopUserspaceDaemon()
		if downErr != nil {
			return mode, formatCommandError(downErr, downOutput)
		}
		return mode, nil
	}

	return mode, fmt.Errorf("unsupported mode: %s", mode)
}

// RemoteStatusInfo returns current remote access state.
func RemoteStatusInfo(settings RemoteSettings) (RemoteStatus, error) {
	settings = normalizeRemoteSettings(settings)
	mode := settings.Mode
	if mode == RemoteModeAuto {
		mode = detectActiveMode()
	}

	st, err := getTailscaleStatus(mode)
	if err != nil {
		return RemoteStatus{Mode: mode, Connected: false}, err
	}

	result := RemoteStatus{
		Mode:         mode,
		BackendState: st.BackendState,
		Connected:    st.BackendState == "Running",
		KeepAwake:    isKeepAwakeRunning(),
	}

	if st.Self != nil {
		result.Hostname = st.Self.HostName
		if result.Hostname == "" {
			result.Hostname = st.Self.DNSName
		}
		if len(st.Self.TailscaleIPs) > 0 {
			result.IP = st.Self.TailscaleIPs[0]
		}
	}

	return result, nil
}

// GetTailscaleFullStatus returns enriched Tailscale status including peers.
// It tries configured remote settings first, then falls back to a direct CLI call.
func GetTailscaleFullStatus() (TailscaleFullStatus, error) {
	var mode RemoteMode
	var st tailscaleStatus

	if settings, err := LoadRemoteSettings(); err == nil && ValidateRemoteSettings(settings) == nil {
		settings = normalizeRemoteSettings(settings)
		mode = settings.Mode
		if mode == RemoteModeAuto {
			mode = detectActiveMode()
		}
		st, err = getTailscaleStatus(mode)
		if err != nil {
			return TailscaleFullStatus{}, err
		}
	} else {
		// No configured settings — try direct CLI
		var err error
		st, err = getTailscaleStatus(RemoteModeAuto)
		if err != nil {
			return TailscaleFullStatus{}, err
		}
		mode = RemoteModeAuto
	}

	result := TailscaleFullStatus{
		BackendState: st.BackendState,
		Connected:    st.BackendState == "Running",
		Mode:         mode,
	}

	if st.Self != nil {
		if len(st.Self.TailscaleIPs) > 0 {
			result.IP = st.Self.TailscaleIPs[0]
		}
	}

	// Find active exit node and collect peers
	for _, peer := range st.Peer {
		if peer == nil {
			continue
		}
		ip := ""
		if len(peer.TailscaleIPs) > 0 {
			ip = peer.TailscaleIPs[0]
		}
		name := peer.HostName
		if name == "" || name == "localhost" {
			// Use DNSName, strip trailing dot and tailnet suffix for readability
			name = strings.TrimSuffix(peer.DNSName, ".")
			if parts := strings.SplitN(name, ".", 2); len(parts) > 0 {
				name = parts[0]
			}
		}
		if name == "" {
			name = "unknown"
		}

		if peer.ExitNode {
			result.ExitNode = name
		}

		result.Peers = append(result.Peers, TailscalePeerInfo{
			Name:   name,
			IP:     ip,
			Online: peer.Online,
		})
	}

	// Sort peers: online first, then alphabetical
	sort.Slice(result.Peers, func(i, j int) bool {
		if result.Peers[i].Online != result.Peers[j].Online {
			return result.Peers[i].Online
		}
		return result.Peers[i].Name < result.Peers[j].Name
	})

	return result, nil
}
