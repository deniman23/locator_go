package service

import "testing"

func TestBuildConfigUpdatePayload(t *testing.T) {
	paused := true
	loc := int64(120)
	pin := "5678"
	payload, err := BuildConfigUpdatePayload(5, DeviceConfigUpdateInput{
		TrackingPaused:          &paused,
		LocationIntervalSeconds: &loc,
		AdminPin:                &pin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload["tracking_paused"] != true {
		t.Fatalf("tracking_paused: %v", payload["tracking_paused"])
	}
	if payload["location_interval_seconds"] != int64(120) {
		t.Fatalf("location_interval_seconds: %v", payload["location_interval_seconds"])
	}
	if payload["admin_pin"] != "5678" {
		t.Fatalf("admin_pin: %v", payload["admin_pin"])
	}
}

func TestBuildConfigUpdatePayloadEmpty(t *testing.T) {
	_, err := BuildConfigUpdatePayload(1, DeviceConfigUpdateInput{})
	if err != ErrDeviceConfigUpdateEmpty {
		t.Fatalf("expected empty error, got %v", err)
	}
}

func TestBuildConfigUpdatePayloadInvalidPin(t *testing.T) {
	bad := "12"
	_, err := BuildConfigUpdatePayload(1, DeviceConfigUpdateInput{AdminPin: &bad})
	if err != ErrDeviceConfigUpdateInvalid {
		t.Fatalf("expected invalid error, got %v", err)
	}
}
