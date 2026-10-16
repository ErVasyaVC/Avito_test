package domain

import "testing"

func TestOrder_CanBeCancelledByUser(t *testing.T) {
	tests := []struct {
		status OrderStatus
		want   bool
	}{
		{OrderStatusCreated, true},
		{OrderStatusAccepted, true},
		{OrderStatusCooking, false},
		{OrderStatusReady, false},
		{OrderStatusDelivered, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			o := Order{Status: tt.status}
			if got := o.CanBeCancelledByUser(); got != tt.want {
				t.Errorf("CanBeCancelledByUser() with status %q = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestOrder_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		current OrderStatus
		next    OrderStatus
		want    bool
	}{
		{"created to cooking", OrderStatusCreated, OrderStatusCooking, true},
		{"accepted to cooking", OrderStatusAccepted, OrderStatusCooking, true},
		{"cooking to ready", OrderStatusCooking, OrderStatusReady, true},
		{"cooking to cancelled_by_venue", OrderStatusCooking, OrderStatusCancelledByVenue, true},
		{"cooking to delivered is invalid", OrderStatusCooking, OrderStatusDelivered, false},
		{"created to ready skips cooking, invalid", OrderStatusCreated, OrderStatusReady, false},
		{"ready to cooking is invalid, no going back", OrderStatusReady, OrderStatusCooking, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Order{Status: tt.current}
			if got := o.CanTransitionTo(tt.next); got != tt.want {
				t.Errorf("CanTransitionTo(%q) with status %q = %v, want %v", tt.next, tt.current, got, tt.want)
			}
		})
	}
}

func TestOrder_CanAdvanceDelivery(t *testing.T) {
	tests := []struct {
		name    string
		current OrderStatus
		next    OrderStatus
		want    bool
	}{
		{"ready to delivering", OrderStatusReady, OrderStatusDelivering, true},
		{"delivering to delivered", OrderStatusDelivering, OrderStatusDelivered, true},
		{"delivered to delivering is invalid, terminal", OrderStatusDelivered, OrderStatusDelivering, false},
		{"cooking to delivering skips ready, invalid", OrderStatusCooking, OrderStatusDelivering, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Order{Status: tt.current}
			if got := o.CanAdvanceDelivery(tt.next); got != tt.want {
				t.Errorf("CanAdvanceDelivery(%q) with status %q = %v, want %v", tt.next, tt.current, got, tt.want)
			}
		})
	}
}
