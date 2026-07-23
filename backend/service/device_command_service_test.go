package service

import (
	"errors"
	"testing"
)

func TestEnqueueCommand_invalidType(t *testing.T) {
	svc := &DeviceCommandService{}
	_, err := svc.EnqueueCommand(1, "not_a_real_command", nil)
	if !errors.Is(err, ErrDeviceCommandInvalidType) {
		t.Fatalf("err=%v", err)
	}
}

func TestEnqueueCommand_allowedTypes(t *testing.T) {
	allowed := []string{
		"location_request",
		"health_check",
		"config_update",
		"app_update",
	}
	for _, typ := range allowed {
		if _, ok := allowedDeviceCommandTypes[typ]; !ok {
			t.Fatalf("missing allowed type %s", typ)
		}
	}
}
