package game

var (
	updateReady       bool
	installRequested  bool
)

// SetUpdateReady is called from Android when a newer APK has been downloaded.
func SetUpdateReady(ready bool) { updateReady = ready }

// UpdateReady reports whether an installable update is waiting.
func UpdateReady() bool { return updateReady }

// RequestInstallUpdate asks the Android shell to launch the system installer.
func RequestInstallUpdate() { installRequested = true }

// ConsumeInstallRequest returns true once per install request for MainActivity
// to poll on the main thread.
func ConsumeInstallRequest() bool {
	if !installRequested {
		return false
	}
	installRequested = false
	return true
}
