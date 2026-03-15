package notify

import (
	"fmt"
	"os/exec"
)

// Send displays a macOS notification using osascript and plays a sound.
func Send(title, message string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, message, title)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return err
	}
	return exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
}
