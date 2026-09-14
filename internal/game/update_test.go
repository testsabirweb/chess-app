package game

import "testing"

func TestUpdateBridge(t *testing.T) {
	SetUpdateReady(true)
	if !UpdateReady() {
		t.Fatal("expected update ready")
	}
	RequestInstallUpdate()
	if !ConsumeInstallRequest() {
		t.Fatal("expected install request")
	}
	if ConsumeInstallRequest() {
		t.Fatal("install request should be consumed once")
	}
	SetUpdateReady(false)
	if UpdateReady() {
		t.Fatal("expected update not ready")
	}
}
