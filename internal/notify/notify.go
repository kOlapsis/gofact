// Package notify affiche une notification sur le bureau de l'utilisateur.
package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Send affiche une notification ; sans notificateur disponible, le message part sur stderr.
func Send(title, body string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd":
		if path, err := exec.LookPath("notify-send"); err == nil {
			cmd = exec.Command(path, "--app-name=gofact", title, body)
		}
	case "darwin":
		cmd = exec.Command("osascript", "-e",
			fmt.Sprintf("display notification %s with title %s", appleString(body), appleString(title)))
	case "windows":
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", windowsToast(title, body))
	}
	if cmd == nil || cmd.Run() != nil {
		fmt.Fprintf(os.Stderr, "%s — %s\n", title, body)
	}
}

func appleString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Windows n'affiche un toast que pour une application inscrite : on emprunte l'identité de PowerShell.
func windowsToast(title, body string) string {
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "''") }
	return `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null;` +
		`$t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02);` +
		`$x = $t.GetElementsByTagName('text');` +
		`$x.Item(0).AppendChild($t.CreateTextNode('` + esc(title) + `')) > $null;` +
		`$x.Item(1).AppendChild($t.CreateTextNode('` + esc(body) + `')) > $null;` +
		`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show([Windows.UI.Notifications.ToastNotification]::new($t))`
}
