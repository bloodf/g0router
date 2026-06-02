package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type installOptions struct {
	User       bool
	Root       string
	HomeDir    string
	Executable string
	RunCommand func(string, ...string) error
	Out        io.Writer
}

// NewInstallCommand builds the systemd install command.
func NewInstallCommand() *cobra.Command {
	var user bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install g0router as a systemd service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(newInstallOptions(cmd.OutOrStdout(), user))
		},
	}
	cmd.Flags().BoolVar(&user, "user", false, "install as a user systemd service")
	return cmd
}

func newUninstallCommand() *cobra.Command {
	var user bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the g0router systemd service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(newInstallOptions(cmd.OutOrStdout(), user))
		},
	}
	cmd.Flags().BoolVar(&user, "user", false, "uninstall the user systemd service")
	return cmd
}

func newInstallOptions(out io.Writer, user bool) installOptions {
	home, _ := os.UserHomeDir()
	exe, _ := os.Executable()
	return installOptions{
		User:       user,
		Root:       string(filepath.Separator),
		HomeDir:    home,
		Executable: exe,
		RunCommand: runCommand,
		Out:        out,
	}
}

func runInstall(options installOptions) error {
	options = normalizeInstallOptions(options)
	paths := installPaths(options)

	if !options.User {
		if err := ensureSystemUser(options.RunCommand); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(paths.Binary), 0o755); err != nil {
		return fmt.Errorf("create binary dir: %w", err)
	}
	if err := copyFile(options.Executable, paths.Binary, 0o755); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if !options.User {
		if err := options.RunCommand("chown", "g0router:g0router", paths.DataDir); err != nil {
			return fmt.Errorf("set data dir owner: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(paths.Service), 0o755); err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}

	serviceBinary := paths.Binary
	serviceDataDir := paths.DataDir
	if !options.User {
		serviceBinary = "/usr/local/bin/g0router"
		serviceDataDir = "/var/lib/g0router"
	}
	service, err := serviceTemplate(serviceBinary, serviceDataDir)
	if err != nil {
		return err
	}
	if err := os.WriteFile(paths.Service, []byte(service), 0o644); err != nil {
		return fmt.Errorf("write service: %w", err)
	}
	if paths.Defaults != "" {
		if err := os.MkdirAll(filepath.Dir(paths.Defaults), 0o755); err != nil {
			return fmt.Errorf("create defaults dir: %w", err)
		}
		defaults, err := readDeployTemplate("g0router.default")
		if err != nil {
			return fmt.Errorf("read default template: %w", err)
		}
		if err := os.WriteFile(paths.Defaults, defaults, 0o644); err != nil {
			return fmt.Errorf("write defaults: %w", err)
		}
	}

	if err := options.RunCommand("systemctl", paths.SystemctlArgs("daemon-reload")...); err != nil {
		return fmt.Errorf("reload systemd: %w", err)
	}
	if err := options.RunCommand("systemctl", paths.SystemctlArgs("enable", "--now", "g0router")...); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	if options.User {
		fmt.Fprintf(options.Out, "installed user service: %s\n", displayPath(paths.Service, options))
		return nil
	}
	fmt.Fprintf(options.Out, "installed system service: %s\n", displayPath(paths.Service, options))
	return nil
}

func runUninstall(options installOptions) error {
	options = normalizeInstallOptions(options)
	paths := installPaths(options)

	if err := options.RunCommand("systemctl", paths.SystemctlArgs("disable", "--now", "g0router")...); err != nil {
		return fmt.Errorf("disable service: %w", err)
	}
	for _, path := range []string{paths.Service, paths.Defaults, paths.Binary} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	if err := options.RunCommand("systemctl", paths.SystemctlArgs("daemon-reload")...); err != nil {
		return fmt.Errorf("reload systemd: %w", err)
	}
	fmt.Fprintf(options.Out, "removed service; kept data in %s\n", displayPath(paths.DataDir, options))
	return nil
}

type resolvedInstallPaths struct {
	Binary   string
	Service  string
	Defaults string
	DataDir  string
	User     bool
}

func (p resolvedInstallPaths) SystemctlArgs(args ...string) []string {
	if !p.User {
		return args
	}
	return append([]string{"--user"}, args...)
}

func installPaths(options installOptions) resolvedInstallPaths {
	if options.User {
		return resolvedInstallPaths{
			Binary:  filepath.Join(options.HomeDir, ".local", "bin", "g0router"),
			Service: filepath.Join(options.HomeDir, ".config", "systemd", "user", "g0router.service"),
			DataDir: filepath.Join(options.HomeDir, ".g0router"),
			User:    true,
		}
	}
	return resolvedInstallPaths{
		Binary:   rooted(options.Root, "/usr/local/bin/g0router"),
		Service:  rooted(options.Root, "/etc/systemd/system/g0router.service"),
		Defaults: rooted(options.Root, "/etc/default/g0router"),
		DataDir:  rooted(options.Root, "/var/lib/g0router"),
	}
}

func normalizeInstallOptions(options installOptions) installOptions {
	if options.Root == "" {
		options.Root = string(filepath.Separator)
	}
	if options.HomeDir == "" {
		options.HomeDir, _ = os.UserHomeDir()
	}
	if options.Executable == "" {
		options.Executable, _ = os.Executable()
	}
	if options.RunCommand == nil {
		options.RunCommand = runCommand
	}
	if options.Out == nil {
		options.Out = io.Discard
	}
	return options
}

func rooted(root, absolute string) string {
	if root == string(filepath.Separator) {
		return absolute
	}
	return filepath.Join(root, strings.TrimPrefix(absolute, string(filepath.Separator)))
}

func serviceTemplate(binary, dataDir string) (string, error) {
	content, err := readDeployTemplate("g0router.service")
	if err != nil {
		return "", fmt.Errorf("read service template: %w", err)
	}
	service := strings.ReplaceAll(string(content), "/usr/local/bin/g0router", binary)
	service = strings.ReplaceAll(service, "/var/lib/g0router", dataDir)
	return service, nil
}

func ensureSystemUser(run func(string, ...string) error) error {
	if err := run("id", "-u", "g0router"); err == nil {
		return nil
	}
	if err := run("useradd", "--system", "--no-create-home", "--shell", "/usr/sbin/nologin", "g0router"); err != nil {
		return fmt.Errorf("create g0router user: %w", err)
	}
	return nil
}

func readDeployTemplate(name string) ([]byte, error) {
	for _, candidate := range []string{
		filepath.Join("deploy", name),
		filepath.Join("..", "..", "deploy", name),
	} {
		content, err := os.ReadFile(candidate)
		if err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("deploy template %s not found", name)
}

func copyFile(src, dst string, mode os.FileMode) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s: %w", name, strings.Join(args, " "), strings.TrimSpace(string(output)), err)
	}
	return nil
}

func displayPath(path string, options installOptions) string {
	if options.Root == string(filepath.Separator) || options.User {
		return path
	}
	return string(filepath.Separator) + strings.TrimPrefix(strings.TrimPrefix(path, options.Root), string(filepath.Separator))
}
