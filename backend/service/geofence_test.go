package service

import (
	"testing"
	"time"
)

func TestGeofenceInside_hysteresis(t *testing.T) {
	radius := 100.0
	buffer := geofenceExitBufferMeters()

	// Вход: строго по радиусу.
	if !geofenceInside(radius, radius, false) {
		t.Fatal("expected inside at radius boundary for enter")
	}
	if geofenceInside(radius+1, radius, false) {
		t.Fatal("expected outside just beyond radius for enter")
	}

	// Активный визит: расширенная зона.
	edge := radius + buffer
	if !geofenceInside(edge, radius, true) {
		t.Fatalf("expected inside at radius+buffer (%.1f)", edge)
	}
	if geofenceInside(edge+1, radius, true) {
		t.Fatalf("expected outside beyond radius+buffer (%.1f)", edge+1)
	}
}

func TestGeofencePendingState_enterGrace(t *testing.T) {
	st := &geofencePendingState{}
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	grace := 30

	if st.pendingEnterElapsed(now, grace) {
		t.Fatal("enter should not be confirmed immediately")
	}

	st.markPendingEnter(now)
	if st.pendingEnterElapsed(now.Add(29*time.Second), grace) {
		t.Fatal("enter should not be confirmed before grace")
	}
	if !st.pendingEnterElapsed(now.Add(30*time.Second), grace) {
		t.Fatal("enter should be confirmed after grace")
	}

	st.clearPendingEnter()
	if st.pendingEnterElapsed(now.Add(60*time.Second), grace) {
		t.Fatal("cleared enter should not be confirmed")
	}
}

func TestGeofencePendingState_exitGrace(t *testing.T) {
	st := &geofencePendingState{}
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	grace := 90

	st.markPendingExit(now)
	if !st.pendingExitElapsed(now.Add(90*time.Second), grace) {
		t.Fatal("exit should be confirmed after grace")
	}
	st.clearPendingExit()
	if st.pendingExitElapsed(now.Add(120*time.Second), grace) {
		t.Fatal("cleared exit should not be confirmed")
	}
}
