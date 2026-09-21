package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const scheduleName = "gofact-sync"

// runSchedule installe ou retire l'exécution périodique de « gofact sync ».
func runSchedule(argv []string) {
	fs := flag.NewFlagSet("gofact schedule", flag.ExitOnError)
	every := fs.Duration("every", time.Hour, "intervalle entre deux synchronisations (minimum 5m)")
	apply := fs.Bool("yes", false, "applique (défaut : affiche seulement ce qui serait fait)")
	remove := fs.Bool("remove", false, "retire la planification")
	_ = fs.Parse(argv)

	if *every < 5*time.Minute {
		fail(fmt.Errorf("-every doit valoir au moins 5m : la plateforme n'a pas à être interrogée plus souvent"))
	}
	exe, err := os.Executable()
	if err != nil {
		fail(err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		fail(err)
	}

	var plan schedulePlan
	switch runtime.GOOS {
	case "linux":
		plan, err = systemdPlan(exe, *every, *remove)
	case "darwin":
		plan, err = launchdPlan(exe, *every, *remove)
	case "windows":
		plan = schtasksPlan(exe, *every, *remove)
	default:
		err = fmt.Errorf("planification non prise en charge sur %s : lancer « %s sync » depuis cron", runtime.GOOS, exe)
	}
	if err != nil {
		fail(err)
	}

	for path, content := range plan.files {
		if content == "" {
			fmt.Printf("→ supprimer %s\n", path)
		} else {
			fmt.Printf("→ écrire %s :\n%s\n", path, content)
		}
	}
	for _, c := range plan.commands {
		fmt.Printf("→ %q\n", c)
	}
	if !*apply {
		fmt.Println("\nRien n'a été modifié. Pour appliquer : gofact schedule -yes")
		return
	}

	for path, content := range plan.files {
		if content == "" {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fail(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fail(err)
		}
	}
	for _, c := range plan.commands {
		if out, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil && !plan.tolerant[c[0]+" "+c[1]] {
			fail(fmt.Errorf("%v : %s", c, out))
		}
	}
	if *remove {
		fmt.Println("✓ Planification retirée")
	} else {
		fmt.Printf("✓ gofact sync tournera toutes les %s\n", *every)
	}
}

type schedulePlan struct {
	files    map[string]string // contenu vide : fichier à supprimer
	commands [][]string
	tolerant map[string]bool // commandes dont l'échec est attendu (retrait d'une planification absente)
}

func systemdPlan(exe string, every time.Duration, remove bool) (schedulePlan, error) {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return schedulePlan{}, fmt.Errorf("systemd introuvable : ajouter à la crontab la ligne\n  */%d * * * * %q sync -q",
			max(int(every.Minutes()), 5), exe)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return schedulePlan{}, err
	}
	dir := filepath.Join(home, ".config", "systemd", "user")
	service := filepath.Join(dir, scheduleName+".service")
	timer := filepath.Join(dir, scheduleName+".timer")
	if remove {
		return schedulePlan{
			files:    map[string]string{service: "", timer: ""},
			commands: [][]string{{"systemctl", "--user", "disable", "--now", scheduleName + ".timer"}, {"systemctl", "--user", "daemon-reload"}},
			tolerant: map[string]bool{"systemctl --user": true},
		}, nil
	}
	return schedulePlan{
		files: map[string]string{
			service: fmt.Sprintf("[Unit]\nDescription=gofact : factures reçues et export comptable\n\n"+
				"[Service]\nType=oneshot\nExecStart=%q sync\n", exe),
			timer: fmt.Sprintf("[Unit]\nDescription=gofact sync périodique\n\n"+
				"[Timer]\nOnBootSec=5min\nOnUnitActiveSec=%s\n\n[Install]\nWantedBy=timers.target\n",
				strconv.Itoa(int(every.Seconds()))+"s"),
		},
		commands: [][]string{{"systemctl", "--user", "daemon-reload"}, {"systemctl", "--user", "enable", "--now", scheduleName + ".timer"}},
	}, nil
}

func launchdPlan(exe string, every time.Duration, remove bool) (schedulePlan, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return schedulePlan{}, err
	}
	label := "com.kolapsis." + scheduleName
	plist := filepath.Join(home, "Library", "LaunchAgents", label+".plist")
	domain := "gui/" + strconv.Itoa(os.Getuid())
	unload := []string{"launchctl", "bootout", domain + "/" + label}
	if remove {
		return schedulePlan{files: map[string]string{plist: ""}, commands: [][]string{unload},
			tolerant: map[string]bool{"launchctl bootout": true}}, nil
	}
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key><array><string>%s</string><string>sync</string></array>
  <key>StartInterval</key><integer>%d</integer>
  <key>RunAtLoad</key><true/>
</dict>
</plist>
`, label, exe, int(every.Seconds()))
	return schedulePlan{
		files:    map[string]string{plist: content},
		commands: [][]string{unload, {"launchctl", "bootstrap", domain, plist}},
		tolerant: map[string]bool{"launchctl bootout": true},
	}, nil
}

func schtasksPlan(exe string, every time.Duration, remove bool) schedulePlan {
	if remove {
		return schedulePlan{commands: [][]string{{"schtasks", "/Delete", "/F", "/TN", scheduleName}}}
	}
	unit, n := "MINUTE", int(every.Minutes())
	if n%60 == 0 {
		unit, n = "HOURLY", min(n/60, 23)
	}
	return schedulePlan{commands: [][]string{{"schtasks", "/Create", "/F", "/SC", unit,
		"/MO", strconv.Itoa(min(n, 1439)), "/TN", scheduleName, "/TR", fmt.Sprintf(`"%s" sync`, exe)}}}
}
